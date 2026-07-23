/*
 * Copyright 2019 National Library of Norway.
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

package recorderproxy

import (
	"context"
	"crypto/sha1"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strings"
	"sync"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
)

const responseRecordNum int32 = 1

type wrappedResponseBody struct {
	io.ReadCloser

	ctx           context.Context
	recordContext *rpcontext.RecordContext
	log           *logger.Logger

	recNum      int32
	size        int64
	blockDigest hash.Hash
	recordMeta  *contentwriterV1.WriteRequestMeta_RecordMeta

	mu        sync.Mutex
	eof       bool
	closed    bool
	finalized bool
	failed    error
}

func WrapResponseBody(
	ctx context.Context,
	rc *rpcontext.RecordContext,
	body io.ReadCloser,
	statusCode int32,
	contentType string,
	recordType contentwriterV1.RecordType,
	prolog []byte,
) (*wrappedResponseBody, error) {
	if body == nil {
		body = http.NoBody
	}

	if err := ensureResponseRecordContext(rc); err != nil {
		return nil, err
	}

	b := &wrappedResponseBody{
		ReadCloser:    body,
		ctx:           ctx,
		recordContext: rc,
		recNum:        responseRecordNum,
		size:          int64(len(prolog)),
		blockDigest:   sha1.New(),
	}

	b.log = rpcontext.
		LogWithRecordContext(rc, "BODY:resp").
		WithField("url", rc.Uri.String())

	b.recordMeta = &contentwriterV1.WriteRequestMeta_RecordMeta{
		RecordNum:         b.recNum,
		Type:              recordType,
		RecordContentType: constants.RecordContentTypeResponse,
	}

	rc.Meta.Meta.RecordMeta[b.recNum] = b.recordMeta
	rc.CrawlLog.StatusCode = statusCode
	rc.CrawlLog.ContentType = contentType

	_, _ = b.blockDigest.Write(prolog)

	if rc.FoundInCache {
		return b, nil
	}

	if err := rc.SendProtocolHeader(b.recNum, prolog); err != nil {
		return nil, fmt.Errorf("error writing response protocol header to content writer: %w", err)
	}

	return b, nil
}

func ensureResponseRecordContext(rc *rpcontext.RecordContext) error {
	if rc == nil {
		return fmt.Errorf("record context is nil")
	}
	if rc.Uri == nil {
		return fmt.Errorf("record context uri is nil")
	}
	if rc.CrawlLog == nil {
		return fmt.Errorf("record context crawl log is nil")
	}
	if rc.Meta == nil || rc.Meta.Meta == nil {
		return fmt.Errorf("record context meta is nil")
	}
	if rc.Meta.Meta.RecordMeta == nil {
		rc.Meta.Meta.RecordMeta = map[int32]*contentwriterV1.WriteRequestMeta_RecordMeta{}
	}

	return nil
}

func (b *wrappedResponseBody) Read(p []byte) (int, error) {
	r, err := b.reader()
	if err != nil {
		return 0, err
	}

	// WIN: single path. The bytes read from upstream are the same bytes sent to
	// the client and the same bytes recorded. No replacement-body fork.
	n, readErr := r.Read(p)
	b.logRead(n, readErr)

	if n > 0 {
		if err := b.recordPayload(p[:n]); err != nil {
			b.setFailed(err)
			return n, err
		}
	}

	if readErr == io.EOF {
		if err := b.finishRecord(); err != nil {
			b.setFailed(err)
			return n, err
		}
	}

	return n, readErr
}

func (b *wrappedResponseBody) reader() (io.Reader, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch {
	case b.closed:
		return nil, http.ErrBodyReadAfterClose
	case b.failed != nil:
		return nil, b.failed
	case b.eof:
		return nil, io.EOF
	case b.ReadCloser == nil:
		b.eof = true
		return nil, io.EOF
	default:
		// WIN: return the reader and release the lock before the blocking Read.
		return b.ReadCloser, nil
	}
}

func (b *wrappedResponseBody) Close() error {
	rc, prematureClose := b.closeState()

	var err error
	if rc != nil {
		err = rc.Close()
	}

	if prematureClose {
		cancelMsg := "Veidemann recorder proxy lost connection to client"

		b.log.WithError(err).Warn("Response body closed before EOF")

		_ = b.recordContext.SendResponseError(
			b.ctx,
			errors.Error(errors.CanceledByBrowser, "CANCELED_BY_BROWSER", cancelMsg),
		)
		_ = b.recordContext.CancelContentWriter(cancelMsg)
	}

	if err != nil {
		b.log.WithError(err).Debug("Close body")
	} else {
		b.log.Debug("Close body")
	}

	return err
}

func (b *wrappedResponseBody) closeState() (io.Closer, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		// WIN: Close is idempotent.
		return nil, false
	}

	b.closed = true

	prematureClose := !b.eof && !b.finalized
	if prematureClose {
		b.eof = true
	}

	return b.ReadCloser, prematureClose
}

func (b *wrappedResponseBody) logRead(n int, err error) {
	if err != nil && err != io.EOF {
		b.log.WithError(err).Warnf("Inner read %d", n)
		return
	}

	b.log.Tracef("Inner read %d", n)
}

func (b *wrappedResponseBody) recordPayload(data []byte) error {
	b.mu.Lock()

	// WIN: cached responses are still counted, but not written to content writer.
	b.size += int64(len(data))

	if b.recordContext.FoundInCache {
		b.mu.Unlock()
		return nil
	}

	_, _ = b.blockDigest.Write(data)

	recNum := b.recNum
	rc := b.recordContext

	b.mu.Unlock()

	// WIN: content-writer errors are no longer swallowed.
	if err := rc.SendPayload(recNum, data); err != nil {
		b.log.WithError(err).Error("Error writing response payload")
		return err
	}

	return nil
}

func (b *wrappedResponseBody) finishRecord() error {
	b.mu.Lock()

	if b.finalized {
		b.eof = true
		b.mu.Unlock()
		return nil
	}

	b.eof = true
	b.finalized = true

	if b.recordContext.FoundInCache {
		b.mu.Unlock()
		return b.finishCachedRecord()
	}

	blockDigest := fmt.Sprintf("sha1:%x", b.blockDigest.Sum(nil))

	b.recordMeta.Size = b.size
	b.recordMeta.BlockDigest = blockDigest

	recNum := b.recNum
	size := b.size
	rc := b.recordContext

	b.mu.Unlock()

	// WIN: finalization is separated from stream reading.
	return finishStoredResponseRecord(b.ctx, rc, recNum, size, blockDigest)
}

func (b *wrappedResponseBody) finishCachedRecord() error {
	cl := b.recordContext.CrawlLog
	cl.Size = b.size

	if err := b.recordContext.SaveCrawlLog(); err != nil {
		b.log.WithError(err).Error("Error saving cached crawllog")
		return err
	}

	_ = b.recordContext.CancelContentWriter("OK: Loaded from cache")
	return nil
}

func (b *wrappedResponseBody) setFailed(err error) {
	if err == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.failed == nil {
		b.failed = err
	}
}

func finishStoredResponseRecord(
	ctx context.Context,
	rc *rpcontext.RecordContext,
	recNum int32,
	size int64,
	blockDigest string,
) error {
	cwReply, err := rc.SendMeta()
	if err != nil {
		return rc.SendResponseError(
			ctx,
			errors.Wrap(
				err,
				errors.RuntimeException,
				"Error writing to content writer",
				err.Error(),
			),
		)
	}

	if cwReply == nil {
		return nil
	}
	if cwReply.Meta == nil {
		return fmt.Errorf("content writer reply meta is nil")
	}

	meta, ok := cwReply.Meta.RecordMeta[recNum]
	if !ok || meta == nil {
		return fmt.Errorf("content writer reply missing record metadata for record %d", recNum)
	}

	cl := rc.CrawlLog
	cl.CollectionFinalName = meta.CollectionFinalName
	cl.WarcId = meta.WarcId
	cl.StorageRef = meta.StorageRef
	cl.WarcRefersTo = meta.RevisitReferenceId
	cl.Size = size
	cl.RecordType = strings.ToLower(meta.Type.String())
	cl.BlockDigest = blockDigest
	cl.PayloadDigest = meta.PayloadDigest

	if err := rc.SaveCrawlLog(); err != nil {
		rpcontext.LogWithRecordContext(rc, "BODY:resp").
			WithError(err).
			Error("Error saving crawllog")
	}

	return nil
}
