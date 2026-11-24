/*
 * Copyright 2020 National Library of Norway.
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

package session

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	"github.com/chromedp/cdproto"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

func (sess *Session) initListeners(ctx context.Context) error {
	if err := chromedp.Do(ctx); err != nil {
		return err
	}
	ctx = context.WithValue(ctx, harvestLogContextKey{}, sess.logger)
	currentTargetID := target.ID(targetIDFromContext(ctx))

	listenerID := sess.listenerSeq.Add(1)
	sess.logger.Log(context.Background(), LevelTrace, "Registering target listener",
		"listenerId", listenerID,
		"targetId", currentTargetID)
	return listenHarvestTarget(ctx, sess.listenFunc(ctx, listenerID))
}

func (sess *Session) listenFunc(ctx context.Context, listenerID int64) func(ev any) {
	return func(ev any) {
		switch ev := ev.(type) {

		case *fetch.EventRequestPaused:
			go sess.continueDocumentRequest(ctx, ev)

		case *network.EventRequestWillBeSent:
			sess.onNetworkEventRequestWillBeSent(ctx, ev, listenerID)

		case *network.EventLoadingFinished:
			sess.onNetworkEventLoadingFinished(ctx, ev, listenerID)

		case *network.EventDataReceived:
			sess.onNetworkEventDataReceived(ctx, ev, listenerID)

		case *network.EventResponseReceived:
			sess.onNetworkEventResponseReceived(ctx, ev, listenerID)

		case *network.EventLoadingFailed:
			sess.onNetworkEventLoadingFailed(ctx, ev, listenerID)

		case *page.EventLifecycleEvent:
			sess.pageLoad.observe(ev)
		case *page.EventFrameNavigated:
			sess.pageLoad.navigated(ev.Frame)

		case *page.EventFileChooserOpened:
			sess.onPageEventFileChooserOpened(ctx, ev, listenerID)

		case *page.EventJavascriptDialogOpening:
			go sess.onPageEventJavascriptDialogOpening(ctx, ev, listenerID)

		case *target.EventAttachedToTarget:
			sess.onTargetAttached(ctx, ev, listenerID)

		case *target.EventDetachedFromTarget:
			sess.detachChild(ev.SessionID)

		}
	}
}

// CDP request IDs are only unique within their target session.
func observedRequestID(ctx context.Context, id network.RequestID) network.RequestID {
	if session := sessionIDFromContext(ctx); session != "" {
		return network.RequestID(string(session) + ":" + string(id))
	}
	return id
}

func (sess *Session) onNetworkEventRequestWillBeSent(ctx context.Context, ev *network.EventRequestWillBeSent, _ int64) {
	sess.networkTracker.noteObservedRequestStart(ev.Type, ev.Request.URL, ev.FrameID != "" || ev.LoaderID != "", observedRequestID(ctx, ev.RequestID))
	sess.observeRequest(ctx, ev)
}
func (sess *Session) onNetworkEventLoadingFinished(ctx context.Context, ev *network.EventLoadingFinished, _ int64) {
	sess.networkTracker.noteRequestDone(observedRequestID(ctx, ev.RequestID))
	sess.observeRequestEnd(ctx, ev.RequestID, "", false)
}
func (sess *Session) onNetworkEventDataReceived(ctx context.Context, ev *network.EventDataReceived, _ int64) {
	sess.networkTracker.noteRequestData(observedRequestID(ctx, ev.RequestID))
	sess.observeData(ctx, ev.RequestID)
}
func (sess *Session) onNetworkEventResponseReceived(ctx context.Context, ev *network.EventResponseReceived, _ int64) {
	sess.networkTracker.noteRequestData(observedRequestID(ctx, ev.RequestID))
	sess.observeResponse(ctx, ev)
}
func (sess *Session) onNetworkEventLoadingFailed(ctx context.Context, ev *network.EventLoadingFailed, _ int64) {
	sess.networkTracker.noteRequestDone(observedRequestID(ctx, ev.RequestID))
	sess.observeRequestEnd(ctx, ev.RequestID, ev.ErrorText, ev.Canceled)
}

func (sess *Session) onPageEventFileChooserOpened(_ context.Context, ev *page.EventFileChooserOpened, listenerID int64) {
	sess.logger.Warn("File chooser opened",
		"listenerId", listenerID,
		"backendNodeId", ev.BackendNodeID,
		"frameId", ev.FrameID,
		"mode", ev.Mode)
}

func (sess *Session) onPageEventJavascriptDialogOpening(ctx context.Context, ev *page.EventJavascriptDialogOpening, listenerID int64) {
	log := sess.logger.With(
		"listenerID", listenerID,
		"url", ev.URL,
	)
	log.Log(context.Background(), LevelTrace, "Javascript dialog opening", "message", ev.Message)
	accept := ev.Type == "alert"
	if _, err := callTarget(ctx, page.HandleJavaScriptDialog, page.HandleJavaScriptDialogParams{Accept: accept}); err != nil {
		log.Error("Failed to handle JavaScript dialog", "error", err)
	}
}

func targetIDFromContext(ctx context.Context) string {
	if c := chromedp.FromContext(ctx); c != nil && c.Target != nil {
		return c.Target.TargetID.String()
	}
	return ""
}

func harvestAutoAttach() target.SetAutoAttachParams {
	// Configure the attempt token before a new target can issue requests.
	return target.SetAutoAttachParams{AutoAttach: true, WaitForDebuggerOnStart: true, Flatten: new(true), Filter: target.Filter{
		{Type: "iframe"}, {Type: "page"}, {Type: "worker"}, {Type: "shared_worker"}, {Type: "service_worker"}, {Exclude: true},
	}}
}

// configureHarvestTarget gives root and child targets the same attempt identity.
// Child targets remain paused until this succeeds.
func (sess *Session) configureHarvestTarget(ctx context.Context) error {
	return chromedp.Do(ctx,
		cdpAction(network.Enable, network.EnableParams{}),
		cdpAction(network.SetCacheDisabled, network.SetCacheDisabledParams{CacheDisabled: true}),
		cdpAction(network.SetExtraHTTPHeaders, network.SetExtraHTTPHeadersParams{Headers: network.Headers{constants.HeaderAttemptId: sess.attemptID}}),
		cdpAction(target.SetAutoAttach, harvestAutoAttach()),
	)
}

// No request registry is needed: the frame identity travels with the HTTP request.
func (sess *Session) documentRequestHeaders(ev *fetch.EventRequestPaused) []*fetch.HeaderEntry {
	headers := make([]*fetch.HeaderEntry, 0, len(ev.Request.Headers)+1)
	for name, value := range ev.Request.Headers {
		if !strings.EqualFold(name, constants.HeaderMainDocument) {
			headers = append(headers, &fetch.HeaderEntry{Name: name, Value: fmt.Sprint(value)})
		}
	}
	if ev.ResourceType == network.ResourceTypeDocument && sess.pageLoad.isRoot(ev.FrameID) {
		headers = append(headers, &fetch.HeaderEntry{Name: constants.HeaderMainDocument, Value: sess.attemptID})
	}
	return headers
}

func (sess *Session) continueDocumentRequest(ctx context.Context, ev *fetch.EventRequestPaused) {
	_, err := callTarget(ctx, fetch.ContinueRequest, fetch.ContinueRequestParams{
		RequestID: ev.RequestID, Headers: sess.documentRequestHeaders(ev),
	})
	if err == nil || ctx.Err() != nil || expiredInterception(err) {
		return
	}
	sess.browserCancel(fmt.Errorf("continue document request: %w", err))
}

func expiredInterception(err error) bool {
	var protocolError *cdproto.Error
	return errors.As(err, &protocolError) && protocolError.Code == -32602 && strings.Contains(protocolError.Message, "Invalid InterceptionId")
}
