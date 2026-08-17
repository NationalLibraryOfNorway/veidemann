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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	browsercontrollerV2 "github.com/NationalLibraryOfNorway/veidemann/api/browsercontroller/v2"
	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	rperrors "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/serviceconnections"
	"google.golang.org/protobuf/proto"
)

var AlreadyCompleted = errors.New("already completed")

func cloneCrawlLog(crawlLog *logV1.CrawlLog) *logV1.CrawlLog {
	if crawlLog == nil {
		return nil
	}
	return proto.Clone(crawlLog).(*logV1.CrawlLog)
}

// UpdateCrawlLog serializes mutations with terminal snapshotting.
func (rc *RecordContext) UpdateCrawlLog(update func(*logV1.CrawlLog)) {
	rc.crawlLogMutex.Lock()
	defer rc.crawlLogMutex.Unlock()
	if rc.CrawlLog != nil {
		update(rc.CrawlLog)
	}
}

// CrawlLogSnapshot returns an independent snapshot for terminal processing.
func (rc *RecordContext) CrawlLogSnapshot() *logV1.CrawlLog {
	rc.crawlLogMutex.RLock()
	defer rc.crawlLogMutex.RUnlock()
	return cloneCrawlLog(rc.CrawlLog)
}

func (rc *RecordContext) HasCrawlLog() bool {
	rc.crawlLogMutex.RLock()
	defer rc.crawlLogMutex.RUnlock()
	return rc.CrawlLog != nil
}

func registerResource(ctx context.Context, conn *serviceconnections.Connections, request *browsercontrollerV2.RegisterResourceRequest) (*browsercontrollerV2.RegisterResourceReply, error) {
	rpcCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	reply, err := conn.BrowserControllerClient().RegisterResource(rpcCtx, request)
	if err != nil {
		return nil, rperrors.WrapInternalError(err, rperrors.RuntimeException, "Error register with browser controller", err.Error())
	}
	return reply, nil
}

func completeResource(_ context.Context, conn *serviceconnections.Connections, request *browsercontrollerV2.CompleteResourceRequest) error {
	rpcCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := conn.BrowserControllerClient().CompleteResource(rpcCtx, request)
	if err != nil {
		return rperrors.WrapInternalError(err, rperrors.RuntimeException, "error sending crawl log to browser controller", err.Error())
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
	return rc.ProxyId != 0 && rc.RequestId == "" && cl != nil && cl.StatusCode == int32(rperrors.CanceledByBrowser)
}

func (rc *RecordContext) finalizeCrawlLog(cl *logV1.CrawlLog) error {
	_, err := rc.coordinateTerminal(terminalRequest{crawlLog: cl})
	return err
}

type terminalRequest struct {
	crawlLog       *logV1.CrawlLog
	originalErr    error
	cancelMessage  string
	sendMeta       bool
	responseRecNum int32
	responseSize   int64
	blockDigest    string
}

func (rc *RecordContext) coordinateTerminal(terminal terminalRequest) (bool, error) {
	rc.mutex.Lock()
	if rc.done {
		done := rc.terminalDone
		rc.mutex.Unlock()
		<-done
		return false, AlreadyCompleted
	}
	rc.done = true
	rc.mutex.Unlock()

	var terminalErr error
	defer func() {
		rc.mutex.Lock()
		rc.terminalErr = terminalErr
		rc.mutex.Unlock()
		rc.closeSession()
		close(rc.terminalDone)
	}()

	cl := cloneCrawlLog(terminal.crawlLog)
	if cl == nil {
		cl = rc.CrawlLogSnapshot()
	}
	if cl == nil {
		cl = &logV1.CrawlLog{}
	}

	if terminal.sendMeta {
		reply, err := rc.terminateContentWriter(true, "")
		if err == nil {
			err = applyContentWriterReply(cl, reply, terminal.responseRecNum, terminal.responseSize, terminal.blockDigest)
		}
		if err != nil {
			terminalErr = rperrors.WrapInternalError(
				err,
				rperrors.RuntimeException,
				"Error writing to content writer",
				err.Error(),
			)
		}
	} else if terminal.cancelMessage != "" {
		_, err := rc.terminateContentWriter(false, terminal.cancelMessage)
		if err != nil {
			LogWithContext(rc.ctx, "PROXY:CWC").WithError(err).Warn("ContentWriter cancellation failed")
			if terminal.originalErr == nil {
				terminalErr = rperrors.WrapInternalError(
					err,
					rperrors.RuntimeException,
					"Error canceling content writer",
					err.Error(),
				)
			}
		}
	}

	authoritativeErr := terminal.originalErr
	if authoritativeErr == nil {
		authoritativeErr = terminalErr
	}
	if authoritativeErr != nil {
		cl.StatusCode = int32(rperrors.Code(authoritativeErr))
		cl.RecordType = constants.RecordResponse
		cl.ContentType = ""
		cl.Error = rperrors.AsCommonsError(authoritativeErr)
	}

	cl.FetchTimeMs = time.Since(rc.FetchTimesTamp).Nanoseconds() / 1000000
	cl.IpAddress = rc.IpAddress
	if !rc.shouldSkipBrowserControllerComplete(cl) {
		request := &browsercontrollerV2.CompleteResourceRequest{
			ProxyId:   rc.ProxyId,
			RequestId: rc.RequestId,
			CrawlLog:  cl,
			Cached:    rc.FoundInCache,
		}

		if err := completeResource(rc.ctx, rc.conn, request); err != nil {
			if terminalErr == nil {
				terminalErr = err
			} else {
				LogWithContext(rc.ctx, "PROXY:BCC").WithError(err).Warn("BrowserController completion failed after terminal error")
			}
		}
	}

	if terminal.originalErr != nil {
		return true, terminal.originalErr
	}
	return true, terminalErr
}

func applyContentWriterReply(cl *logV1.CrawlLog, reply *contentwriterV1.WriteReply, recNum int32, size int64, blockDigest string) error {
	if reply == nil || reply.GetMeta() == nil {
		return fmt.Errorf("content writer reply meta is nil")
	}
	meta, ok := reply.GetMeta().RecordMeta[recNum]
	if !ok || meta == nil {
		return fmt.Errorf("content writer reply missing record metadata for record %d", recNum)
	}

	cl.CollectionFinalName = meta.CollectionFinalName
	cl.WarcId = meta.WarcId
	cl.StorageRef = meta.StorageRef
	cl.WarcRefersTo = meta.RevisitReferenceId
	cl.Size = size
	cl.RecordType = strings.ToLower(meta.Type.String())
	cl.BlockDigest = blockDigest
	cl.PayloadDigest = meta.PayloadDigest
	return nil
}

func (rc *RecordContext) SaveCrawlLog() error {
	return rc.finalizeCrawlLog(rc.CrawlLogSnapshot())
}

// FinalizeStoredResponse terminates ContentWriter, validates its reply, and
// completes BrowserController as one idempotent operation.
func (rc *RecordContext) FinalizeStoredResponse(recNum int32, size int64, blockDigest string) error {
	_, err := rc.coordinateTerminal(terminalRequest{
		crawlLog:       rc.CrawlLogSnapshot(),
		sendMeta:       true,
		responseRecNum: recNum,
		responseSize:   size,
		blockDigest:    blockDigest,
	})
	return err
}

// FinalizeCachedResponse cancels the unused writer stream before completing
// BrowserController.
func (rc *RecordContext) FinalizeCachedResponse(cancelMessage string) error {
	_, err := rc.coordinateTerminal(terminalRequest{
		crawlLog:      rc.CrawlLogSnapshot(),
		cancelMessage: cancelMessage,
	})
	return err
}

func (rc *RecordContext) finalizeError(originalErr error, cancelMessage string) error {
	_, err := rc.coordinateTerminal(terminalRequest{
		crawlLog:      rc.CrawlLogSnapshot(),
		originalErr:   originalErr,
		cancelMessage: cancelMessage,
	})
	if errors.Is(err, AlreadyCompleted) {
		return originalErr
	}
	return err
}

func (rc *RecordContext) SendRequestError(ctx context.Context, reqErr error) error {
	l := LogWithContext(rc.ctx, "PROXY:BCC")

	if reqErr == nil {
		l.Panic("BUG: SendRequestError with nil error")
	}

	return rc.finalizeError(reqErr, rperrors.Detail(reqErr))
}

func (rc *RecordContext) SendResponseError(ctx context.Context, respErr error) error {
	l := LogWithContext(rc.ctx, "PROXY:BCC")

	if respErr == nil {
		l.Panic("BUG: SendResponseError with nil error")
	}

	return rc.finalizeError(respErr, rperrors.Detail(respErr))
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

	reply, err := registerResource(rc.ctx, rc.conn, request)
	if err != nil {
		l.WithError(err).Info("Error register with browser controller")
		return err
	}

	switch result := reply.Result.(type) {
	case *browsercontrollerV2.RegisterResourceReply_Cancel:
		if result.Cancel == "Blocked by robots.txt" {
			rc.PrecludedByRobots = true
			return rperrors.Error(rperrors.PrecludedByRobots, "PRECLUDED_BY_ROBOTS", "Robots.txt rules precluded fetch")
		}
		return rperrors.Error(rperrors.CanceledByBrowser, "CANCELED_BY_BROWSER", result.Cancel)
	case *browsercontrollerV2.RegisterResourceReply_Registered:
		applyRegisteredState(rc.ctx, result.Registered)
		rc.ReplacementScript = nil
		rc.CrawlExecutionId = result.Registered.CrawlExecutionId
		rc.JobExecutionId = result.Registered.JobExecutionId
		rc.CollectionRef = result.Registered.CollectionRef
		rc.UpdateCrawlLog(func(cl *logV1.CrawlLog) {
			cl.JobExecutionId = rc.JobExecutionId
			cl.ExecutionId = rc.CrawlExecutionId
		})
		return nil
	default:
		return rperrors.Error(rperrors.RuntimeException, "INVALID_BROWSER_CONTROLLER_REPLY", "Browser controller returned an invalid register reply")
	}
}

func RegisterConnectRequest(ctx context.Context, conn *serviceconnections.Connections, proxyId int32, req *http.Request, uri *url.URL) error {
	l := LogWithContext(ctx, "PROXY:BCC")

	resolveIdsFromHttpHeader(ctx, req)

	reply, err := registerResource(ctx, conn, &browsercontrollerV2.RegisterResourceRequest{
		ProxyId:   proxyId,
		Method:    http.MethodConnect,
		Uri:       uri.String(),
		RequestId: GetRequestId(ctx),
	})
	if err != nil {
		l.WithError(err).Warn("Error registering CONNECT request with browser controller")
		return err
	}

	switch result := reply.Result.(type) {
	case *browsercontrollerV2.RegisterResourceReply_Registered:
		applyRegisteredState(ctx, result.Registered)

	case *browsercontrollerV2.RegisterResourceReply_Cancel:
		l.Infof("CONNECT request canceled by browser controller: %s", result.Cancel)
		return &rperrors.BrowserControllerCancelError{Reason: result.Cancel}
	}

	return nil
}
