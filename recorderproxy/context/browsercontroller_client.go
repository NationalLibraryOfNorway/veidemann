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
	errors2 "errors"
	"net/http"
	"net/url"
	"time"

	browsercontrollerV2 "github.com/NationalLibraryOfNorway/veidemann/api/browsercontroller/v2"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/serviceconnections"
	"github.com/opentracing/opentracing-go"
	otLog "github.com/opentracing/opentracing-go/log"
	"google.golang.org/protobuf/proto"
)

var AlreadyCompleted = errors2.New("already completed")

func cloneCrawlLog(crawlLog *logV1.CrawlLog) *logV1.CrawlLog {
	if crawlLog == nil {
		return nil
	}
	return proto.Clone(crawlLog).(*logV1.CrawlLog)
}

func registerResource(ctx context.Context, conn *serviceconnections.Connections, request *browsercontrollerV2.RegisterResourceRequest) (*browsercontrollerV2.RegisterResourceReply, error) {
	rpcCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if span := opentracing.SpanFromContext(ctx); span != nil {
		rpcCtx = opentracing.ContextWithSpan(rpcCtx, span)
	}

	reply, err := conn.BrowserControllerClient().RegisterResource(rpcCtx, request)
	if err != nil {
		return nil, errors.WrapInternalError(err, errors.RuntimeException, "Error register with browser controller", err.Error())
	}
	return reply, nil
}

func completeResource(ctx context.Context, conn *serviceconnections.Connections, request *browsercontrollerV2.CompleteResourceRequest) error {
	rpcCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if span := opentracing.SpanFromContext(ctx); span != nil {
		rpcCtx = opentracing.ContextWithSpan(rpcCtx, span)
	}

	_, err := conn.BrowserControllerClient().CompleteResource(rpcCtx, request)
	if err != nil {
		return errors.WrapInternalError(err, errors.RuntimeException, "error sending crawl log to browser controller", err.Error())
	}
	return nil
}

func applyRegisteredState(ctx context.Context, registered *browsercontrollerV2.ResourceRegistered) {
	if registered == nil {
		return
	}

	SetJobExecutionId(ctx, registered.JobExecutionId)
	SetCrawlExecutionId(ctx, registered.CrawlExecutionId)
	SetCollectionRef(ctx, registered.CollectionRef)
}

func (rc *RecordContext) shouldBypassBrowserControllerRegister() bool {
	return rc.ProxyId == 0 && rc.HasExplicitHarvestHeaders
}

func (rc *RecordContext) shouldSkipBrowserControllerComplete(cl *logV1.CrawlLog) bool {
	return rc.ProxyId != 0 && rc.RequestId == "" && cl != nil && cl.StatusCode == int32(errors.CanceledByBrowser)
}

func (rc *RecordContext) finalizeCrawlLog(cl *logV1.CrawlLog) error {
	rc.mutex.Lock()
	if rc.done {
		rc.mutex.Unlock()
		return AlreadyCompleted
	}
	rc.done = true
	rc.mutex.Unlock()
	defer rc.closeSession()

	cl.FetchTimeMs = time.Since(rc.FetchTimesTamp).Nanoseconds() / 1000000
	cl.IpAddress = rc.IpAddress
	if rc.shouldSkipBrowserControllerComplete(cl) {
		return nil
	}

	request := &browsercontrollerV2.CompleteResourceRequest{
		ProxyId:   rc.ProxyId,
		RequestId: rc.RequestId,
		CrawlLog:  cl,
		Cached:    rc.FoundInCache,
	}

	if span := opentracing.SpanFromContext(rc.ctx); span != nil {
		span.LogFields(
			otLog.String("event", "CompleteResource"),
			otLog.Int32("ProxyId", rc.ProxyId),
			otLog.String("Uri", cl.RequestedUri),
			otLog.String("RequestId", request.RequestId),
		)
	}

	return completeResource(rc.ctx, rc.conn, request)
}

func (rc *RecordContext) SaveCrawlLog() error {
	return rc.finalizeCrawlLog(cloneCrawlLog(rc.CrawlLog))
}

func (rc *RecordContext) SendRequestError(ctx context.Context, reqErr error) error {
	l := LogWithContext(rc.ctx, "PROXY:BCC")

	if reqErr == nil {
		l.Panic("BUG: SendRequestError with nil error")
	}

	rc.CrawlLog.StatusCode = int32(errors.Code(reqErr))
	rc.CrawlLog.RecordType = constants.RecordResponse
	rc.CrawlLog.Error = errors.AsCommonsError(reqErr)

	if err := rc.finalizeCrawlLog(cloneCrawlLog(rc.CrawlLog)); err != nil && !errors2.Is(err, AlreadyCompleted) {
		return err
	}
	return reqErr
}

func (rc *RecordContext) SendResponseError(ctx context.Context, respErr error) error {
	l := LogWithContext(rc.ctx, "PROXY:BCC")

	if respErr == nil {
		l.Panic("BUG: SendResponseError with nil error")
	}

	rc.CrawlLog.StatusCode = int32(errors.Code(respErr))
	rc.CrawlLog.RecordType = constants.RecordResponse
	rc.CrawlLog.ContentType = ""
	rc.CrawlLog.Error = errors.AsCommonsError(respErr)

	if err := rc.finalizeCrawlLog(cloneCrawlLog(rc.CrawlLog)); err != nil && !errors2.Is(err, AlreadyCompleted) {
		return err
	}
	return respErr
}

func (rc *RecordContext) RegisterNewRequest(ctx context.Context) error {
	l := LogWithContext(rc.ctx, "PROXY:BCC")

	if rc.shouldBypassBrowserControllerRegister() {
		rc.ReplacementScript = nil
		return nil
	}

	request := &browsercontrollerV2.RegisterResourceRequest{
		ProxyId:          rc.ProxyId,
		Method:           rc.Method,
		Uri:              rc.Uri.String(),
		RequestId:        rc.RequestId,
		CrawlExecutionId: rc.CrawlExecutionId,
		JobExecutionId:   rc.JobExecutionId,
		CollectionRef:    rc.CollectionRef,
	}

	if span := opentracing.SpanFromContext(rc.ctx); span != nil {
		fields := []otLog.Field{
			otLog.String("event", "Send RegisterResource request"),
			otLog.Int32("ProxyId", rc.ProxyId),
			otLog.String("Uri", rc.Uri.String()),
			otLog.String("RequestId", request.RequestId),
			otLog.String("CrawlExecutionId", request.CrawlExecutionId),
			otLog.String("JobExecutionId", request.JobExecutionId),
		}
		if request.CollectionRef != nil {
			fields = append(fields, otLog.String("CollectionId", request.CollectionRef.Id))
		} else {
			fields = append(fields, otLog.String("CollectionId", ""))
		}
		span.LogFields(fields...)
	}

	reply, err := registerResource(rc.ctx, rc.conn, request)
	if err != nil {
		l.WithError(err).Info("Error register with browser controller")
		return err
	}

	switch result := reply.Result.(type) {
	case *browsercontrollerV2.RegisterResourceReply_Cancel:
		if result.Cancel == "Blocked by robots.txt" {
			rc.PrecludedByRobots = true
			return errors.Error(errors.PrecludedByRobots, "PRECLUDED_BY_ROBOTS", "Robots.txt rules precluded fetch")
		}
		return errors.Error(errors.CanceledByBrowser, "CANCELLED_BY_BROWSER", result.Cancel)
	case *browsercontrollerV2.RegisterResourceReply_Registered:
		applyRegisteredState(rc.ctx, result.Registered)
		rc.ReplacementScript = nil
		rc.CrawlExecutionId = result.Registered.CrawlExecutionId
		rc.JobExecutionId = result.Registered.JobExecutionId
		rc.CollectionRef = result.Registered.CollectionRef
		rc.CrawlLog.JobExecutionId = rc.JobExecutionId
		rc.CrawlLog.ExecutionId = rc.CrawlExecutionId
		return nil
	default:
		return errors.Error(errors.RuntimeException, "INVALID_BROWSER_CONTROLLER_REPLY", "Browser controller returned an invalid register reply")
	}
}

func RegisterConnectRequest(ctx context.Context, conn *serviceconnections.Connections, proxyId int32, req *http.Request, uri *url.URL) {
	l := LogWithContext(ctx, "PROXY:BCC")

	resolveIdsFromHttpHeader(ctx, req)

	reply, err := registerResource(ctx, conn, &browsercontrollerV2.RegisterResourceRequest{
		ProxyId:          proxyId,
		Method:           http.MethodConnect,
		Uri:              uri.String(),
		RequestId:        GetRequestId(ctx),
		CrawlExecutionId: GetCrawlExecutionId(ctx),
		JobExecutionId:   GetJobExecutionId(ctx),
		CollectionRef:    GetCollectionRef(ctx),
	})
	if err != nil {
		l.WithError(err).Warn("Error registering CONNECT request with browser controller")
		return
	}

	switch result := reply.Result.(type) {
	case *browsercontrollerV2.RegisterResourceReply_Registered:
		applyRegisteredState(ctx, result.Registered)
	case *browsercontrollerV2.RegisterResourceReply_Cancel:
		l.Infof("CONNECT request canceled by browser controller: %s", result.Cancel)
	}
}
