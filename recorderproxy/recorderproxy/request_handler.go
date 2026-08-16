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
	"sync"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
)

const requestRecordNum int32 = 0

type wrappedRequestBody struct {
	io.ReadCloser

	ctx           context.Context
	recordContext *rpcontext.RecordContext
	log           *logger.Logger

	recNum      int32
	size        int64
	blockDigest hash.Hash
	recordMeta  *contentwriterV1.WriteRequestMeta_RecordMeta

	mu     sync.Mutex
	eof    bool
	closed bool
	failed error
}

func WrapRequestBody(
	ctx context.Context,
	rc *rpcontext.RecordContext,
	body io.ReadCloser,
	contentType string,
	prolog []byte,
) (*wrappedRequestBody, error) {
	if body == nil {
		body = http.NoBody
	}

	if err := ensureRequestRecordContext(rc); err != nil {
		return nil, err
	}

	b := &wrappedRequestBody{
		ReadCloser:    body,
		ctx:           ctx,
		recordContext: rc,
		recNum:        requestRecordNum,
		size:          int64(len(prolog)),
		blockDigest:   sha1.New(),
	}

	b.log = rpcontext.
		LogWithRecordContext(rc, "BODY:req").
		WithField("url", rc.Uri.String())

	b.recordMeta = &contentwriterV1.WriteRequestMeta_RecordMeta{
		RecordNum:         b.recNum,
		Type:              contentwriterV1.RecordType_REQUEST,
		RecordContentType: constants.RecordContentTypeRequest,
	}

	rc.Meta.Meta.RecordMeta[b.recNum] = b.recordMeta
	rc.UpdateCrawlLog(func(cl *logV1.CrawlLog) {
		cl.StatusCode = -1
		cl.ContentType = contentType
	})

	_, _ = b.blockDigest.Write(prolog)

	if err := rc.SendProtocolHeader(b.recNum, prolog); err != nil {
		return nil, fmt.Errorf("error writing request protocol header to content writer: %w", err)
	}

	return b, nil
}

func ensureRequestRecordContext(rc *rpcontext.RecordContext) error {
	if rc == nil {
		return fmt.Errorf("record context is nil")
	}
	if rc.Uri == nil {
		return fmt.Errorf("record context uri is nil")
	}
	if !rc.HasCrawlLog() {
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

func (b *wrappedRequestBody) Read(p []byte) (int, error) {
	r, err := b.reader()
	if err != nil {
		return 0, err
	}

	n, readErr := r.Read(p)
	b.logRead(n, readErr)

	if n > 0 {
		if err := b.recordPayload(p[:n]); err != nil {
			b.setFailed(err)
			return n, err
		}
	}

	if readErr == io.EOF {
		b.finishRecord()
	}

	return n, readErr
}

func (b *wrappedRequestBody) reader() (io.Reader, error) {
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
		return b.ReadCloser, nil
	}
}

func (b *wrappedRequestBody) Close() error {
	rc := b.closeState()

	var err error
	if rc != nil {
		err = rc.Close()
	}

	if err != nil {
		b.log.WithError(err).Debug("Close body")
	} else {
		b.log.Debug("Close body")
	}

	return err
}

func (b *wrappedRequestBody) closeState() io.Closer {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true
	return b.ReadCloser
}

func (b *wrappedRequestBody) logRead(n int, err error) {
	if err != nil && err != io.EOF {
		b.log.WithError(err).Warnf("Inner read %d", n)
		return
	}

	b.log.Tracef("Inner read %d", n)
}

func (b *wrappedRequestBody) recordPayload(data []byte) error {
	b.mu.Lock()

	b.size += int64(len(data))
	_, _ = b.blockDigest.Write(data)

	recNum := b.recNum
	rc := b.recordContext

	b.mu.Unlock()

	if err := rc.SendPayload(recNum, data); err != nil {
		b.log.WithError(err).Error("Error writing request payload")
		return err
	}

	return nil
}

func (b *wrappedRequestBody) finishRecord() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.eof {
		return
	}

	b.eof = true

	blockDigest := fmt.Sprintf("sha1:%x", b.blockDigest.Sum(nil))

	b.recordMeta.Size = b.size
	b.recordMeta.BlockDigest = blockDigest
}

func (b *wrappedRequestBody) setFailed(err error) {
	if err == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.failed == nil {
		b.failed = err
	}
}
