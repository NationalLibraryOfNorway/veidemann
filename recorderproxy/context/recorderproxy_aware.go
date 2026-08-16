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

package context

import (
	"context"
	"net/url"
	"sync"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
)

type ctxKey string

const (
	ctxKeyRecorderProxyAware = ctxKey("recorderProxyAware")
	ctxKeyRCTX               = ctxKey("recordContext")
	ctxKeyHost               = ctxKey("host")
	ctxKeyPort               = ctxKey("port")
	ctxKeyUrl                = ctxKey("url")
	ctxKeyRequestId          = ctxKey("reqid")
	ctxKeyCrawlExecutionId   = ctxKey("eid")
	ctxKeyJobExecutionId     = ctxKey("jid")
	ctxKeyCollectionRef      = ctxKey("cid")
	ctxKeyIp                 = ctxKey("ip")
)

type stateHandle struct {
	mu     sync.RWMutex
	values map[ctxKey]interface{}
}

func lookupStateHandle(ctx context.Context) (*stateHandle, bool) {
	h, ok := ctx.Value(ctxKeyRecorderProxyAware).(*stateHandle)
	return h, ok
}

func RecordProxyDataAware(ctx context.Context) context.Context {
	if _, ok := lookupStateHandle(ctx); ok {
		return ctx
	}

	return newStateContext(ctx)
}

func newStateContext(ctx context.Context) context.Context {
	h := &stateHandle{values: make(map[ctxKey]interface{}, 8)}
	return context.WithValue(ctx, ctxKeyRecorderProxyAware, h)
}

func copySessionMetadata(dst context.Context, src context.Context) {
	if host := GetHost(src); host != "" {
		SetHost(dst, host)
	}
	if port := GetPort(src); port != "" {
		SetPort(dst, port)
	}
	if uri := GetUri(src); uri != nil {
		uriCopy := *uri
		SetUri(dst, &uriCopy)
	}
	if crawlExecutionId := GetCrawlExecutionId(src); crawlExecutionId != "" {
		SetCrawlExecutionId(dst, crawlExecutionId)
	}
	if jobExecutionId := GetJobExecutionId(src); jobExecutionId != "" {
		SetJobExecutionId(dst, jobExecutionId)
	}
	if collectionRef := GetCollectionRef(src); collectionRef != nil {
		SetCollectionRef(dst, collectionRef)
	}
}

func CopySessionMetadata(dst context.Context, src context.Context) {
	copySessionMetadata(dst, src)
}

func NewRequestContextFrom(ctx context.Context, source context.Context, preserveSessionMetadata bool) context.Context {
	ctx = newStateContext(ctx)

	if !preserveSessionMetadata {
		return ctx
	}

	copySessionMetadata(ctx, source)

	return ctx
}

func NewRequestContext(ctx context.Context, preserveSessionMetadata bool) context.Context {
	return NewRequestContextFrom(ctx, ctx, preserveSessionMetadata)
}

func WithStateHandle(ctx context.Context, source context.Context) context.Context {
	h, ok := lookupStateHandle(source)
	if !ok {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyRecorderProxyAware, h)
}

func HasStateHandle(ctx context.Context) bool {
	_, ok := lookupStateHandle(ctx)
	return ok
}

func ResetRequestState(ctx context.Context, preserveSessionMetadata bool) {
	h, ok := lookupStateHandle(ctx)
	if !ok {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.values, ctxKeyRCTX)
	delete(h.values, ctxKeyRequestId)
	delete(h.values, ctxKeyIp)

	if !preserveSessionMetadata {
		delete(h.values, ctxKeyHost)
		delete(h.values, ctxKeyPort)
		delete(h.values, ctxKeyUrl)
	}

	if !preserveSessionMetadata {
		delete(h.values, ctxKeyCrawlExecutionId)
		delete(h.values, ctxKeyJobExecutionId)
		delete(h.values, ctxKeyCollectionRef)
	}
}

func SetHost(ctx context.Context, host string) {
	setValue(ctx, ctxKeyHost, host)
}

func SetPort(ctx context.Context, port string) {
	setValue(ctx, ctxKeyPort, port)
}

func SetUri(ctx context.Context, uri *url.URL) {
	setValue(ctx, ctxKeyUrl, uri)
}

func SetRecordContext(ctx context.Context, rc *RecordContext) {
	setValue(ctx, ctxKeyRCTX, rc)
}

func SetRequestId(ctx context.Context, reqid string) {
	setValue(ctx, ctxKeyRequestId, reqid)
}

func SetCrawlExecutionId(ctx context.Context, eid string) {
	setValue(ctx, ctxKeyCrawlExecutionId, eid)
}

func SetJobExecutionId(ctx context.Context, jid string) {
	setValue(ctx, ctxKeyJobExecutionId, jid)
}

func SetCollectionRef(ctx context.Context, cid *configV1.ConfigRef) {
	setValue(ctx, ctxKeyCollectionRef, cid)
}

func SetIp(ctx context.Context, ip string) {
	setValue(ctx, ctxKeyIp, ip)
}

func GetHost(ctx context.Context) (host string) {
	host, _ = getValue(ctx, ctxKeyHost).(string)
	return
}

func GetPort(ctx context.Context) (port string) {
	port, _ = getValue(ctx, ctxKeyPort).(string)
	return
}

func GetUri(ctx context.Context) (uri *url.URL) {
	uri, _ = getValue(ctx, ctxKeyUrl).(*url.URL)
	return
}

func GetRecordContext(ctx context.Context) (recordContext *RecordContext) {
	recordContext, _ = getValue(ctx, ctxKeyRCTX).(*RecordContext)
	return
}

func GetRequestId(ctx context.Context) (reqid string) {
	reqid, _ = getValue(ctx, ctxKeyRequestId).(string)
	return
}

func GetCrawlExecutionId(ctx context.Context) (eid string) {
	eid, _ = getValue(ctx, ctxKeyCrawlExecutionId).(string)
	return
}

func GetJobExecutionId(ctx context.Context) (jid string) {
	jid, _ = getValue(ctx, ctxKeyJobExecutionId).(string)
	return
}

func GetCollectionRef(ctx context.Context) (cid *configV1.ConfigRef) {
	cid, _ = getValue(ctx, ctxKeyCollectionRef).(*configV1.ConfigRef)
	return
}

func GetIp(ctx context.Context) (ip string) {
	ip, _ = getValue(ctx, ctxKeyIp).(string)
	return
}

func getValue(ctx context.Context, key ctxKey) interface{} {
	h, ok := lookupStateHandle(ctx)
	if !ok {
		return nil
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.values[key]
}

func setValue(ctx context.Context, key ctxKey, value interface{}) {
	h, ok := lookupStateHandle(ctx)
	if !ok {
		logger.Log.Panic("BUG: Tried to set RecordProxyDataAware on uninitialized context")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if value == nil {
		delete(h.values, key)
		return
	}
	h.values[key] = value
}

func WrapIfNecessary(ctx context.Context) context.Context {
	return ctx
}
