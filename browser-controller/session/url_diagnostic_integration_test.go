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
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/requests"
	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/security"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
)

var (
	diagnosticURL      = flag.String("diagnostic-url", "", "URL to inspect with TestURLDiagnostics")
	diagnosticTarget   = flag.String("diagnostic-target", "", "optional regexp selecting image URLs for detailed output")
	diagnosticModes    = flag.String("diagnostic-modes", "isolated-scroll", "comma-separated diagnostic modes: isolated-scroll, main-scroll, foreground-scroll, scroll-into-view, eager")
	diagnosticWidth    = flag.Int("diagnostic-width", 1920, "browser viewport width")
	diagnosticHeight   = flag.Int("diagnostic-height", 1080, "browser viewport height")
	diagnosticIdleWait = flag.Duration("diagnostic-idle-wait", 2*time.Second, "quiet period after the diagnostic action")
	diagnosticTimeout  = flag.Duration("diagnostic-timeout", 180*time.Second, "timeout for each diagnostic mode")
)

const maxDiagnosticDetails = 50

type urlDiagnosticScriptOptions struct {
	Mode       string `json:"mode"`
	Target     string `json:"target"`
	IdleWaitMs int64  `json:"idleWaitMs"`
	MaxDetails int    `json:"maxDetails"`
}

type urlDiagnosticScriptResult struct {
	Scroll struct {
		Container      string `json:"container"`
		StartTop       int64  `json:"startTop"`
		EndTop         int64  `json:"endTop"`
		StartHeight    int64  `json:"startHeight"`
		EndHeight      int64  `json:"endHeight"`
		ViewportHeight int64  `json:"viewportHeight"`
		ScrollCount    int    `json:"scrollCount"`
		Rounds         int    `json:"rounds"`
		StoppedBecause string `json:"stoppedBecause"`
	} `json:"scroll"`
	LazyImageCount            int                  `json:"lazyImageCount"`
	IntersectionCount         int                  `json:"intersectionCount"`
	ObserverIntersectionCount int                  `json:"observerIntersectionCount"`
	ViewportVisitCount        int                  `json:"viewportVisitCount"`
	LoadedImageCount          int                  `json:"loadedImageCount"`
	LoadEventCount            int                  `json:"loadEventCount"`
	ErrorEventCount           int                  `json:"errorEventCount"`
	DocumentVisibility        string               `json:"documentVisibility"`
	DocumentHidden            bool                 `json:"documentHidden"`
	DocumentHasFocus          bool                 `json:"documentHasFocus"`
	AnimationFrameCount       int                  `json:"animationFrameCount"`
	DetailsTruncated          bool                 `json:"detailsTruncated"`
	Images                    []urlDiagnosticImage `json:"images"`
}

type urlDiagnosticImage struct {
	Src            string  `json:"src"`
	CurrentSrc     string  `json:"currentSrc"`
	Complete       bool    `json:"complete"`
	NaturalWidth   int     `json:"naturalWidth"`
	Visible        bool    `json:"visible"`
	Intersected    bool    `json:"intersected"`
	Top            float64 `json:"top"`
	Bottom         float64 `json:"bottom"`
	Width          float64 `json:"width"`
	Height         float64 `json:"height"`
	NetworkStatus  string  `json:"networkStatus"`
	NetworkError   string  `json:"networkError,omitempty"`
	Classification string  `json:"classification"`
}

type urlDiagnosticReport struct {
	URL               string                        `json:"url"`
	Mode              string                        `json:"mode"`
	ViewportWidth     int                           `json:"viewportWidth"`
	ViewportHeight    int                           `json:"viewportHeight"`
	BrowserVersion    string                        `json:"browserVersion"`
	BrowserArguments  []string                      `json:"browserArguments,omitempty"`
	Script            urlDiagnosticScriptResult     `json:"script"`
	ImageRequests     []urlDiagnosticNetworkRequest `json:"imageRequests"`
	RequestsTruncated bool                          `json:"requestsTruncated"`
}

type urlDiagnosticNetworkRequest struct {
	RequestID string `json:"requestId"`
	URL       string `json:"url"`
	Finished  bool   `json:"finished"`
	Failed    bool   `json:"failed"`
	ErrorText string `json:"errorText,omitempty"`
}

type urlDiagnosticNetworkCollector struct {
	mu         sync.Mutex
	requests   map[network.RequestID]*urlDiagnosticNetworkRequest
	lastChange time.Time
	updates    chan struct{}
}

func newURLDiagnosticNetworkCollector() *urlDiagnosticNetworkCollector {
	return &urlDiagnosticNetworkCollector{
		requests:   make(map[network.RequestID]*urlDiagnosticNetworkRequest),
		lastChange: time.Now(),
		updates:    make(chan struct{}, 1),
	}
}

func (c *urlDiagnosticNetworkCollector) handleEvent(event any) {
	c.mu.Lock()
	changed := false
	switch ev := event.(type) {
	case *network.EventRequestWillBeSent:
		if ev.Type == network.ResourceTypeImage {
			c.requests[ev.RequestID] = &urlDiagnosticNetworkRequest{
				RequestID: string(ev.RequestID),
				URL:       ev.Request.URL,
			}
			changed = true
		}
	case *network.EventLoadingFinished:
		if req := c.requests[ev.RequestID]; req != nil {
			req.Finished = true
			changed = true
		}
	case *network.EventLoadingFailed:
		if req := c.requests[ev.RequestID]; req != nil {
			req.Failed = true
			req.ErrorText = ev.ErrorText
			changed = true
		}
	}
	if changed {
		c.lastChange = time.Now()
	}
	c.mu.Unlock()
	if changed {
		select {
		case c.updates <- struct{}{}:
		default:
		}
	}
}

func (c *urlDiagnosticNetworkCollector) waitForIdle(ctx context.Context, idleWait time.Duration) error {
	for {
		c.mu.Lock()
		remaining := idleWait - time.Since(c.lastChange)
		c.mu.Unlock()
		if remaining <= 0 {
			return nil
		}

		timer := time.NewTimer(remaining)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-c.updates:
			timer.Stop()
		case <-timer.C:
			return nil
		}
	}
}

func (c *urlDiagnosticNetworkCollector) snapshot(target *regexp.Regexp) ([]urlDiagnosticNetworkRequest, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]urlDiagnosticNetworkRequest, 0, len(c.requests))
	for _, req := range c.requests {
		if target != nil && !target.MatchString(req.URL) {
			continue
		}
		clone := *req
		clone.URL, _ = requests.BoundedURLForLog(clone.URL)
		result = append(result, clone)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].URL == result[j].URL {
			return result[i].RequestID < result[j].RequestID
		}
		return result[i].URL < result[j].URL
	})
	truncated := len(result) > maxDiagnosticDetails
	if truncated {
		result = result[:maxDiagnosticDetails]
	}
	return result, truncated
}

func TestURLDiagnostics(t *testing.T) {
	if strings.TrimSpace(*diagnosticURL) == "" {
		t.Skip("set -diagnostic-url to run the arbitrary-URL diagnostic")
	}
	parsedURL, err := url.ParseRequestURI(*diagnosticURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		t.Fatalf("invalid -diagnostic-url %q", *diagnosticURL)
	}
	if *diagnosticWidth <= 0 || *diagnosticHeight <= 0 {
		t.Fatal("diagnostic viewport dimensions must be positive")
	}
	if *diagnosticIdleWait <= 0 || *diagnosticTimeout <= 0 {
		t.Fatal("diagnostic durations must be positive")
	}

	targetRegexp, err := compileURLDiagnosticTarget(*diagnosticTarget)
	if err != nil {
		t.Fatalf("invalid -diagnostic-target: %v", err)
	}
	modes, err := parseURLDiagnosticModes(*diagnosticModes)
	if err != nil {
		t.Fatal(err)
	}

	containerTimeout := time.Duration(len(modes))*(*diagnosticTimeout) + time.Minute
	containerCtx, cancelContainer := context.WithTimeout(context.Background(), containerTimeout)
	defer cancelContainer()
	browserless, host, port := startBrowserlessContainer(t, containerCtx)
	defer terminateBrowserlessContainer(browserless)

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			report := runURLDiagnosticMode(t, containerCtx, host, port, mode, targetRegexp)
			encoded, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				t.Fatalf("marshal diagnostic report: %v", err)
			}
			t.Logf("URL diagnostic report:\n%s", encoded)
		})
	}
}

func runURLDiagnosticMode(t *testing.T, parent context.Context, host string, port int, mode string, target *regexp.Regexp) urlDiagnosticReport {
	t.Helper()
	ctx, cancel := context.WithTimeout(parent, *diagnosticTimeout)
	defer cancel()

	sess := newDefaultSession(WithBrowserHost(host), WithBrowserPort(port))
	sess.browserTimeout = int((*diagnosticTimeout).Milliseconds())
	endpoint, err := sess.compileBrowserWebsocketEndpoint()
	if err != nil {
		t.Fatalf("compile browser websocket endpoint: %v", err)
	}
	allocatorCtx, cancelAllocator := chromedp.NewRemoteAllocator(ctx, endpoint, chromedp.NoModifyURL)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx)
	defer cancelBrowser()

	collector := newURLDiagnosticNetworkCollector()
	chromedp.ListenTarget(browserCtx, collector.handleEvent)

	var browserVersion string
	var browserUserAgent string
	var browserArguments []string
	err = chromedp.Run(browserCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var actionErr error
			_, browserVersion, _, browserUserAgent, _, actionErr = browser.GetVersion().Do(ctx)
			return actionErr
		}),
	)
	if err != nil {
		t.Fatalf("get browser version: %v", err)
	}
	_ = chromedp.Run(browserCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		arguments, actionErr := browser.GetBrowserCommandLine().Do(ctx)
		if actionErr == nil {
			browserArguments = filterURLDiagnosticBrowserArguments(arguments)
		}
		return nil
	}))
	err = chromedp.Run(browserCtx,
		security.SetIgnoreCertificateErrors(true),
		network.Enable(),
		page.Enable(),
		runtime.Enable(),
		network.SetCacheDisabled(true),
		chromedp.Emulate(device.Info{
			Name:      "Veidemann URL diagnostic",
			UserAgent: strings.ReplaceAll(browserUserAgent, "HeadlessChrome", "Chrome") + " veidemann/development",
			Width:     int64(*diagnosticWidth),
			Height:    int64(*diagnosticHeight),
			Scale:     1,
			Landscape: *diagnosticWidth > *diagnosticHeight,
		}),
		chromedp.Navigate(*diagnosticURL),
	)
	if err != nil {
		t.Fatalf("navigate to %q: %v", *diagnosticURL, err)
	}
	if mode == "foreground-scroll" {
		if err := chromedp.Run(browserCtx, page.BringToFront()); err != nil {
			t.Fatalf("bring diagnostic page to front: %v", err)
		}
	}

	options := urlDiagnosticScriptOptions{
		Mode:       mode,
		Target:     *diagnosticTarget,
		IdleWaitMs: diagnosticIdleWait.Milliseconds(),
		MaxDetails: maxDiagnosticDetails,
	}
	arguments, err := json.Marshal(options)
	if err != nil {
		t.Fatalf("marshal diagnostic options: %v", err)
	}
	result, err := executeURLDiagnosticScript(browserCtx, mode, arguments)
	if err != nil {
		t.Fatalf("execute %s diagnostic: %v", mode, err)
	}
	if err := collector.waitForIdle(browserCtx, *diagnosticIdleWait); err != nil {
		t.Fatalf("wait for image network idle: %v", err)
	}

	var scriptResult urlDiagnosticScriptResult
	if err := json.Unmarshal(result, &scriptResult); err != nil {
		t.Fatalf("decode diagnostic result: %v", err)
	}
	classifyURLDiagnosticImages(scriptResult.Images, collector)
	imageRequests, requestsTruncated := collector.snapshot(target)

	return urlDiagnosticReport{
		URL:               *diagnosticURL,
		Mode:              mode,
		ViewportWidth:     *diagnosticWidth,
		ViewportHeight:    *diagnosticHeight,
		BrowserVersion:    browserVersion,
		BrowserArguments:  browserArguments,
		Script:            scriptResult,
		ImageRequests:     imageRequests,
		RequestsTruncated: requestsTruncated,
	}
}

func filterURLDiagnosticBrowserArguments(arguments []string) []string {
	keywords := []string{"headless", "feature", "background", "visibility", "render", "image", "blink"}
	var filtered []string
	for _, argument := range arguments {
		lower := strings.ToLower(argument)
		for _, keyword := range keywords {
			if strings.Contains(lower, keyword) {
				filtered = append(filtered, argument)
				break
			}
		}
	}
	return filtered
}

func executeURLDiagnosticScript(ctx context.Context, mode string, arguments json.RawMessage) (json.RawMessage, error) {
	if mode == "main-scroll" {
		expression := fmt.Sprintf("(%s)(%s)", urlDiagnosticScript, arguments)
		return evaluateScript(ctx, expression)
	}
	executionContextID, err := getExecutionContextID(ctx)
	if err != nil {
		return nil, err
	}
	return callScript(ctx, executionContextID, urlDiagnosticScript, arguments)
}

func compileURLDiagnosticTarget(pattern string) (*regexp.Regexp, error) {
	if strings.TrimSpace(pattern) == "" {
		return nil, nil
	}
	return regexp.Compile(pattern)
}

func parseURLDiagnosticModes(raw string) ([]string, error) {
	allowed := map[string]bool{
		"isolated-scroll":   true,
		"main-scroll":       true,
		"foreground-scroll": true,
		"scroll-into-view":  true,
		"eager":             true,
	}
	seen := make(map[string]bool)
	var modes []string
	for _, part := range strings.Split(raw, ",") {
		mode := strings.TrimSpace(part)
		if mode == "" {
			continue
		}
		if !allowed[mode] {
			return nil, fmt.Errorf("unsupported diagnostic mode %q", mode)
		}
		if !seen[mode] {
			modes = append(modes, mode)
			seen[mode] = true
		}
	}
	if len(modes) == 0 {
		return nil, fmt.Errorf("at least one diagnostic mode is required")
	}
	return modes, nil
}

func classifyURLDiagnosticImages(images []urlDiagnosticImage, collector *urlDiagnosticNetworkCollector) {
	collector.mu.Lock()
	defer collector.mu.Unlock()

	for idx := range images {
		image := &images[idx]
		selectedURL := image.CurrentSrc
		if selectedURL == "" {
			selectedURL = image.Src
		}
		var matched *urlDiagnosticNetworkRequest
		for _, req := range collector.requests {
			if req.URL == selectedURL {
				matched = req
				if req.Finished && !req.Failed {
					break
				}
			}
		}

		switch {
		case !image.Visible || selectedURL == "":
			image.Classification = "not-rendered"
		case matched != nil && matched.Finished && !matched.Failed && image.Complete && image.NaturalWidth > 0:
			image.NetworkStatus = "completed"
			image.Classification = "completed"
		case matched != nil:
			image.NetworkStatus = "failed"
			image.NetworkError = matched.ErrorText
			if image.NetworkError == "" && !matched.Finished {
				image.NetworkError = "request did not finish before the idle timeout"
			}
			image.Classification = "requested-but-failed"
		case !image.Intersected:
			image.NetworkStatus = "not-requested"
			image.Classification = "never-intersected"
		default:
			image.NetworkStatus = "not-requested"
			image.Classification = "intersected-but-not-requested"
		}

		image.Src, _ = requests.BoundedURLForLog(image.Src)
		image.CurrentSrc, _ = requests.BoundedURLForLog(image.CurrentSrc)
	}
}

const urlDiagnosticScript = `async function diagnoseLazyImages(options) {
  const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
  let animationFrameCount = 0;
  const nextFrame = () => new Promise(resolve => {
    let settled = false;
    const finish = () => {
      if (settled) return;
      settled = true;
      resolve();
    };
    setTimeout(finish, 100);
    requestAnimationFrame(() => {
      animationFrameCount += 1;
      requestAnimationFrame(() => {
        animationFrameCount += 1;
        finish();
      });
    });
  });
  const bounded = value => {
    const text = String(value || '');
    return text.length <= 512 ? text : text.slice(0, 511) + '…';
  };
  const target = options.target ? new RegExp(options.target) : null;
  const maxDetails = Math.max(1, Number(options.maxDetails) || 50);
  const container = document.scrollingElement || document.documentElement || document.body;
  const lazyImages = Array.from(document.querySelectorAll('img[loading="lazy"]'));
  const observerIntersected = new Set();
  const viewportVisited = new Set();
  let loadEventCount = 0;
  let errorEventCount = 0;

  for (const img of lazyImages) {
    img.addEventListener('load', () => { loadEventCount += 1; }, {once: true});
    img.addEventListener('error', () => { errorEventCount += 1; }, {once: true});
  }
  const observer = new IntersectionObserver(entries => {
    for (const entry of entries) {
      if (entry.isIntersecting) observerIntersected.add(entry.target);
    }
  });
  lazyImages.forEach(img => observer.observe(img));
  const recordViewportVisits = () => {
    for (const img of lazyImages) {
      const rect = img.getBoundingClientRect();
      if (rect.width > 0 && rect.height > 0 && rect.bottom > 0 &&
          rect.top < window.innerHeight && rect.right > 0 && rect.left < window.innerWidth) {
        viewportVisited.add(img);
      }
    }
  };

  const startTop = container.scrollTop;
  const startHeight = container.scrollHeight;
  const viewportHeight = container.clientHeight;
  let scrollCount = 0;
  let rounds = 0;
  let stoppedBecause = options.mode;

  await nextFrame();
  recordViewportVisits();

  if (options.mode === 'eager') {
    lazyImages.forEach(img => { img.loading = 'eager'; });
    stoppedBecause = 'loading-set-to-eager';
    await nextFrame();
  } else if (options.mode === 'scroll-into-view') {
    for (const img of lazyImages) {
      img.scrollIntoView({behavior: 'instant', block: 'center', inline: 'nearest'});
      scrollCount += 1;
      rounds += 1;
      await nextFrame();
      recordViewportVisits();
      await sleep(250);
    }
    stoppedBecause = 'all-lazy-images-visited';
  } else {
    const maxIdleRounds = 5;
    const maxScrolls = 1000;
    const maxRounds = 1200;
    const bottomThreshold = 5;
    const step = Math.max(1, Math.floor(container.clientHeight * 0.8));
    let lastScrollHeight = container.scrollHeight;
    let idleRounds = 0;

    while (scrollCount < maxScrolls && rounds < maxRounds) {
      rounds += 1;
      const currentTop = container.scrollTop;
      const scrollHeight = container.scrollHeight;
      const maxTop = Math.max(0, scrollHeight - container.clientHeight);
      const nearBottom = currentTop + container.clientHeight >= scrollHeight - bottomThreshold;

      if (nearBottom) {
        await sleep(1000);
        const newScrollHeight = container.scrollHeight;
        if (newScrollHeight === lastScrollHeight) {
          idleRounds += 1;
        } else {
          idleRounds = 0;
          lastScrollHeight = newScrollHeight;
        }
        if (idleRounds >= maxIdleRounds) {
          stoppedBecause = 'bottom-settled';
          break;
        }
        continue;
      }

      lastScrollHeight = scrollHeight;
      const nextTop = Math.min(currentTop + step, maxTop);
      if (nextTop === currentTop) {
        idleRounds += 1;
      } else {
        container.scrollTop = nextTop;
        scrollCount += 1;
        idleRounds = 0;
      }
      await sleep(250);
      recordViewportVisits();
    }
    if (scrollCount >= maxScrolls) stoppedBecause = 'max-scrolls';
    if (rounds >= maxRounds) stoppedBecause = 'max-rounds';
  }

  await sleep(Math.max(1, Number(options.idleWaitMs) || 2000));
  await nextFrame();
  recordViewportVisits();
  observer.disconnect();

  const matching = lazyImages.filter(img => {
    if (!target) return true;
    return [img.src, img.currentSrc, img.getAttribute('srcset')]
      .filter(Boolean)
      .some(value => target.test(value));
  });
  const images = matching.slice(0, maxDetails).map(img => {
    const rect = img.getBoundingClientRect();
    const style = getComputedStyle(img);
    const visible = rect.width > 0 && rect.height > 0 &&
      style.display !== 'none' && style.visibility !== 'hidden';
    return {
      src: bounded(img.src),
      currentSrc: bounded(img.currentSrc),
      complete: img.complete,
      naturalWidth: img.naturalWidth,
      visible,
      intersected: observerIntersected.has(img) || viewportVisited.has(img),
      top: rect.top,
      bottom: rect.bottom,
      width: rect.width,
      height: rect.height
    };
  });

  return {
    scroll: {
      container: bounded(container.tagName + (container.id ? '#' + container.id : '')),
      startTop,
      endTop: container.scrollTop,
      startHeight,
      endHeight: container.scrollHeight,
      viewportHeight,
      scrollCount,
      rounds,
      stoppedBecause
    },
    lazyImageCount: lazyImages.length,
    intersectionCount: new Set([...observerIntersected, ...viewportVisited]).size,
    observerIntersectionCount: observerIntersected.size,
    viewportVisitCount: viewportVisited.size,
    loadedImageCount: lazyImages.filter(img => img.complete && img.naturalWidth > 0).length,
    loadEventCount,
    errorEventCount,
    documentVisibility: document.visibilityState,
    documentHidden: document.hidden,
    documentHasFocus: document.hasFocus(),
    animationFrameCount,
    detailsTruncated: matching.length > maxDetails,
    images
  };
}`
