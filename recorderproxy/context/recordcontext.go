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
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	rperrors "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/serviceconnections"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// session variable must be aligned in i386
// see http://golang.org/src/pkg/sync/atomic/doc.go#L41
var sess int64
var closedSess int64

const defaultFinalizationTimeout = 30 * time.Second

func OpenSessions() int64 {
	return sess - closedSess
}

type RecordContext struct {
	Error error

	CloseFunc func()

	// Will connect a request to a response
	session int64

	conn                      *serviceconnections.Connections
	ctx                       context.Context
	cwc                       *CwcSession
	Method                    string
	Uri                       *url.URL
	RequestId                 string
	CrawlExecutionId          string
	JobExecutionId            string
	CollectionRef             *configV1.ConfigRef
	IpAddress                 string
	FetchTimesTamp            time.Time
	Meta                      *contentwriterV1.WriteRequest_Meta
	CrawlLog                  *logV1.CrawlLog
	ReplacementScript         *configV1.BrowserScript
	closed                    bool
	FoundInCache              bool
	PrecludedByRobots         bool
	HasExplicitHarvestHeaders bool
	done                      bool
	mutex                     sync.Mutex
	crawlLogMutex             sync.RWMutex
	terminalDone              chan struct{}
	terminalErr               error
	finalizationTimeout       time.Duration
	lifecycleCancel           context.CancelFunc
	cwcCreating               bool
	cwcReady                  chan struct{}
	cwcErr                    error
	InitDone                  bool
	ProxyId                   int32
	log                       *logger.Logger
}

// NewRecordContext creates a new RecordContext
func NewRecordContext(finalizationTimeout ...time.Duration) *RecordContext {
	timeout := defaultFinalizationTimeout
	if len(finalizationTimeout) > 0 && finalizationTimeout[0] > 0 {
		timeout = finalizationTimeout[0]
	}
	rc := &RecordContext{
		session:             atomic.AddInt64(&sess, 1),
		terminalDone:        make(chan struct{}),
		finalizationTimeout: timeout,
	}

	return rc
}

func (rc *RecordContext) Init(proxyId int32, conn *serviceconnections.Connections, req *http.Request, uri *url.URL) *RecordContext {
	rc.conn = conn
	rc.ctx = req.Context()
	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background())
	rc.lifecycleCancel = lifecycleCancel
	rc.ProxyId = proxyId
	rc.Method = req.Method
	rc.Uri = uri
	rc.HasExplicitHarvestHeaders = req.Header.Get(constants.HeaderCrawlExecutionId) != "" ||
		req.Header.Get(constants.HeaderJobExecutionId) != "" ||
		req.Header.Get(constants.HeaderCollectionId) != ""

	resolveIdsFromHttpHeader(rc.ctx, req)
	rc.RequestId = GetRequestId(rc.ctx)
	rc.CrawlExecutionId = GetCrawlExecutionId(rc.ctx)
	rc.JobExecutionId = GetJobExecutionId(rc.ctx)
	rc.CollectionRef = GetCollectionRef(rc.ctx)
	rc.IpAddress = GetIp(rc.ctx)

	req.Header.Del(constants.HeaderRequestId)
	req.Header.Del(constants.HeaderCrawlExecutionId)
	req.Header.Del(constants.HeaderJobExecutionId)
	req.Header.Del(constants.HeaderCollectionId)

	rc.FetchTimesTamp = time.Now()

	rc.CrawlLog = &logV1.CrawlLog{
		JobExecutionId: rc.JobExecutionId,
		ExecutionId:    rc.CrawlExecutionId,
		FetchTimeStamp: timestamppb.New(rc.FetchTimesTamp),
		RequestedUri:   uri.String(),
		Method:         rc.Method,
		IpAddress:      rc.IpAddress,
	}

	rc.InitDone = true

	rc.log = logger.Log.WithFields(logger.Fields{
		"component": "PROXY",
		"method":    req.Method,
		"url":       uri.String(),
		"session":   rc.Session(),
	})

	rc.log.Infof("New session")

	go func() {
		select {
		case <-rc.ctx.Done():
			cancelErr := rperrors.Error(
				rperrors.CanceledByBrowser,
				"CANCELED_BY_BROWSER",
				"Veidemann recorder proxy lost connection to client",
			)
			_ = rc.finalizeError(cancelErr, "Veidemann recorder proxy lost connection to client")
		case <-lifecycleCtx.Done():
		}
	}()

	return rc
}

func (rc *RecordContext) Session() int64 {
	return rc.session
}

func (rc *RecordContext) closeSession() {
	if rc == nil {
		return
	}

	rc.mutex.Lock()
	if rc.closed {
		rc.mutex.Unlock()
		return
	}
	rc.closed = true
	rc.mutex.Unlock()

	atomic.AddInt64(&closedSess, 1)
	if rc.CloseFunc != nil {
		rc.CloseFunc()
	}
	if rc.lifecycleCancel != nil {
		rc.lifecycleCancel()
	}
}

func LogWithRecordContext(rc *RecordContext, componentName string) *logger.Logger {
	return rc.log.WithField("component", componentName)
}

func LogWithContext(ctx context.Context, componentName string) *logger.Logger {
	var l *logger.Logger
	rc := GetRecordContext(ctx)
	if rc != nil {
		l = rc.log
	} else {
		l = logger.Log
	}
	l = l.WithField("component", componentName)
	return l
}

func LogWithContextAndRequest(ctx context.Context, req *http.Request, componentName string) *logger.Logger {
	var l *logger.Logger

	rc := GetRecordContext(ctx)
	if rc != nil {
		l = rc.log
	} else {
		l = logger.Log.WithFields(logger.Fields{
			"method": req.Method,
			"url":    req.URL.String(),
		})
	}
	l = l.WithField("component", componentName)
	return l
}

func resolveIdsFromHttpHeader(ctx context.Context, req *http.Request) {
	jid := req.Header.Get(constants.HeaderJobExecutionId)
	eid := req.Header.Get(constants.HeaderCrawlExecutionId)
	reqid := req.Header.Get(constants.HeaderRequestId)
	SetJobExecutionId(ctx, jid)
	SetCrawlExecutionId(ctx, eid)
	SetRequestId(ctx, reqid)

	if req.Header.Get(constants.HeaderCollectionId) != "" {
		cid := req.Header.Get(constants.HeaderCollectionId)
		SetCollectionRef(ctx, &configV1.ConfigRef{
			Kind: configV1.Kind_collection,
			Id:   cid,
		})
	}
}
