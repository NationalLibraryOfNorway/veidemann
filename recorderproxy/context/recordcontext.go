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

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	rperrors "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
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
	MainDocument bool
	Harvest      *recorderV1.BeginHarvestRequest
	// Complete persists the resource outcome; attemptFailure can differ from its logged error.
	Complete func(crawlLog *logV1.CrawlLog, cached bool, attemptFailure *commonsV1.Error)
	// ResolveDiscovery runs during Init, before terminal processing can start.
	// The owning attempt serializes initialization with admission.
	ResolveDiscovery func(*http.Request, *url.URL, bool) string
	// ObserveResponse records ancestry evidence before headers reach the browser.
	ObserveResponse func(*http.Response)

	CloseFunc func()

	// Will connect a request to a response
	session int64

	contentWriterClient contentwriterV1.ContentWriterClient
	ctx                 context.Context
	cwc                 *CwcSession
	Method              string
	Uri                 *url.URL
	ResourceType        string
	CrawlExecutionId    string
	JobExecutionId      string
	CollectionRef       *configV1.ConfigRef
	IpAddress           string
	FetchTimesTamp      time.Time
	Meta                *contentwriterV1.WriteRequest_Meta
	CrawlLog            *logV1.CrawlLog
	closed              bool
	FoundInCache        bool
	PrecludedByRobots   bool
	done                bool
	mutex               sync.Mutex
	crawlLogMutex       sync.RWMutex
	terminalDone        chan struct{}
	terminalErr         error
	finalizationTimeout time.Duration
	lifecycleCancel     context.CancelFunc
	cwcCreating         bool
	cwcReady            chan struct{}
	cwcErr              error
	InitDone            bool
	ProxyId             int32
	log                 *logger.Logger
}

// NewRecordContext creates a new RecordContext
func NewRecordContext(finalizationTimeout ...time.Duration) *RecordContext {
	timeout := defaultFinalizationTimeout
	if len(finalizationTimeout) > 0 && finalizationTimeout[0] > 0 {
		timeout = finalizationTimeout[0]
	}
	rc := &RecordContext{

		terminalDone:        make(chan struct{}),
		finalizationTimeout: timeout,
	}

	return rc
}

func (rc *RecordContext) Init(proxyId int32, contentWriter contentwriterV1.ContentWriterClient, req *http.Request, uri *url.URL) *RecordContext {
	rc.session = atomic.AddInt64(&sess, 1)
	rc.contentWriterClient = contentWriter
	rc.ctx = req.Context()
	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background())
	rc.lifecycleCancel = lifecycleCancel
	rc.ProxyId = proxyId
	rc.Method = req.Method
	marker := req.Header.Get(constants.HeaderMainDocument)
	rc.MainDocument = marker != "" && marker == rc.Harvest.GetAttemptId()
	req.Header.Del(constants.HeaderMainDocument)
	rc.ResourceType = resourceType(req.Header.Get("Sec-Fetch-Dest"))
	rc.Uri = uri
	rc.CrawlExecutionId = GetCrawlExecutionId(rc.ctx)
	rc.JobExecutionId = GetJobExecutionId(rc.ctx)
	rc.CollectionRef = GetCollectionRef(rc.ctx)
	rc.IpAddress = GetIp(rc.ctx)
	if rc.Harvest != nil {
		rc.CrawlExecutionId = rc.Harvest.CrawlExecutionId
		rc.JobExecutionId = rc.Harvest.JobExecutionId
		rc.CollectionRef = rc.Harvest.CollectionRef
		SetCrawlExecutionId(rc.ctx, rc.CrawlExecutionId)
		SetJobExecutionId(rc.ctx, rc.JobExecutionId)
		SetCollectionRef(rc.ctx, rc.CollectionRef)
	}

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
		Referrer:       req.Referer(),
		Method:         rc.Method,
		IpAddress:      rc.IpAddress,
	}

	if rc.Harvest != nil {
		rc.CrawlLog.DiscoveryPath = rc.Harvest.DiscoveryPath
		if !rc.MainDocument {
			switch req.Header.Get("Sec-Fetch-Dest") {
			case "empty":
				rc.CrawlLog.DiscoveryPath += "X"
			default:
				rc.CrawlLog.DiscoveryPath += "E"
			}
		}
	}
	if rc.ResolveDiscovery != nil {
		rc.CrawlLog.DiscoveryPath = rc.ResolveDiscovery(req, uri, rc.MainDocument)
	}
	rc.InitDone = true

	rc.log = logger.Log.WithFields(logger.Fields{
		"harvestId":         rc.Harvest.GetAttemptId(),
		"slot":              rc.Harvest.GetProxyId(),
		"pageUrl":           logger.DiagnosticURL(rc.Harvest.GetRequestedUri()),
		"method":            req.Method,
		"url":               logger.DiagnosticURL(uri.String()),
		"recorderRequestId": rc.Session(),
	})

	rc.log.Trace("Recorder request admitted")

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
	return rc.log.WithField("phase", componentName)
}

func LogWithContext(ctx context.Context, componentName string) *logger.Logger {
	var l *logger.Logger
	rc := GetRecordContext(ctx)
	if rc != nil {
		l = rc.log
	} else {
		l = HarvestLogger(ctx)
	}
	l = l.WithField("phase", componentName)
	return l
}

func LogWithContextAndRequest(ctx context.Context, req *http.Request, componentName string) *logger.Logger {
	var l *logger.Logger

	rc := GetRecordContext(ctx)
	if rc != nil {
		l = rc.log
	} else {
		l = HarvestLogger(ctx).WithFields(logger.Fields{
			"method": req.Method,
			"url":    logger.DiagnosticURL(req.URL.String()),
		})
	}
	l = l.WithField("phase", componentName)
	return l
}

type harvestLogKey struct{}

// WithHarvestLogger pins diagnostic correlation to the accepted connection.
func WithHarvestLogger(ctx context.Context, log *logger.Logger) context.Context {
	return context.WithValue(ctx, harvestLogKey{}, log)
}

// HarvestLogger returns the diagnostic context pinned to the connection.
func HarvestLogger(ctx context.Context) *logger.Logger {
	if log, ok := ctx.Value(harvestLogKey{}).(*logger.Logger); ok {
		return log
	}
	return logger.Log
}

// HTTP fetch metadata supplies coarse page-log labels without CDP correlation.
func resourceType(destination string) string {
	switch destination {
	case "document", "frame", "iframe":
		return "Document"
	case "script", "worker", "sharedworker", "serviceworker":
		return "Script"
	case "style":
		return "Stylesheet"
	case "image":
		return "Image"
	case "font":
		return "Font"
	case "audio", "video", "track":
		return "Media"
	case "manifest":
		return "Manifest"
	default:
		return ""
	}
}
