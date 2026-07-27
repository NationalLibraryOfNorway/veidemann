//go:build integration

/*
 * Copyright 2026 National Library of Norway.
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
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestPrepareRemotePage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	browserless, host, port := startBrowserlessContainer(t, ctx)
	defer terminateBrowserlessContainer(browserless)

	sess := newDefaultSession(WithBrowserHost(host), WithBrowserPort(port))
	endpoint, err := sess.compileBrowserWebsocketEndpoint()
	if err != nil {
		t.Fatalf("compile browser WebSocket endpoint: %v", err)
	}

	allocatorCtx, cancelAllocator := chromedp.NewRemoteAllocator(ctx, endpoint, chromedp.NoModifyURL)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx)
	defer cancelBrowser()

	var visibilityState string
	var hidden bool
	var focused bool
	if err := chromedp.Run(browserCtx,
		chromedp.ActionFunc(prepareRemotePage),
		chromedp.Navigate("data:text/html,<html><body>remote page activation</body></html>"),
		chromedp.Evaluate("document.visibilityState", &visibilityState),
		chromedp.Evaluate("document.hidden", &hidden),
		chromedp.Evaluate("document.hasFocus()", &focused),
	); err != nil {
		t.Fatalf("prepare and navigate remote page: %v", err)
	}

	if visibilityState != "visible" {
		t.Errorf("document.visibilityState = %q, want visible", visibilityState)
	}
	if hidden {
		t.Error("document.hidden = true, want false")
	}
	if !focused {
		t.Error("document.hasFocus() = false, want true")
	}
}
