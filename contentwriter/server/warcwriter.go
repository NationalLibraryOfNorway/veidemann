/*
 * Copyright 2021 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package server

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/database"
	"github.com/google/uuid"
	"github.com/nlnwa/gowarc"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// now is a function so that tests can override the clock.
var now = time.Now

const warcFileScheme = "warcfile"

type warcWriter struct {
	opts             WriterOptions
	collectionConfig *configV1.ConfigObject
	subCollection    configV1.Collection_SubCollectionType
	filePrefix       string
	fileWriter       *gowarc.WarcFileWriter
	configAdapter    database.ConfigAdapter
	contentAdapter   database.ContentAdapter
	timer            *time.Timer
	done             chan interface{}
	lock             sync.Mutex
	revisitProfile   string
}

type WriterOptions struct {
	WarcDir            string
	WarcVersion        *gowarc.WarcVersion
	WarcWriterPoolSize int
	FlushRecord        bool
}

func newWarcWriter(
	settings WriterOptions,
	config database.ConfigAdapter,
	content database.ContentAdapter,
	collection *configV1.ConfigObject,
	recordMeta *contentwriterV1.WriteRequestMeta_RecordMeta) *warcWriter {

	collectionConfig := collection.GetCollection()
	ww := &warcWriter{
		opts:             settings,
		configAdapter:    config,
		contentAdapter:   content,
		collectionConfig: collection,
		subCollection:    recordMeta.GetSubCollection(),
		filePrefix:       createFilePrefix(collection.GetMeta().GetName(), recordMeta.GetSubCollection(), now(), collection.GetCollection().GetCollectionDedupPolicy()),
	}
	switch settings.WarcVersion {
	case gowarc.V1_1:
		ww.revisitProfile = gowarc.ProfileIdenticalPayloadDigestV1_1
	case gowarc.V1_0:
		ww.revisitProfile = gowarc.ProfileIdenticalPayloadDigestV1_0
	default:
		panic(fmt.Sprintf("unsupported WARC version: '%s'", settings.WarcVersion))
	}
	ww.initFileWriter()

	rotationPolicy := collectionConfig.GetFileRotationPolicy()
	dedupPolicy := collectionConfig.GetCollectionDedupPolicy()
	if dedupPolicy != configV1.Collection_NONE && dedupPolicy < rotationPolicy {
		rotationPolicy = dedupPolicy
	}
	if d, ok := timeToNextRotation(now(), rotationPolicy); ok {
		ww.timer = time.NewTimer(d)
		ww.done = make(chan interface{})
		go func() {
			for ww.waitForTimer(rotationPolicy) {
				// wait
			}
		}()
	}

	return ww
}

func (ww *warcWriter) CollectionName() string {
	return ww.filePrefix[:len(ww.filePrefix)-1]
}

func (ww *warcWriter) Write(meta *contentwriterV1.WriteRequestMeta, record ...gowarc.WarcRecord) (*contentwriterV1.WriteReply, error) {
	ww.lock.Lock()
	defer ww.lock.Unlock()

	dedupPolicy := ww.collectionConfig.GetCollection().GetCollectionDedupPolicy()
	ttl := timeToLive(dedupPolicy)
	collection := ww.filePrefix[:len(ww.filePrefix)-1]
	revisitKeys := make([]string, len(record))

	for i, r := range record {
		defer func() { _ = r.Close() }()

		if r.Type() != gowarc.Response && r.Type() != gowarc.Resource {
			record[i], revisitKeys[i] = r, ""
			continue
		}

		digest := r.WarcHeader().Get(gowarc.WarcPayloadDigest)
		if digest == "" {
			digest = r.WarcHeader().Get(gowarc.WarcBlockDigest)
		}
		if digest == "" {
			record[i], revisitKeys[i] = r, ""
			continue
		}

		crawledContent, err := ww.contentAdapter.HasCrawledContent(context.TODO(), collection, digest)
		if err != nil {
			log.Warn().Err(err).
				Str("collection", collection).
				Str("digest", digest).
				Msg("Error checking for crawled content")
			record[i], revisitKeys[i] = r, digest
			continue
		}
		if crawledContent == nil {
			log.Debug().
				Str("collection", collection).
				Str("digest", digest).
				Msg("No crawled content found")
			record[i], revisitKeys[i] = r, digest
			continue
		}
		revisitRecord, err := ww.toRevisitRecord(int32(i), r, meta, crawledContent)
		if err != nil {
			log.Err(err).Msg("Could not create revisit record")
			record[i], revisitKeys[i] = r, digest
			continue
		}
		record[i], revisitKeys[i] = revisitRecord, ""
	}

	results := ww.fileWriter.Write(record...)

	var err error
	recordMeta := map[int32]*contentwriterV1.WriteResponseMeta_RecordMeta{}

	for i, res := range results {
		recNum := int32(i)
		rec := record[i]
		revisitKey := revisitKeys[i]

		if res.Err != nil {
			log.Err(res.Err).Msgf("Error writing record: %s", rec)
		}
		// If writing records failed. Set err to the first error
		if err == nil && res.Err != nil {
			err = res.Err
		}

		// Get WarcRecordId from header: '<urn:uuid:xxxxxxxx-xxx-xxx-xxx-xxxxxxxxx>'
		headerWarcRecordId := rec.WarcHeader().Get(gowarc.WarcRecordID)
		// Trim '<' and '>'
		warcRecordId := strings.TrimSuffix(strings.TrimPrefix(headerWarcRecordId, "<"), ">")

		// Parse as 'urn:uuid:xxxxxxxx-xxx-xxx-xxx-xxxxxxxxx'
		warcId, parseErr := uuid.Parse(warcRecordId)
		if parseErr != nil {
			log.Err(parseErr).Str("warcRecordId", warcRecordId).Msgf("failed to parse %s as UUID at %s:%d", gowarc.WarcRecordID, res.FileName, res.FileOffset)
		}

		log.Debug().Msgf("Written record num %d: WarcId: %s, StorageRef: %s:%d", recNum, warcId.String(), res.FileName, res.FileOffset)

		if res.Err == nil && parseErr == nil && revisitKey != "" {
			writeErr := func() error {
				t, err := time.Parse(time.RFC3339, rec.WarcHeader().Get(gowarc.WarcDate))
				if err != nil {
					return err
				}
				cr := &contentwriterV1.CrawledContent{
					Digest:    revisitKey,
					WarcId:    warcId.String(),
					TargetUri: meta.GetTargetUri(),
					Date:      timestamppb.New(t),
				}
				return ww.contentAdapter.WriteCrawledContent(context.TODO(), collection, ttl, cr)
			}()
			if writeErr != nil {
				log.Warn().Err(writeErr).
					Str("collection", collection).
					Str("digest", revisitKey).
					Msg("Failed to writecrawled content")
			}
		}

		storageRef := warcFileScheme + ":" + res.FileName + ":" + strconv.FormatInt(res.FileOffset, 10)
		collectionFinalName := ww.filePrefix[:len(ww.filePrefix)-1]
		recordMeta[recNum] = &contentwriterV1.WriteResponseMeta_RecordMeta{
			RecordNum:           recNum,
			Type:                FromGowarcRecordType(record[i].Type()),
			WarcId:              warcId.String(),
			StorageRef:          storageRef,
			BlockDigest:         rec.WarcHeader().Get(gowarc.WarcBlockDigest),
			PayloadDigest:       rec.WarcHeader().Get(gowarc.WarcPayloadDigest),
			RevisitReferenceId:  rec.WarcHeader().Get(gowarc.WarcRefersTo),
			CollectionFinalName: collectionFinalName,
		}
	}
	return &contentwriterV1.WriteReply{
		Meta: &contentwriterV1.WriteResponseMeta{
			RecordMeta: recordMeta,
		},
	}, err
}

func (ww *warcWriter) toRevisitRecord(recordNum int32, record gowarc.WarcRecord, meta *contentwriterV1.WriteRequestMeta, crawledContent *contentwriterV1.CrawledContent) (gowarc.WarcRecord, error) {
	ref := &gowarc.RevisitRef{
		Profile:        ww.revisitProfile,
		TargetRecordId: "<urn:uuid:" + crawledContent.GetWarcId() + ">",
		TargetUri:      crawledContent.GetTargetUri(),
		TargetDate:     crawledContent.GetDate().AsTime().In(time.UTC).Format(time.RFC3339),
	}
	revisit, err := record.ToRevisitRecord(ref)
	if err != nil {
		return record, fmt.Errorf("failed to create revisit record: %w", err)
	}

	newRecordMeta := meta.GetRecordMeta()[recordNum]
	newRecordMeta.Type = contentwriterV1.RecordType_REVISIT
	newRecordMeta.BlockDigest = revisit.Block().BlockDigest()
	if r, ok := revisit.Block().(gowarc.PayloadBlock); ok {
		newRecordMeta.PayloadDigest = r.PayloadDigest()
	}

	size, err := strconv.ParseInt(revisit.WarcHeader().Get(gowarc.ContentLength), 10, 64)
	if err != nil {
		return record, fmt.Errorf("failed to parse content length from revisit record: %w", err)
	}
	newRecordMeta.Size = size
	meta.GetRecordMeta()[recordNum] = newRecordMeta

	return revisit, nil
}

func (ww *warcWriter) initFileWriter() {
	log.Debug().Msgf("Initializing filewriter with dir: '%s' and file prefix: '%s'", ww.opts.WarcDir, ww.filePrefix)
	c := ww.collectionConfig.GetCollection()
	namer := &gowarc.PatternNameGenerator{
		Directory: ww.opts.WarcDir,
		Prefix:    ww.filePrefix,
	}

	opts := []gowarc.WarcFileWriterOption{
		gowarc.WithCompression(c.GetCompress()),
		gowarc.WithMaxFileSize(c.GetFileSize()),
		gowarc.WithFileNameGenerator(namer),
		gowarc.WithWarcInfoFunc(ww.warcInfoGenerator),
		gowarc.WithMaxConcurrentWriters(ww.opts.WarcWriterPoolSize),
		gowarc.WithAddWarcConcurrentToHeader(true),
		gowarc.WithFlush(ww.opts.FlushRecord),
		gowarc.WithRecordOptions(gowarc.WithVersion(ww.opts.WarcVersion)),
	}

	ww.fileWriter = gowarc.NewWarcFileWriter(opts...)
}

func (ww *warcWriter) waitForTimer(rotationPolicy configV1.Collection_RotationPolicy) bool {
	select {
	case <-ww.done:
	case <-ww.timer.C:
		c := ww.collectionConfig.GetCollection()
		prefix := createFilePrefix(ww.collectionConfig.GetMeta().GetName(), ww.subCollection, now(), c.GetCollectionDedupPolicy())
		if prefix != ww.filePrefix {
			ww.lock.Lock()
			defer ww.lock.Unlock()
			ww.filePrefix = prefix
			if err := ww.fileWriter.Close(); err != nil {
				log.Err(err).Msg("failed closing file writer")
			}
			ww.fileWriter = nil
			ww.initFileWriter()
		} else {
			if err := ww.fileWriter.Rotate(); err != nil {
				log.Err(err).Msg("failed rotating file")
			}
		}

		if d, ok := timeToNextRotation(now(), rotationPolicy); ok {
			ww.timer.Reset(d)
		}
		return true
	}

	// We still need to check the return value
	// of Stop, because timer could have fired
	// between the receive on done and this line.
	if !ww.timer.Stop() {
		<-ww.timer.C
	}
	return false
}

func (ww *warcWriter) Shutdown() {
	if ww.timer != nil {
		close(ww.done)
	}
	if err := ww.fileWriter.Close(); err != nil {
		log.Err(err).Msg("failed closing file writer")
	}
}

func timeToNextRotation(now time.Time, p configV1.Collection_RotationPolicy) (time.Duration, bool) {
	var t2 time.Time

	switch p {
	case configV1.Collection_HOURLY:
		t2 = time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
	case configV1.Collection_DAILY:
		t2 = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	case configV1.Collection_MONTHLY:
		t2 = time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	case configV1.Collection_YEARLY:
		t2 = time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, now.Location())
	default:
		return 0, false
	}

	d := t2.Sub(now)
	return d, true
}

func createFileRotationKey(now time.Time, p configV1.Collection_RotationPolicy) string {
	switch p {
	case configV1.Collection_HOURLY:
		return now.Format("2006010215")
	case configV1.Collection_DAILY:
		return now.Format("20060102")
	case configV1.Collection_MONTHLY:
		return now.Format("200601")
	case configV1.Collection_YEARLY:
		return now.Format("2006")
	default:
		return ""
	}
}

func createFilePrefix(collectionName string, subCollection configV1.Collection_SubCollectionType, ts time.Time, dedupPolicy configV1.Collection_RotationPolicy) string {
	if subCollection != configV1.Collection_UNDEFINED {
		collectionName += "_" + subCollection.String()
	}

	dedupRotationKey := createFileRotationKey(ts, dedupPolicy)
	if dedupRotationKey == "" {
		return collectionName + "-"
	} else {
		return collectionName + "_" + dedupRotationKey + "-"
	}
}

func timeToLive(p configV1.Collection_RotationPolicy) time.Duration {
	now := time.Now().UTC()

	var next time.Time

	switch p {
	case configV1.Collection_HOURLY:
		next = now.Truncate(time.Hour).Add(time.Hour)

	case configV1.Collection_DAILY:
		next = time.Date(
			now.Year(), now.Month(), now.Day(),
			0, 0, 0, 0, time.UTC,
		).AddDate(0, 0, 1)

	case configV1.Collection_MONTHLY:
		next = time.Date(
			now.Year(), now.Month(), 1,
			0, 0, 0, 0, time.UTC,
		).AddDate(0, 1, 0)

	case configV1.Collection_YEARLY:
		next = time.Date(
			now.Year(), 1, 1,
			0, 0, 0, 0, time.UTC,
		).AddDate(1, 0, 0)

	default:
		return 0
	}

	return time.Until(next)
}
