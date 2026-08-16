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
	"fmt"
	"strings"

	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/requests"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/url"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/security"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

func (sess *Session) initListeners(ctx context.Context) {
	currentTargetID := target.ID(targetIDFromContext(ctx))
	sess.rootTargetID = currentTargetID
	if currentTargetID != "" {
		sess.markTargetInitialized(currentTargetID)
	}

	listenerID := sess.listenerSeq.Add(1)
	sess.loggerOrDefault().Debug("Registering target listener",
		"listenerId", listenerID,
		"targetId", currentTargetID)
	chromedp.ListenTarget(ctx, sess.listenFunc(ctx, listenerID))
}

func (sess *Session) listenFunc(ctx context.Context, listenerID int64) func(ev any) {
	return func(ev any) {
		switch ev := ev.(type) {

		case *network.EventRequestWillBeSent:
			sess.onNetworkEventRequestWillBeSent(ctx, ev, listenerID)

		case *network.EventLoadingFinished:
			sess.onNetworkEventLoadingFinished(ctx, ev, listenerID)

		case *network.EventDataReceived:
			sess.onNetworkEventDataReceived(ctx, ev, listenerID)

		case *network.EventLoadingFailed:
			sess.onNetworkEventLoadingFailed(ctx, ev, listenerID)

		case *page.EventFrameStartedLoading:
			sess.onPageEventFrameStartedLoading(ctx, ev, listenerID)

		case *page.EventFrameStoppedLoading:
			sess.onPageEventFrameStoppedLoading(ctx, ev, listenerID)

		case *page.EventFrameSubtreeWillBeDetached:
			sess.onPageEventFrameSubtreeWillBeDetached(ctx, ev, listenerID)

		case *page.EventFrameDetached:
			sess.onPageEventFrameDetached(ctx, ev, listenerID)

		case *page.EventFileChooserOpened:
			sess.onPageEventFileChooserOpened(ctx, ev, listenerID)

		case *page.EventJavascriptDialogOpening:
			go sess.onPageEventJavascriptDialogOpening(ctx, ev, listenerID)

		case *target.EventTargetCreated:
			sess.onTargetEventTargetCreated(ctx, ev, listenerID)

		case *fetch.EventRequestPaused:
			go sess.onFetchEventRequestPaused(ctx, ev, listenerID)
		}
	}
}

func (sess *Session) onNetworkEventRequestWillBeSent(_ context.Context, ev *network.EventRequestWillBeSent, listenerID int64) {
	id := logicalIDFromNetworkWillBeSent(ev)
	loggedURL, urlLength := requests.BoundedURLForLog(ev.Request.URL)
	documentURL, documentURLLength := requests.BoundedURLForLog(ev.DocumentURL)
	redirectURL, redirectURLLength := requests.BoundedURLForLog(redirectFromURL(ev))

	log := sess.loggerOrDefault().With(
		"id", id,
		"listenerID", listenerID,
		"resourceType", ev.Type.String(),
		"frameID", string(ev.FrameID),
		"initiator", ev.Initiator.Type.String(),
		"loaderID", string(ev.LoaderID),
		"documentURL", documentURL,
		"documentURLLength", documentURLLength,
		"redirectFromURL", redirectURL,
		"redirectFromURLLength", redirectURLLength,
		"url", loggedURL,
		"urlLength", urlLength,
		"method", ev.Request.Method,
	)

	if id == "" {
		log.Warn("Network event has no id")
		return
	}

	if !sess.acceptingRequests() {
		return
	}
	if url.IsBrowserLocal(ev.Request.URL) {
		log.Debug("Ignoring browser-local network request")
		return
	}

	_, added := sess.Requests.GetOrAddRequest(requestFromNetworkWillBeSent(ev))
	if added {
		sess.networkTracker.noteRequestStart(ev)
	}
}

func (sess *Session) onNetworkEventLoadingFinished(_ context.Context, ev *network.EventLoadingFinished, listenerID int64) {
	sess.networkTracker.noteRequestDone(ev.RequestID)

	id := string(ev.RequestID)

	_ = sess.Requests.GotComplete(id)
}

func (sess *Session) onNetworkEventDataReceived(_ context.Context, ev *network.EventDataReceived, _ int64) {
	sess.networkTracker.noteRequestData(ev.RequestID)
}

func (sess *Session) onNetworkEventLoadingFailed(_ context.Context, ev *network.EventLoadingFailed, listenerID int64) {
	sess.networkTracker.noteRequestDone(ev.RequestID)

	log := sess.loggerOrDefault().With(
		"listenerId", listenerID,
		"requestID", ev.RequestID,
		"type", ev.Type,
		"errorText", ev.ErrorText,
		"blockedReason", ev.BlockedReason,
		"canceled", ev.Canceled,
	)

	id := string(ev.RequestID)

	// TODO should it be completed when failed ?
	req := sess.Requests.GotComplete(id)
	if req != nil {
		loggedURL, urlLength := requests.BoundedURLForLog(req.URL)
		log = log.With("url", loggedURL, "urlLength", urlLength)
	}
	log.Log(context.Background(), resourceFailureLogLevel(req, ev.Type.String()), "Loading failed")
}

func (sess *Session) onPageEventFrameStartedLoading(ctx context.Context, ev *page.EventFrameStartedLoading, listenerID int64) {
	if !sess.shouldTrackFrameLifecycle(ctx) {
		return
	}
	tracked := sess.frameLoads.Start(string(ev.FrameID))
	sess.loggerOrDefault().Debug("Tracked frame started loading",
		"listenerId", listenerID,
		"targetId", targetIDFromContext(ctx),
		"frameId", string(ev.FrameID),
		"tracked", tracked)
}

func (sess *Session) onPageEventFrameStoppedLoading(ctx context.Context, ev *page.EventFrameStoppedLoading, listenerID int64) {
	if !sess.shouldTrackFrameLifecycle(ctx) {
		return
	}
	tracked := sess.frameLoads.Finish(string(ev.FrameID))
	message := "Tracked frame stopped loading"
	if !tracked {
		message = "Tracked frame stopped loading without prior start"
	}
	sess.loggerOrDefault().Debug(message,
		"listenerId", listenerID,
		"targetId", targetIDFromContext(ctx),
		"frameId", string(ev.FrameID),
		"tracked", tracked)
}

func (sess *Session) onPageEventFrameDetached(ctx context.Context, ev *page.EventFrameDetached, listenerID int64) {
	log := sess.loggerOrDefault().With("listenerID", listenerID)

	log.Debug("Frame detached", "frameId", string(ev.FrameID))
	if !sess.shouldTrackFrameLifecycle(ctx) {
		return
	}
	tracked := sess.frameLoads.Finish(string(ev.FrameID))
	log.Debug("Tracked frame detached",
		"listenerId", listenerID,
		"targetId", targetIDFromContext(ctx),
		"frameId", string(ev.FrameID),
		"reason", string(ev.Reason),
		"tracked", tracked)
}

func (sess *Session) onPageEventFrameSubtreeWillBeDetached(ctx context.Context, ev *page.EventFrameSubtreeWillBeDetached, listenerID int64) {
	log := sess.loggerOrDefault().With("listenerID", listenerID)

	log.Debug("Frame subtree will be detached", "frameId", string(ev.FrameID))
	if !sess.shouldTrackFrameLifecycle(ctx) {
		return
	}
	tracked := sess.frameLoads.Finish(string(ev.FrameID))
	log.Debug("Tracked frame subtree will be detached",
		"listenerId", listenerID,
		"targetId", targetIDFromContext(ctx),
		"frameId", string(ev.FrameID),
		"tracked", tracked)
}

func (sess *Session) onPageEventFileChooserOpened(_ context.Context, ev *page.EventFileChooserOpened, listenerID int64) {
	sess.loggerOrDefault().Warn("File chooser opened",
		"listenerId", listenerID,
		"backendNodeId", ev.BackendNodeID,
		"frameId", ev.FrameID,
		"mode", ev.Mode)
}

func (sess *Session) onPageEventJavascriptDialogOpening(ctx context.Context, ev *page.EventJavascriptDialogOpening, listenerID int64) {
	log := sess.loggerOrDefault().With(
		"listenerID", listenerID,
		"url", ev.URL,
	)
	log.Debug("Javascript dialog opening", "message", ev.Message)
	accept := ev.Type == "alert"
	if err := chromedp.Run(ctx, page.HandleJavaScriptDialog(accept)); err != nil {
		log.Error("Failed to handle JavaScript dialog", "error", err)
	}
}

func (sess *Session) onTargetEventTargetCreated(ctx context.Context, ev *target.EventTargetCreated, listenerID int64) {
	log := sess.loggerOrDefault().With("listenerID", listenerID)

	log.Debug("Target created",
		"targetId", ev.TargetInfo.TargetID.String(),
		"openerId", ev.TargetInfo.OpenerID.String(),
		"browserContextId", string(ev.TargetInfo.BrowserContextID),
		"targetType", ev.TargetInfo.Type,
		"title", ev.TargetInfo.Title,
		"url", ev.TargetInfo.URL,
		"attached", ev.TargetInfo.Attached)

	if c := chromedp.FromContext(ctx); c != nil && c.Target != nil && ev.TargetInfo.TargetID == c.Target.TargetID {
		log.Debug("Skipping current target initialization", "targetId", ev.TargetInfo.TargetID.String(), "targetType", ev.TargetInfo.Type)
		return
	}
	if !sess.markTargetInitialized(ev.TargetInfo.TargetID) {
		log.Debug("Skipping duplicate target initialization", "targetId", ev.TargetInfo.TargetID.String(), "targetType", ev.TargetInfo.Type)
		return
	}
	if ev.TargetInfo.Type == "worker" {
		err := sess.SignalCompletionActivity()
		if err != nil {
			log.Error("Failed to notify session of new target", "error", err)
		}
		return
	}
	if !shouldInitChildTarget(ev.TargetInfo.Type) {
		log.Debug("Ignoring unsupported child target type", "targetId", ev.TargetInfo.TargetID.String(), "targetType", ev.TargetInfo.Type)
		return
	}

	sess.initChildTarget(ctx, ev.TargetInfo.TargetID, ev.TargetInfo.Type)
	err := sess.SignalCompletionActivity()
	if err != nil {
		log.Error("Failed to notify session of new target", "error", err)
	}
}

func (sess *Session) onFetchEventRequestPaused(ctx context.Context, ev *fetch.EventRequestPaused, listenerID int64) {
	if !sess.acceptingRequests() {
		sess.quiescePausedRequest(ctx, ev)
		return
	}

	id := logicalIDFromFetchPaused(ev)
	loggedURL, urlLength := requests.BoundedURLForLog(ev.Request.URL)

	log := sess.loggerOrDefault().With(
		"listenerID", listenerID,
		"id", id,
		"fetchRequestID", ev.RequestID,
		"networkID", ev.NetworkID,
		"targetID", targetIDFromContext(ctx),
		"url", loggedURL,
		"urlLength", urlLength,
		"method", ev.Request.Method,
		"resourceType", ev.ResourceType.String(),
	)

	isResponseStage := ev.ResponseStatusCode != 0 || ev.ResponseErrorReason != ""
	if isResponseStage {
		if err := sess.continuePausedRequest(ctx, ev, nil); err != nil {
			log.Warn("Failed to continue paused request",
				"statusCode", ev.ResponseStatusCode,
				"errorReason", ev.ResponseErrorReason,
				"error", err)
		}
		return
	}

	sess.networkTracker.noteObservedRequestStart(
		ev.ResourceType,
		ev.Request.URL,
		ev.FrameID != "",
		ev.NetworkID,
	)

	browserLocal := url.IsBrowserLocal(ev.Request.URL)
	var req *requests.Request
	added := false
	if !browserLocal {
		req, added = sess.Requests.GetOrAddRequest(requestFromFetchPaused(ev))
	}

	if err := sess.continuePausedRequest(ctx, ev, req); err != nil {
		rolledBack := sess.rollbackPausedRequest(req, added)
		log.Warn("Rolled back paused request after continue failure",
			"rolledBack", rolledBack,
			"added", added,
			"error", err)

		sess.networkTracker.noteRequestDone(ev.NetworkID)

		if rolledBack && isInvalidInterceptionIDError(err) {
			log.Debug("Ignored continue failure for rolled-back paused request", "error", err)
		}

		return
	}
	if browserLocal {
		log.Debug("Continued browser-local request without registering it")
		return
	}

	if req != nil {
		if err := sess.SignalRequestActivity(req); err != nil {
			log.Error("Failed to notify session after request continuation",
				"id", req.ID,
				"fetchRequestID", req.FetchRequestID,
				"networkID", req.NetworkID,
				"error", err)
		}
		return
	}

	if err := sess.SignalCompletionActivity(); err != nil {
		log.Error("Failed to notify session after request continuation",
			"error", err)
	}
}

func (sess *Session) continuePausedRequest(
	ctx context.Context,
	ev *fetch.EventRequestPaused,
	req *requests.Request,
) error {
	continueRequest := fetch.ContinueRequest(ev.RequestID)

	// Preserve original URL/method if you really need to. If you are not modifying
	// them, this is optional. Keeping it here matches your current behavior.
	continueRequest = continueRequest.
		WithURL(ev.Request.URL).
		WithMethod(ev.Request.Method)

	headers := buildFetchHeaders(ev, req)
	if len(headers) > 0 {
		continueRequest = continueRequest.WithHeaders(headers)
	}

	return chromedp.Run(ctx, continueRequest)
}

func buildFetchHeaders(ev *fetch.EventRequestPaused, req *requests.Request) []*fetch.HeaderEntry {
	headerCount := len(ev.Request.Headers)

	if req != nil && req.ID != "" {
		headerCount++
	}

	headers := make([]*fetch.HeaderEntry, 0, headerCount)

	for k, v := range ev.Request.Headers {
		if strings.EqualFold(k, "veidemann_reqid") {
			continue
		}

		headers = append(headers, &fetch.HeaderEntry{
			Name:  k,
			Value: interfaceToString(v),
		})
	}

	if req != nil && req.ID != "" {
		headers = append(headers, &fetch.HeaderEntry{
			Name:  "veidemann_reqid",
			Value: req.ID,
		})
	}

	return headers
}

func (sess *Session) quiescePausedRequest(ctx context.Context, ev *fetch.EventRequestPaused) {
	if err := sess.finalizePausedRequest(ctx, ev); err != nil {
		sess.loggerOrDefault().Debug("Failed to quiesce paused request",
			"error", err,
			"targetID", targetIDFromContext(ctx))
	}
}

func (sess *Session) shouldTrackFrameLifecycle(ctx context.Context) bool {
	return sess.shouldTrackFrameLifecycleTarget(target.ID(targetIDFromContext(ctx)))
}

func (sess *Session) shouldTrackFrameLifecycleTarget(targetID target.ID) bool {
	return targetID != "" && targetID == sess.rootTargetID
}

func targetIDFromContext(ctx context.Context) string {
	if c := chromedp.FromContext(ctx); c != nil && c.Target != nil {
		return c.Target.TargetID.String()
	}
	return ""
}

func (sess *Session) markTargetInitialized(targetID target.ID) bool {
	sess.initializedTargetsMu.Lock()
	defer sess.initializedTargetsMu.Unlock()

	if sess.initializedTargets == nil {
		sess.initializedTargets = make(map[target.ID]struct{})
	}
	if _, ok := sess.initializedTargets[targetID]; ok {
		return false
	}

	sess.initializedTargets[targetID] = struct{}{}
	return true
}

func (sess *Session) childTargetActions(targetType string) []chromedp.Action {
	actions := []chromedp.Action{
		fetch.Enable(),
		runtime.Enable(),
		target.SetAutoAttach(true, false).WithFlatten(true),
		runtime.RunIfWaitingForDebugger(),
		network.Enable(),
		network.SetCacheDisabled(true),
		network.SetCookies(sess.getCookieParams(sess.RequestedUrl)),
	}

	if targetType != "shared_worker" && targetType != "service_worker" {
		actions = append(actions,
			page.Enable(),
			security.SetIgnoreCertificateErrors(true),
		)
	}

	return actions
}

func shouldInitChildTarget(targetType string) bool {
	switch targetType {
	case "iframe":
		return true
	default:
		return false
	}
}

func (sess *Session) initChildTarget(ctx context.Context, targetID target.ID, targetType string) {
	newCtx, _ := chromedp.NewContext(ctx, chromedp.WithTargetID(targetID))
	go func() {
		<-ctx.Done()
		_ = chromedp.Cancel(newCtx)
	}()

	actions := sess.childTargetActions(targetType)
	go func() {
		listenerID := sess.listenerSeq.Add(1)
		sess.loggerOrDefault().Debug("Registering child target listener", "listenerId", listenerID, "targetId", targetID.String(), "targetType", targetType)
		chromedp.ListenTarget(newCtx, sess.listenFunc(newCtx, listenerID))

		if err := chromedp.Run(newCtx, actions...); err != nil {
			sess.loggerOrDefault().Error("Failed initializing new target", "targetType", targetType, "error", err)
		}
	}()
}

func (sess *Session) finalizePausedRequest(ctx context.Context, ev *fetch.EventRequestPaused) error {
	if ev.ResponseStatusCode != 0 || ev.ResponseErrorReason != "" {
		return chromedp.Run(ctx, fetch.ContinueRequest(ev.RequestID))
	}
	return chromedp.Run(ctx, fetch.FailRequest(ev.RequestID, network.ErrorReasonAborted))
}

func isInvalidInterceptionIDError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Invalid InterceptionId")
}

func (sess *Session) rollbackPausedRequest(req *requests.Request, added bool) bool {
	if !added || req == nil || sess.Requests == nil {
		return false
	}

	if !sess.Requests.RemoveRequest(req) {
		return false
	}

	if req.BlocksPageCompletion() {
		_ = sess.SignalCompletionActivity()
	}

	return true
}

func interfaceToString(i any) string {
	if i == nil {
		return ""
	}
	return fmt.Sprintf("%v", i)
}

func logicalIDFromFetchPaused(ev *fetch.EventRequestPaused) string {
	id := string(ev.NetworkID)
	if id == "" {
		id = string(ev.RequestID)
	}
	return id
}

func requestFromFetchPaused(ev *fetch.EventRequestPaused) *requests.Request {
	id := logicalIDFromFetchPaused(ev)

	return &requests.Request{
		ID:             id,
		FetchRequestID: string(ev.RequestID),
		NetworkID:      string(ev.NetworkID),
		URL:            ev.Request.URL,
		Method:         ev.Request.Method,
		ResourceType:   string(ev.ResourceType),
	}
}

func logicalIDFromNetworkWillBeSent(ev *network.EventRequestWillBeSent) string {
	return string(ev.RequestID)
}

func requestFromNetworkWillBeSent(ev *network.EventRequestWillBeSent) *requests.Request {
	id := logicalIDFromNetworkWillBeSent(ev)

	req := &requests.Request{
		ID:           id,
		NetworkID:    id,
		URL:          url.Normalize(ev.Request.URL),
		Method:       ev.Request.Method,
		ResourceType: ev.Type.String(),
		Initiator:    ev.Initiator.Type.String(),
		Referrer:     interfaceToString(ev.Request.Headers["Referer"]),
	}

	if redirectUrl := redirectFromURL(ev); redirectUrl != "" {
		req.Redirected = true
		req.RedirectFromURL = redirectUrl
	}

	return req
}

func redirectFromURL(ev *network.EventRequestWillBeSent) string {
	if ev == nil || ev.RedirectResponse == nil {
		return ""
	}
	return ev.RedirectResponse.URL
}
