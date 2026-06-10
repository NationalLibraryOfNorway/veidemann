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
	chromedp.ListenTarget(ctx, sess.listenFunc(ctx))
}

func (sess *Session) listenFunc(ctx context.Context) func(ev any) {
	log := sess.logger
	return func(ev any) {
		switch ev := ev.(type) {
		case *network.EventRequestWillBeSent:
			log.Debug("Request will be sent",
				"requestID", ev.RequestID,
				"type", ev.Type,
				"frameID", ev.FrameID,
				"initiator", ev.Initiator.Type,
				"loaderID", ev.LoaderID,
				"documentURL", ev.DocumentURL)
			if req := sess.Requests.GetByNetworkId(ev.RequestID.String()); req != nil {
				req.Initiator = ev.Initiator.Type.String()
			}
		case *network.EventLoadingFailed:
			log.Debug("Loading failed",
				"type", ev.Type,
				"errorText", ev.ErrorText,
				"blockedReason", ev.BlockedReason,
				"canceled", ev.Canceled,
				"requestID", ev.RequestID)
		case *page.EventFrameStartedLoading:
			log.Debug("Frame started loading", "frameID", ev.FrameID)
			sess.Requests.NotifyLoadStart()
		case *page.EventFrameStoppedLoading:
			log.Debug("Frame stopped loading", "frameID", ev.FrameID)
			sess.Requests.NotifyLoadFinished()
		case *page.EventFileChooserOpened:
			log.Warn("File chooser opened", "backendNodeID", ev.BackendNodeID, "frameID", ev.FrameID, "mode", ev.Mode)
		case *page.EventJavascriptDialogOpening:
			log.Debug("Javascript dialog opening", "message", ev.Message)
			go func() {
				accept := ev.Type == "alert"
				if err := chromedp.Run(ctx,
					page.HandleJavaScriptDialog(accept),
				); err != nil {
					log.Error("Could not handle JavaScript dialog", "error", err)
				}
			}()
		case *target.EventTargetCreated:
			log.Debug("Target created",
				"targetID", ev.TargetInfo.TargetID,
				"openerID", ev.TargetInfo.OpenerID,
				"browserContextID", ev.TargetInfo.BrowserContextID,
				"type", ev.TargetInfo.Type,
				"title", ev.TargetInfo.Title,
				"url", ev.TargetInfo.URL,
				"attached", ev.TargetInfo.Attached)
			newCtx, _ := chromedp.NewContext(ctx, chromedp.WithTargetID(ev.TargetInfo.TargetID))
			go func() {
				<-ctx.Done()
				_ = chromedp.Cancel(newCtx)
			}()
			if err := chromedp.Run(newCtx); err != nil {
				log.Warn("Failed connecting to new target", "error", err)
			}

			var actions []chromedp.Action

			switch ev.TargetInfo.Type {
			case "service_worker":
				actions = []chromedp.Action{
					fetch.Enable(),
					runtime.Enable(),
					target.SetAutoAttach(true, false).WithFlatten(true),
					runtime.RunIfWaitingForDebugger(),
					network.SetCacheDisabled(true),
					network.SetCookies(sess.getCookieParams(sess.RequestedUrl)),
				}
			case "worker":
				actions = []chromedp.Action{
					runtime.Enable(),
					target.SetAutoAttach(true, false).WithFlatten(true),
					runtime.RunIfWaitingForDebugger(),
					network.SetCacheDisabled(true),
				}
			default:
				actions = []chromedp.Action{
					fetch.Enable(),
					runtime.Enable(),
					target.SetAutoAttach(true, false).WithFlatten(true),
					runtime.RunIfWaitingForDebugger(),
					network.Enable(),
					page.Enable(),
					network.SetCacheDisabled(true),
					security.SetIgnoreCertificateErrors(true),
					network.SetCookies(sess.getCookieParams(sess.RequestedUrl)),
				}
			}

			go func() {
				if err := chromedp.Run(newCtx, actions...); err != nil {
					log.Error("Failed initializing new target", "error", err)
				}

				chromedp.ListenTarget(newCtx, sess.listenFunc(newCtx))
			}()
			err := sess.Notify(ev.TargetInfo.TargetID.String())
			if err != nil {
				log.Error("Failed to notify session of new target", "error", err)
			}
		case *fetch.EventRequestPaused:
			go func() {
				continueRequest := fetch.ContinueRequest(ev.RequestID)
				if ev.ResponseStatusCode == 0 && ev.ResponseErrorReason == "" {
					continueRequest = continueRequest.WithURL(ev.Request.URL).WithMethod(ev.Request.Method)
					req := &requests.Request{
						Method:       ev.Request.Method,
						Url:          url.Normalize(ev.Request.URL + ev.Request.URLFragment),
						RequestId:    ev.RequestID.String(),
						NetworkId:    ev.NetworkID.String(),
						Referrer:     interfaceToString(ev.Request.Headers["Referer"]),
						ResourceType: ev.ResourceType.String(),
					}

					sess.Requests.AddRequest(req)

					if ev.Request.Headers["veidemann_reqid"] != nil {
						delete(ev.Request.Headers, "veidemann_reqid")
					}
					h := make([]*fetch.HeaderEntry, len(ev.Request.Headers)+1)
					i := 0
					for k, v := range ev.Request.Headers {
						h[i] = &fetch.HeaderEntry{Name: k, Value: interfaceToString(v)}
						i++
					}
					h[i] = &fetch.HeaderEntry{Name: "veidemann_reqid", Value: ev.RequestID.String()}
					continueRequest = continueRequest.WithHeaders(h)
				} else {
					log.Debug("RESPONSE REQUEST", "statusCode", ev.ResponseStatusCode, "errorReason", ev.ResponseErrorReason, "url", ev.Request.URL)
				}
				if err := chromedp.Run(ctx, continueRequest); err != nil {
					log.Debug("Failed sending continue", "error", err)
				} else {
					err = sess.Notify(ev.RequestID.String())
					if err != nil {
						log.Error("Failed to notify session after request continuation", "error", err)
					}
				}
			}()
		}
	}
}

func interfaceToString(i any) string {
	if i == nil {
		return ""
	}
	return fmt.Sprintf("%v", i)
}
