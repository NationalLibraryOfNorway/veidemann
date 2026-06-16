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
	"net/http"
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
			sess.onNetworkEventRequestWillBeSent(ctx, ev)

		case *network.EventLoadingFinished:
			sess.onNetworkEventLoadingFinished(ctx, ev)

		case *network.EventDataReceived:
			sess.onNetworkEventDataReceived(ctx, ev)

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

func (sess *Session) onNetworkEventRequestWillBeSent(_ context.Context, ev *network.EventRequestWillBeSent) {
	if sess.networkTracker != nil {
		sess.networkTracker.noteRequestStart(ev)
	}
	sess.loggerOrDefault().Debug("Request will be sent",
		"requestId", ev.RequestID.String(),
		"resourceType", ev.Type.String(),
		"frameId", string(ev.FrameID),
		"initiator", ev.Initiator.Type.String(),
		"loaderId", string(ev.LoaderID),
		"documentURL", ev.DocumentURL)
	if req := sess.Requests.GetByNetworkId(ev.RequestID.String()); req != nil {
		req.Initiator = ev.Initiator.Type.String()
	}
}

func (sess *Session) onNetworkEventLoadingFinished(_ context.Context, ev *network.EventLoadingFinished) {
	if sess.networkTracker != nil {
		sess.networkTracker.noteRequestDone(ev.RequestID)
	}
}

func (sess *Session) onNetworkEventDataReceived(_ context.Context, ev *network.EventDataReceived) {
	if sess.networkTracker != nil {
		sess.networkTracker.noteRequestData(ev.RequestID)
	}
}

func (sess *Session) onNetworkEventLoadingFailed(ctx context.Context, ev *network.EventLoadingFailed, listenerID int64) {
	if sess.networkTracker != nil {
		sess.networkTracker.noteRequestDone(ev.RequestID)
	}
	sess.loggerOrDefault().Debug("Loading failed",
		"type", ev.Type,
		"errorText", ev.ErrorText,
		"blockedReason", ev.BlockedReason,
		"canceled", ev.Canceled,
		"requestID", ev.RequestID)
}

func (sess *Session) onPageEventFrameStartedLoading(ctx context.Context, ev *page.EventFrameStartedLoading, listenerID int64) {
	if !sess.shouldTrackFrameLifecycle(ctx) {
		return
	}
	previousCount, currentCount := sess.noteFrameLoadStart(string(ev.FrameID))
	counted := previousCount == 0 && currentCount > 0
	sess.loggerOrDefault().Debug("Tracked frame started loading",
		"listenerId", listenerID,
		"targetId", targetIDFromContext(ctx),
		"frameId", string(ev.FrameID),
		"previousCount", previousCount,
		"counted", counted,
		"currentCount", currentCount)
	if counted {
		sess.Requests.NotifyLoadStart()
	}
}

func (sess *Session) onPageEventFrameStoppedLoading(ctx context.Context, ev *page.EventFrameStoppedLoading, listenerID int64) {
	if !sess.shouldTrackFrameLifecycle(ctx) {
		return
	}
	previousCount, currentCount, tracked := sess.noteFrameLoadFinished(string(ev.FrameID))
	counted := tracked && currentCount == 0
	message := "Tracked frame stopped loading"
	if !tracked {
		message = "Tracked frame stopped loading without prior start"
	}
	sess.loggerOrDefault().Debug(message,
		"listenerId", listenerID,
		"targetId", targetIDFromContext(ctx),
		"frameId", string(ev.FrameID),
		"tracked", tracked,
		"previousCount", previousCount,
		"counted", counted,
		"currentCount", currentCount)
	if counted {
		sess.Requests.NotifyLoadFinished()
	}
}

func (sess *Session) onPageEventFrameDetached(ctx context.Context, ev *page.EventFrameDetached, listenerID int64) {
	log := sess.loggerOrDefault().With("listenerID", listenerID)

	log.Debug("Frame detached", "frameId", string(ev.FrameID))
	if !sess.shouldTrackFrameLifecycle(ctx) {
		return
	}
	previousCount, tracked := sess.noteFrameLoadDetached(string(ev.FrameID))
	log.Debug("Tracked frame detached",
		"listenerId", listenerID,
		"targetId", targetIDFromContext(ctx),
		"frameId", string(ev.FrameID),
		"reason", string(ev.Reason),
		"tracked", tracked,
		"previousCount", previousCount)
	if tracked {
		sess.Requests.NotifyLoadFinished()
	}
}

func (sess *Session) onPageEventFrameSubtreeWillBeDetached(ctx context.Context, ev *page.EventFrameSubtreeWillBeDetached, listenerID int64) {
	log := sess.loggerOrDefault().With("listenerID", listenerID)

	log.Debug("Frame subtree will be detached", "frameId", string(ev.FrameID))
	if !sess.shouldTrackFrameLifecycle(ctx) {
		return
	}
	previousCount, tracked := sess.noteFrameLoadDetached(string(ev.FrameID))
	log.Debug("Tracked frame subtree will be detached",
		"listenerId", listenerID,
		"targetId", targetIDFromContext(ctx),
		"frameId", string(ev.FrameID),
		"tracked", tracked,
		"previousCount", previousCount)
	if tracked {
		sess.Requests.NotifyLoadFinished()
	}
}

func (sess *Session) onPageEventFileChooserOpened(_ context.Context, ev *page.EventFileChooserOpened, listenerID int64) {
	sess.loggerOrDefault().Warn("File chooser opened",
		"listenerId", listenerID,
		"backendNodeId", ev.BackendNodeID,
		"frameId", ev.FrameID,
		"mode", ev.Mode)
}

func (sess *Session) onPageEventJavascriptDialogOpening(ctx context.Context, ev *page.EventJavascriptDialogOpening, listenerID int64) {
	log := sess.loggerOrDefault().With("listenerID", listenerID)

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
		err := sess.Notify(ev.TargetInfo.TargetID.String())
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
	err := sess.Notify(ev.TargetInfo.TargetID.String())
	if err != nil {
		log.Error("Failed to notify session of new target", "error", err)
	}
}

func (sess *Session) onFetchEventRequestPaused(ctx context.Context, ev *fetch.EventRequestPaused, listenerID int64) {
	log := sess.loggerOrDefault().With("listenerID", listenerID)

	var req *requests.Request
	added := false

	if sess.networkTracker != nil && ev.ResponseStatusCode == 0 && ev.ResponseErrorReason == "" {
		sess.networkTracker.noteObservedRequestStart(ev.ResourceType, ev.Request.URL, ev.FrameID != "", ev.NetworkID)
	}

	if !sess.acceptingRequests() {
		if err := sess.finalizePausedRequest(ctx, ev); err != nil {
			log.Debug("Failed to quiesce paused request during shutdown", "error", err, "targetId", targetIDFromContext(ctx), "fetchRequestId", ev.RequestID.String())
		}
		return
	}

	continueRequest := fetch.ContinueRequest(ev.RequestID)
	if ev.ResponseStatusCode == 0 && ev.ResponseErrorReason == "" {
		continueRequest = continueRequest.WithURL(ev.Request.URL).WithMethod(ev.Request.Method)
		candidate := &requests.Request{
			Method:       ev.Request.Method,
			Url:          url.Normalize(ev.Request.URL + ev.Request.URLFragment),
			RequestId:    ev.RequestID.String(),
			NetworkId:    ev.NetworkID.String(),
			Referrer:     interfaceToString(ev.Request.Headers["Referer"]),
			ResourceType: ev.ResourceType.String(),
		}
		if !sess.acceptingRequests() {
			if err := sess.finalizePausedRequest(ctx, ev); err != nil {
				log.Debug("Failed to quiesce paused request during shutdown", "error", err, "targetId", targetIDFromContext(ctx), "fetchRequestId", ev.RequestID.String())
			}
			return
		}

		req = sess.findRootRequestReuseCandidate(candidate)
		if req == nil {
			req, added = sess.Requests.GetOrAddRequest(candidate)
		} else {
			added = false
		}
		if added {
			log.Debug("Registered paused request",
				"targetId", targetIDFromContext(ctx),
				"fetchRequestId", candidate.RequestId,
				"networkId", candidate.NetworkId,
				"logicalRequestId", req.RequestId,
				"method", req.Method,
				"resourceType", req.ResourceType,
				"url", req.Url)
		}
		if !added && req != nil && req.RequestId != candidate.RequestId {
			log.Debug("Reusing paused request registration",
				"targetId", targetIDFromContext(ctx),
				"fetchRequestId", candidate.RequestId,
				"reusedRequestId", req.RequestId,
				"networkId", candidate.NetworkId,
				"url", candidate.Url)
		}

		if ev.Request.Headers["veidemann_reqid"] != nil {
			delete(ev.Request.Headers, "veidemann_reqid")
		}
		h := make([]*fetch.HeaderEntry, len(ev.Request.Headers)+1)
		i := 0
		for k, v := range ev.Request.Headers {
			h[i] = &fetch.HeaderEntry{Name: k, Value: interfaceToString(v)}
			i++
		}
		h[i] = &fetch.HeaderEntry{Name: "veidemann_reqid", Value: req.RequestId}
		continueRequest = continueRequest.WithHeaders(h)
	} else {
		log.Debug("Response request", "statusCode", ev.ResponseStatusCode, "errorReason", ev.ResponseErrorReason, "url", ev.Request.URL)
	}
	if err := chromedp.Run(ctx, continueRequest); err != nil {
		rolledBack := sess.rollbackPausedRequest(req, added)
		if sess.networkTracker != nil && ev.ResponseStatusCode == 0 && ev.ResponseErrorReason == "" {
			sess.networkTracker.noteRequestDone(ev.NetworkID)
		}
		if rolledBack && isInvalidInterceptionIDError(err) {
			log.Debug("Ignored continue failure for rolled-back paused request", "error", err)
		} else {
			log.Debug("Failed sending continue", "error", err)
		}
	} else {
		if req != nil {
			err = sess.NotifyRequest(req)
		} else {
			err = sess.Notify(interfaceToString(ev.RequestID))
		}
		if err != nil {
			log.Error("Failed to notify session after request continuation", "error", err)
		}
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

func (sess *Session) findRootRequestReuseCandidate(req *requests.Request) *requests.Request {
	if sess.RequestedUrl == nil {
		return nil
	}
	if req.NetworkId != "" || req.Url != sess.RequestedUrl.Uri || req.Method != http.MethodGet {
		return nil
	}

	initial := sess.Requests.InitialRequest()
	if initial != nil && initial.Url == req.Url && initial.Method == req.Method {
		return initial
	}

	existing := sess.Requests.GetByUrl(req.Url, false)
	if existing == nil {
		return nil
	}
	if existing.Method != req.Method {
		return nil
	}
	return existing
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
	if initial := sess.Requests.InitialRequest(); initial == req {
		return false
	}
	if !sess.Requests.RemoveRequest(req) {
		return false
	}
	if req.BlocksPageCompletion() && sess.timer != nil {
		_ = sess.Notify(req.RequestId)
	}
	return true
}

func interfaceToString(i any) string {
	if i == nil {
		return ""
	}
	return fmt.Sprintf("%v", i)
}
