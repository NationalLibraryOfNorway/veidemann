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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/database"
	bcerrors "github.com/NationalLibraryOfNorway/veidemann/browser-controller/errors"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/frontier"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/logwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/requests"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/screenshotwriter"
	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/security"
	"github.com/chromedp/cdproto/serviceworker"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	tracelog "github.com/opentracing/opentracing-go/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var defaultChromiumLaunchArgs = []string{
	"--disable-background-networking",
	"--disable-background-mode",
	"--disable-client-side-phishing-detection",
	"--disable-component-update",
	"--disable-component-extensions-with-background-pages",
	"--disable-default-apps",
	"--disable-domain-reliability",
	"--disable-extensions",
	"--disable-features=AutofillServerCommunication,OptimizationHints,MediaRouter,Translate,InterestFeedContentSuggestions",
	"--disable-gaia-services",
	"--disable-sync",
	"--ignore-certificate-errors",
	"--metrics-recording-only",
	"--no-default-browser-check",
	"--no-first-run",
	"--no-service-autorun",
	"--password-store=basic",
	"--safebrowsing-disable-auto-update",
	"--use-mock-keychain",
}

type defaultViewPort struct {
	DeviceScaleFactor float64 `json:"deviceScaleFactor"`
	HasTouch          bool    `json:"hasTouch"`
	IsMobile          bool    `json:"isMobile"`
	IsLandscape       bool    `json:"isLandscape"`
	Height            int     `json:"height"`
	Width             int     `json:"width"`
}

type launch struct {
	Headless            bool            `json:"headless"`
	Args                []string        `json:"args"`
	AcceptInsecureCerts bool            `json:"acceptInsecureCerts,omitempty"`
	DefaultViewport     defaultViewPort `json:"defaultViewport,omitzero"`
}

type Session struct {
	ctx               context.Context
	Requests          *requests.Registry
	configAdapter     database.ConfigAdapter
	screenShotWriter  screenshotwriter.ScreenshotWriter
	logWriter         logwriter.LogWriter
	logger            *slog.Logger
	browserHost       string
	proxyHost         string
	browserWsEndpoint string
	UserAgent         string
	browserVersion    string
	rootTargetID      target.ID

	RequestedUrl       *frontierV1.QueuedUri
	CrawlConfig        *configV1.CrawlConfig
	browserConfig      *configV1.BrowserConfig
	PolitenessConfig   *configV1.ConfigObject
	frameLoads         *frameLoadTracker
	loadCancel         func()
	networkTracker     *networkActivityTracker
	completionActivity chan struct{}
	scripts            *sessionScripts

	initializedTargets   map[target.ID]struct{}
	initializedTargetsMu sync.Mutex

	listenerSeq atomic.Int64

	Id             int
	browserPort    int
	browserTimeout int
	proxyPort      int

	acceptRequests atomic.Int32
}

var errInitialRequestCached = errors.New("initial request served from cache")

func newDefaultSession(opts ...Option) *Session {
	s := &Session{
		browserHost:        "localhost",
		browserPort:        3000,
		browserTimeout:     500 * 1000,
		proxyPort:          3000,
		networkTracker:     newNetworkActivityTracker(),
		initializedTargets: make(map[target.ID]struct{}),
	}
	for _, opt := range opts {
		opt.apply(s)
	}
	return s
}

func newSession(sessionId int, opts ...Option) *Session {
	sess := newDefaultSession(opts...)
	sess.Id = sessionId
	sess.logger = sess.loggerOrDefault().With("session", sessionId)

	return sess
}

func (sess *Session) loggerOrDefault() *slog.Logger {
	if sess == nil {
		return slog.Default()
	}
	if sess.logger == nil {
		sess.logger = slog.Default()
	}
	return sess.logger
}

func (sess *Session) SignalCompletionActivity() error {
	if sess.ctx != nil {
		select {
		case <-sess.ctx.Done():
			return sess.ctx.Err()
		default:
		}
	}

	signalCompletionActivity(sess.completionActivity)
	return nil
}

func (sess *Session) SignalRequestActivity(req *requests.Request) error {
	if req == nil || !req.BlocksPageCompletion() {
		return nil
	}
	return sess.SignalCompletionActivity()
}

func (sess *Session) startAcceptingRequests() {
	sess.acceptRequests.Store(1)
}

func (sess *Session) stopAcceptingRequests() {
	sess.acceptRequests.Store(0)
}

func (sess *Session) acceptingRequests() bool {
	return sess.acceptRequests.Load() == 1
}

func (sess *Session) stopLoading() error {
	if sess.ctx == nil {
		return nil
	}
	return chromedp.Run(sess.ctx, page.StopLoading())
}

func (sess *Session) Context() context.Context {
	return sess.ctx
}

func (sess *Session) compileBrowserWebsocketEndpoint() (string, error) {
	// Browserless accepts launch configuration on its WebSocket endpoint. A
	// direct CDP backend must resolve its debugger endpoint separately instead
	// of reusing these provider-specific query parameters.
	browserWsUrl := fmt.Sprintf("ws://%s:%d", sess.browserHost, sess.browserPort)
	browserWsEndpoint, err := url.Parse(browserWsUrl)
	if err != nil {
		return "", err
	}

	args := append([]string{}, defaultChromiumLaunchArgs...)

	if sess.proxyHost != "" {
		proxy := "http://" + sess.proxyHost + ":" + strconv.Itoa(sess.proxyPort+sess.Id)
		args = append(args,
			"--proxy-server="+proxy,
			"--proxy-bypass-list=<-loopback>",
		)
	}

	launch := launch{
		Headless:            true,
		Args:                args,
		AcceptInsecureCerts: true,
	}

	b, err := json.Marshal(launch)
	if err != nil {
		return "", err
	}

	q := browserWsEndpoint.Query()

	q.Set("timeout", strconv.Itoa(sess.browserTimeout))
	q.Set("trackingId", strconv.Itoa(sess.Id))
	q.Set("launch", base64.StdEncoding.EncodeToString(b))

	browserWsEndpoint.RawQuery = q.Encode()

	return browserWsEndpoint.String(), nil
}

func (sess *Session) loadFetchConfig(ctx context.Context, phs *frontierV1.PageHarvestSpec) (time.Duration, time.Duration, error) {
	sess.RequestedUrl = phs.GetQueuedUri()
	sess.CrawlConfig = phs.GetCrawlConfig().GetCrawlConfig()

	bConf, err := sess.configAdapter.GetConfigObject(ctx, sess.CrawlConfig.BrowserConfigRef)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get browser config: %v", err)
	}
	sess.browserConfig = bConf.GetBrowserConfig()

	sess.PolitenessConfig, err = sess.configAdapter.GetConfigObject(ctx, sess.CrawlConfig.PolitenessRef)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get politeness config: %v", err)
	}

	if scripts, err := sess.loadScripts(ctx); err != nil {
		return 0, 0, fmt.Errorf("failed to load scripts: %w", err)
	} else {
		sess.scripts = scripts
	}

	browserWsEndpoint, err := sess.compileBrowserWebsocketEndpoint()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to compile browser websocket endpoint: %v", err)
	}
	sess.browserWsEndpoint = browserWsEndpoint

	maxTotalTime := durationFromMilliseconds(sess.browserConfig.PageLoadTimeoutMs)
	maxIdleTime := durationFromMilliseconds(sess.browserConfig.MaxInactivityTimeMs)

	return maxTotalTime, maxIdleTime, nil
}

func (sess *Session) startBrowserSession(ctx context.Context, maxTotalTime time.Duration) (context.Context, context.Context, func(), error) {
	log := sess.loggerOrDefault()

	allocatorContext, allocatorCancel := chromedp.NewRemoteAllocator(ctx, sess.browserWsEndpoint, chromedp.NoModifyURL)
	cdpCtx, cdpCancel := chromedp.NewContext(allocatorContext,
		chromedp.WithErrorf(chromedpErrorf(log)),
	)
	sess.ctx = cdpCtx

	var browserUserAgent string
	var browserVersion string
	if err := chromedp.Run(sess.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			_, browserVersion, _, browserUserAgent, _, err = browser.GetVersion().Do(ctx)
			return err
		}),
	); err != nil {
		cdpCancel()
		allocatorCancel()
		return nil, nil, nil, fmt.Errorf("failed to start browser: %w", err)
	}
	sess.browserVersion = browserVersion

	sess.UserAgent = strings.ReplaceAll(browserUserAgent, "HeadlessChrome", "Chrome")
	if sess.browserConfig.UserAgent != "" {
		sess.UserAgent += " " + sess.browserConfig.UserAgent
	}

	log.Debug("Browser session", "version", browserVersion, "userAgent", sess.UserAgent, "endpoint", sess.browserWsEndpoint)

	loadCtx, loadCancel := context.WithTimeout(sess.ctx, maxTotalTime)
	sess.loadCancel = loadCancel

	sess.frameLoads = newFrameLoadTracker()
	sess.Requests = requests.NewRegistry(log)
	sess.completionActivity = make(chan struct{}, 1)

	sess.initListeners(cdpCtx)

	browserWidth := int64(sess.browserConfig.WindowWidth)
	browserHeight := int64(sess.browserConfig.WindowHeight)
	isLandscape := browserWidth > browserHeight

	deviceInfo := &device.Info{
		Name:      "Desktop",
		UserAgent: sess.UserAgent,
		Width:     browserWidth,
		Height:    browserHeight,
		Scale:     1,
		Landscape: isLandscape,
		Mobile:    false,
		Touch:     false,
	}

	// run task list
	if err := chromedp.Run(sess.ctx,
		security.SetIgnoreCertificateErrors(true),
		network.SetCacheDisabled(true),
		serviceworker.Enable(),
		chromedp.Emulate(deviceInfo),
		fetch.Enable(),
		network.Enable(),
		chromedp.ActionFunc(prepareRemotePage),
		network.SetCookies(sess.getCookieParams(sess.RequestedUrl)),
		runtime.Enable(),
		target.SetAutoAttach(true, false).WithFlatten(true),
	); err != nil {
		loadCancel()
		cdpCancel()
		allocatorCancel()
		return nil, nil, nil, err
	}

	cleanup := func() {
		loadCancel()
		cdpCancel()
		allocatorCancel()
	}

	return cdpCtx, loadCtx, cleanup, nil
}

// prepareRemotePage establishes the page lifecycle state expected by fetch
// scripts. Remote browser backends may leave a newly created target in the
// background, which suppresses visibility-dependent browser behavior.
func prepareRemotePage(ctx context.Context) error {
	if err := page.Enable().Do(ctx); err != nil {
		return fmt.Errorf("enable page domain: %w", err)
	}
	if err := page.BringToFront().Do(ctx); err != nil {
		return fmt.Errorf("bring remote page to front: %w", err)
	}
	return nil
}

func (sess *Session) navigate(loadCtx context.Context) error {
	err := chromedp.Run(loadCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, _, _, _, err := page.Navigate(sess.RequestedUrl.Uri).WithTransitionType(page.TransitionTypeOther).Do(ctx)
			return err
		}),
	)
	if err != nil {
		return sess.classifyNavigationError(err)
	}

	return nil
}

func (sess *Session) classifyNavigationError(err error) error {
	initialRequest := sess.Requests.InitialRequest()

	switch {
	case errors.Is(err, context.Canceled) && initialRequest != nil && initialRequest.FromCache:
		return cacheHitFetchError(sess.RequestedUrl.Uri)
	case errors.Is(err, context.DeadlineExceeded):
		return pageloadTimeoutError(sess.RequestedUrl.Uri, "navigation")
	default:
		return fmt.Errorf("failed to navigate: %w", err)
	}
}

func (sess *Session) waitForInitialPageLoad(loadCtx context.Context) (error, bool) {
	log := sess.loggerOrDefault()

	waitErr := sess.frameLoads.Wait(loadCtx)
	if waitErr == nil {
		return nil, false
	}
	fetchErr, returnNow := classifyFrameWaitError(sess.Requests.InitialRequest(), sess.RequestedUrl.Uri, waitErr)
	if fetchErr == nil {
		return nil, false
	}

	if returnNow {
		return fetchErr, true
	}

	if loadingFrames := sess.frameLoads.Snapshot(); len(loadingFrames) > 0 {
		log.Warn("Frames still marked as loading at timeout", "loadingFrames", loadingFrames)
	} else {
		log.Warn("No frames remained marked as loading at timeout")
	}
	log.Warn("Timed out while waiting for frames to finish loading", "error", fetchErr)
	return fetchErr, false

}

func (sess *Session) runOnLoadBehavior(loadCtx context.Context) {
	if loadCtx.Err() != nil {
		return
	}

	if err := sess.executeScripts(loadCtx, configV1.BrowserScript_ON_LOAD); err != nil {
		sess.loggerOrDefault().Warn("Failed to execute scripts", "phase", configV1.BrowserScript_ON_LOAD.String(), "error", err)
	}
}

func (sess *Session) Fetch(ctx context.Context, phs *frontierV1.PageHarvestSpec) (result *frontier.RenderResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = recoverFetchError(r)
		}
	}()

	span, ctx := opentracing.StartSpanFromContext(ctx, "fetch")
	defer span.Finish()

	span.SetTag("eid", phs.GetQueuedUri().GetExecutionId()).
		SetTag("jeid", phs.GetQueuedUri().GetJobExecutionId()).
		LogFields(
			tracelog.String("uri", phs.GetQueuedUri().GetUri()),
			tracelog.String("seed", phs.GetQueuedUri().GetSeedUri()),
		)
	sess.logger = sess.loggerOrDefault().With(
		"uri", phs.GetQueuedUri().GetUri(),
		"eid", phs.GetQueuedUri().GetExecutionId(),
	)

	log := sess.logger

	maxTotalTime, maxIdleTime, err := sess.loadFetchConfig(ctx, phs)
	if err != nil {
		return nil, err
	}

	log.Info("Start fetch", "maxIdleTime", fmt.Sprintf("%.2fs", maxIdleTime.Seconds()), "maxTotalTime", fmt.Sprintf("%.2fs", maxTotalTime.Seconds()))
	fetchStart := time.Now()

	cdpCtx, loadCtx, cleanup, err := sess.startBrowserSession(ctx, maxTotalTime)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	sess.startAcceptingRequests()
	defer sess.stopAcceptingRequests()

	if err := sess.registerNewDocumentScripts(loadCtx); err != nil {
		return nil, err
	}

	if err := sess.navigate(loadCtx); err != nil {
		return nil, err
	}

	var fetchErr error

	if err, returnNow := sess.waitForInitialPageLoad(loadCtx); returnNow {
		return nil, err
	} else if err != nil {
		fetchErr = err
	}

	sess.runOnLoadBehavior(loadCtx)

	// Wait for the browser to go idle and for tracked requests to receive crawl logs.
	// A frame timeout consumes the load context deadline, so there is no useful
	// completion work left to do in that case.
	var completionErr error
	if loadCtx.Err() == nil {
		completionErr = sess.waitForSettledNetworkAndRequests(loadCtx, maxIdleTime)
	}
	if completionErr != nil {
		log.Warn("Timed out while waiting for outstanding requests", "error", completionErr)
		fetchErr = classifyCompletionWaitError(sess.RequestedUrl.Uri, completionErr)
	}

	if fetchErr != nil {
		sess.stopAcceptingRequests()
		// Only force-stop the page on error or timeout. On the success path, keep
		// the page alive through screenshot capture so late image loads can finish.
		if stopErr := sess.stopLoading(); stopErr != nil {
			log.Warn("Failed to stop loading before finalizing fetch", "error", stopErr)
		}
	}

	rootSnapshot := sess.Requests.RootRequestSnapshot()

	if sess.CrawlConfig.GetExtra().CreateScreenshot {
		screenshotCtx, cancelScreenshot := context.WithTimeout(
			screenshotParentContext(cdpCtx, loadCtx, fetchErr),
			maxIdleTime,
		)
		sess.saveScreenshot(screenshotCtx, rootSnapshot)
		cancelScreenshot()
	}

	cookies := sess.extractCookies()
	outlinks := sess.extractOutlinks(rootSnapshot)

	sess.stopAcceptingRequests()
	if stopErr := sess.stopLoading(); stopErr != nil {
		log.Debug("Failed to stop loading during fetch teardown", "error", stopErr)
	}

	err = chromedp.Cancel(cdpCtx)
	if err != nil {
		log.Warn("Cancel CDP context", "error", err)
	}

	responses := sess.Requests.FinalizeResponses(sess.RequestedUrl)
	initialRequest := responses.InitialRequest
	if initialRequest == nil {
		return nil, errors.New("missing initial request")
	}
	if initialRequest.CrawlLog == nil {
		return nil, errors.New("initial request has no crawllog")
	}
	rootRequest := responses.RootRequest
	if rootRequest == nil || rootRequest.CrawlLog == nil {
		return nil, errors.New("root request has no crawllog")
	}
	fetchDuration := time.Since(fetchStart)

	var crawlLogCount int32
	var bytesDownloaded int64
	var resources []*logV1.PageLog_Resource
	var crawlLogs []*logV1.CrawlLog

	for _, r := range responses.Requests {
		if r.CrawlLog == nil {
			continue
		}

		if r.CrawlLog.GetWarcId() != "" {
			crawlLogs = append(crawlLogs, r.CrawlLog)
			crawlLogCount++
			bytesDownloaded += r.CrawlLog.Size
		}

		resource := &logV1.PageLog_Resource{
			Uri:           r.URL,
			FromCache:     r.FromCache,
			Renderable:    false,
			ResourceType:  r.ResourceType,
			ContentType:   r.CrawlLog.GetContentType(),
			StatusCode:    r.CrawlLog.GetStatusCode(),
			DiscoveryPath: r.CrawlLog.GetDiscoveryPath(),
			WarcId:        r.CrawlLog.GetWarcId(),
			Referrer:      r.Referrer,
			Error:         r.CrawlLog.GetError(),
			Method:        r.Method,
		}
		resources = append(resources, resource)
	}
	if err := sess.logWriter.WriteCrawlLogs(ctx, crawlLogs); err != nil {
		log.Error("Failed to write crawl logs", "error", err)
	}

	pageWarcID := initialRequest.CrawlLog.GetWarcId()
	if pageWarcID == "" {
		pageWarcID = uuid.New().String()
	}
	pageLog := &logV1.PageLog{
		WarcId:              pageWarcID,
		Uri:                 sess.RequestedUrl.Uri,
		ExecutionId:         sess.RequestedUrl.ExecutionId,
		Referrer:            initialRequest.Referrer,
		JobExecutionId:      sess.RequestedUrl.JobExecutionId,
		CollectionFinalName: initialRequest.CrawlLog.CollectionFinalName,
		Method:              initialRequest.Method,
		Resource:            resources,
		Outlink:             outlinks,
	}
	if err := sess.logWriter.WritePageLog(ctx, pageLog); err != nil {
		log.Error("Failed to write pageLog", "error", err)
	}

	qUris := make([]*frontierV1.QueuedUri, len(outlinks))
	for i, uri := range outlinks {
		qUris[i] = &frontierV1.QueuedUri{
			ExecutionId:         sess.RequestedUrl.ExecutionId,
			DiscoveredTimeStamp: timestamppb.Now(),
			Uri:                 uri,
			DiscoveryPath:       rootRequest.CrawlLog.DiscoveryPath + "L",
			Referrer:            rootRequest.URL,
			Cookies:             cookies,
			JobExecutionId:      sess.RequestedUrl.JobExecutionId,
		}
	}
	result = &frontier.RenderResult{
		BytesDownloaded: bytesDownloaded,
		UriCount:        crawlLogCount,
		Outlinks:        qUris,
		Error:           initialRequest.CrawlLog.Error,
		PageFetchTimeMs: fetchDuration.Milliseconds(),
	}

	attrs := []any{
		"bytesDownloaded", bytesDownloaded,
		"outlinks", len(outlinks),
		"crawllogs", len(crawlLogs),
		"resources", len(resources),
		"blockingRequests", responses.BlockingCount,
		"resolvedRequests", responses.ResolvedCount,
		"unresolvedRequests", responses.UnresolvedCount,
		"ignoredRequests", responses.IgnoredCount,
		"duration", fmt.Sprintf("%.2fs", fetchDuration.Seconds()),
	}
	if fetchErr != nil {
		attrs = append(attrs, "error", fetchErr)
	}
	log.Info("Fetch done", attrs...)

	return result, fetchErr
}

func (sess *Session) getCookieParams(uri *frontierV1.QueuedUri) []*network.CookieParam {
	log := sess.loggerOrDefault()

	if cookieCount := len(uri.GetCookies()); cookieCount > 0 {
		log.Debug("Restoring browser cookies", "cookieCount", cookieCount)
	}
	cookies := make([]*network.CookieParam, len(uri.GetCookies()))
	for i, c := range uri.GetCookies() {
		expSec, expNsec := math.Modf(c.Expires)
		expires := cdp.TimeSinceEpoch(time.Unix(int64(expSec), int64(expNsec*(1e9))))

		cookies[i] = &network.CookieParam{
			Name:     c.Name,
			Value:    c.Value,
			URL:      uri.Uri,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HTTPOnly: c.HttpOnly,
			SameSite: network.CookieSameSite(c.SameSite),
			Expires:  &expires,
		}
	}
	return cookies
}

func (sess *Session) extractCookies() []*frontierV1.Cookie {
	log := sess.loggerOrDefault()
	var result []*frontierV1.Cookie

	if err := chromedp.Run(sess.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, err := network.GetCookies().Do(ctx)
			if err != nil {
				return err
			}

			result = make([]*frontierV1.Cookie, len(cookies))
			for i, c := range cookies {
				result[i] = &frontierV1.Cookie{
					Name:     c.Name,
					Value:    c.Value,
					Domain:   c.Domain,
					Path:     c.Path,
					Expires:  c.Expires,
					Size:     int32(c.Size),
					HttpOnly: c.HTTPOnly,
					Secure:   c.Secure,
					Session:  c.Session,
					SameSite: c.SameSite.String(),
				}
			}
			return nil
		}),
	); err != nil {
		log.Error("Could not extract cookies", "error", err)
	}

	return result
}

func (sess *Session) saveScreenshot(ctx context.Context, rootRequest *requests.Request) {
	log := sess.loggerOrDefault()
	if rootRequest == nil {
		log.Debug("Skipping screenshot: missing root request")
		return
	}

	span, ctx := opentracing.StartSpanFromContext(ctx, "screenshot")
	defer span.Finish()
	// Skip screenshot of pages loaded from cache
	if rootRequest.FromCache {
		log.Debug("Skipping screenshot: from cache", "resourceType", rootRequest.ResourceType)
		return
	}
	// Check if page is renderable
	if rootRequest.ResourceType != "Document" && rootRequest.ResourceType != "Image" {
		log.Debug("Skipping screenshot: not renderable", "resourceType", rootRequest.ResourceType)
		return
	}
	// Check if CrawlLog is present for root request
	if rootRequest.CrawlLog == nil {
		log.Debug("Skipping screenshot: missing crawlLog", "resourceType", rootRequest.ResourceType)
		return
	}
	// Check if CrawlLog has WarcId
	if rootRequest.CrawlLog.WarcId == "" {
		log.Debug("Skipping screenshot: crawlLog has empty warcId", "resourceType", rootRequest.ResourceType)
		return
	}
	var data []byte
	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) (err error) {
			capture := page.CaptureScreenshot().WithFormat(page.CaptureScreenshotFormatPng)
			if sess.browserConfig != nil && sess.browserConfig.WindowWidth > 0 && sess.browserConfig.WindowHeight > 0 {
				capture = capture.
					WithClip(&page.Viewport{
						X:      0,
						Y:      0,
						Width:  float64(sess.browserConfig.WindowWidth),
						Height: float64(sess.browserConfig.WindowHeight),
						Scale:  1,
					}).
					WithCaptureBeyondViewport(true)
			}

			data, err = capture.Do(ctx)
			return
		}),
	)
	if err != nil {
		log.Error("Error capturing screenshot", "error", err)
		return
	}
	metadata := screenshotwriter.Metadata{
		CrawlConfig:    sess.CrawlConfig,
		CrawlLog:       rootRequest.CrawlLog,
		BrowserConfig:  sess.browserConfig,
		BrowserVersion: sess.browserVersion,
	}
	if err = sess.screenShotWriter.Write(ctx, data, metadata); err != nil {
		log.Error("Error writing screenshot", "error", err)
		return
	}
}

func (sess *Session) extractOutlinks(rootRequest *requests.Request) []string {
	var extractedUrls []string

	for _, s := range sess.scripts.Get(configV1.BrowserScript_EXTRACT_OUTLINKS) {
		scriptStart := time.Now()
		log := sess.loggerOrDefault().With(
			"scriptType", configV1.BrowserScript_EXTRACT_OUTLINKS.String(),
			"scriptName", s.GetMeta().GetName(),
			"scriptId", s.GetId(),
		)

		script := s.GetBrowserScript().GetScript()
		res, err := evaluateScript(sess.ctx, script)
		if err != nil {
			log.Warn("Script execution failed", "duration", time.Since(scriptStart), "outcome", "failure", "error", err)
			continue
		}
		if res == nil {
			log.Debug("Script execution completed", "duration", time.Since(scriptStart), "outcome", "success", "resultCount", 0)
			continue
		}

		var links []string
		err = json.Unmarshal(res, &links)
		if err != nil {
			log.Warn("Script result decoding failed", "duration", time.Since(scriptStart), "outcome", "failure", "error", err)
			continue
		}
		log.Debug("Script execution completed", "duration", time.Since(scriptStart), "outcome", "success", "resultCount", len(links))

		for _, link := range links {
			link = strings.TrimSpace(link)
			link = strings.Trim(link, "\"\\")
			if link != "" && (rootRequest == nil || link != rootRequest.URL) {
				extractedUrls = append(extractedUrls, link)
			}
		}

	}

	return extractedUrls
}

func (sess *Session) AbortFetch() error {
	if sess.frameLoads != nil {
		sess.frameLoads.Cancel(errInitialRequestCached)
	}
	if sess.loadCancel != nil {
		defer sess.loadCancel()
	}
	if sess.ctx == nil {
		return nil
	}
	return chromedp.Run(sess.ctx, page.StopLoading())
}

func (sess *Session) waitForNetworkIdle(ctx context.Context, maxIdleTime time.Duration) error {
	return sess.networkTracker.waitForIdle(ctx, networkSettleIdleTime(maxIdleTime))
}

func (sess *Session) waitForSettledNetworkAndRequests(ctx context.Context, maxIdleTime time.Duration) error {
	if sess.Requests == nil {
		return errors.New("request registry is not initialized")
	}

	return waitForSettled(
		ctx,
		maxIdleTime,
		sess.completionActivity,
		func(ctx context.Context) error {
			return sess.waitForNetworkIdle(ctx, maxIdleTime)
		},
		sess.Requests.MatchCrawlLogs,
	)
}

func screenshotParentContext(cdpCtx, loadCtx context.Context, fetchErr error) context.Context {
	if fetchErr != nil {
		return cdpCtx
	}
	return loadCtx
}

func recoverFetchError(r any) bcerrors.FetchError {
	fetchErr := recoveredAsFetchError(r)

	// Keep original stack-detail behavior.
	fetchErr.CommonsError().Detail += "\n" + string(debug.Stack())

	return fetchErr
}

func recoveredAsFetchError(r any) bcerrors.FetchError {
	if err, ok := r.(error); ok {
		var fetchErr bcerrors.FetchError
		if errors.As(err, &fetchErr) {
			return fetchErr
		}

		return bcerrors.New(-5, "Runtime error", err.Error())
	}

	return bcerrors.New(-5, "Runtime error", fmt.Sprintf("%v", r))
}

func cacheHitFetchError(uri string) error {
	return bcerrors.New(-4100, "Already seen", "Initial request was found in cache. Url: "+uri)
}

func pageloadTimeoutError(uri, phase string) error {
	detail := "Pageload timed out. Url: " + uri
	if phase != "" {
		detail = fmt.Sprintf("Pageload timed out while waiting for %s. Url: %s", phase, uri)
	}
	return bcerrors.New(-5004, "Runtime exceeded", detail)
}

func classifyFrameWaitError(initialRequest *requests.Request, uri string, waitErr error) (error, bool) {
	switch {
	case errors.Is(waitErr, errInitialRequestCached):
		if initialRequest != nil && initialRequest.FromCache {
			return cacheHitFetchError(uri), true
		}
		return waitErr, true
	case errors.Is(waitErr, context.DeadlineExceeded):
		return pageloadTimeoutError(uri, "frames to finish loading"), false
	case waitErr != nil:
		return waitErr, true
	}

	return nil, false
}

func classifyCompletionWaitError(uri string, waitErr error) error {
	switch {
	case errors.Is(waitErr, errCompletionIdleTimeout), errors.Is(waitErr, context.DeadlineExceeded):
		return pageloadTimeoutError(uri, "outstanding requests to complete")
	default:
		return waitErr
	}
}

func networkSettleIdleTime(maxIdleTime time.Duration) time.Duration {
	if maxIdleTime < time.Second {
		return time.Second
	}
	return maxIdleTime
}

func durationFromMilliseconds(milliseconds int64) time.Duration {
	return time.Duration(milliseconds) * time.Millisecond
}
