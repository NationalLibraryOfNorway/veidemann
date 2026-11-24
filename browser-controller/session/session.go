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
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/database"
	bcerrors "github.com/NationalLibraryOfNorway/veidemann/browser-controller/errors"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/screenshotwriter"
	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/serviceworker"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
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
	"--disable-features=AutofillServerCommunication,OptimizationHints,MediaRouter,Translate,InterestFeedContentSuggestions,NetworkTimeServiceQuerying,AimEnabled",
	"--disable-gaia-services",
	"--disable-sync",
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
	Headless        bool            `json:"headless"`
	Args            []string        `json:"args"`
	DefaultViewport defaultViewPort `json:"defaultViewport,omitzero"`
}

type Session struct {
	poisoned                bool
	recorder                recorderV1.RecorderControlClient
	admissionToken          string
	attemptID               string
	mainFrameID             cdp.FrameID
	captureTimeout          time.Duration
	persistenceTimeout      time.Duration
	ctx                     context.Context
	browserCancel           context.CancelCauseFunc
	configAdapter           database.ConfigAdapter
	screenShotWriter        screenshotwriter.ScreenshotWriter
	logger                  *slog.Logger
	diagnostics             *harvestDiagnostics
	browserHost             string
	proxyHost               string
	recorderCertificateSPKI string
	browserWsEndpoint       string
	UserAgent               string
	browserVersion          string

	RequestedUrl     *frontierV1.QueuedUri
	CrawlConfig      *configV1.CrawlConfig
	browserConfig    *configV1.BrowserConfig
	PolitenessConfig *configV1.ConfigObject
	pageLoad         *pageLoadSignal
	networkTracker   *networkActivityTracker
	scripts          *sessionScripts

	childrenMu     sync.Mutex
	children       map[target.SessionID]*childSession
	childSetups    int
	childrenSealed bool
	childErr       error
	childUpdates   chan struct{}

	listenerSeq atomic.Int64

	Id             int
	browserPort    int
	browserTimeout int
	proxyPort      int

	mediaSettleTimeout     time.Duration
	completionDrainTimeout time.Duration
	readinessReason        string
}

func newDefaultSession(opts ...Option) *Session {
	s := &Session{
		browserHost:            "localhost",
		captureTimeout:         30 * time.Second,
		persistenceTimeout:     30 * time.Second,
		browserPort:            3000,
		browserTimeout:         500 * 1000,
		proxyPort:              3000,
		networkTracker:         newNetworkActivityTracker(),
		logger:                 slog.Default(),
		diagnostics:            newHarvestDiagnostics(),
		pageLoad:               newPageLoadSignal(),
		children:               make(map[target.SessionID]*childSession),
		childUpdates:           make(chan struct{}, 1),
		mediaSettleTimeout:     5 * time.Second,
		completionDrainTimeout: 45 * time.Second,
	}
	for _, opt := range opts {
		opt.apply(s)
	}
	return s
}

func New(sessionId int, opts ...Option) *Session {
	sess := newDefaultSession(opts...)
	sess.Id = sessionId
	sess.logger = sess.logger.With("component", "browser-controller", "slot", sessionId)

	return sess
}

// Logger carries the current harvest's correlation fields for orchestration logs.
func (sess *Session) Logger() *slog.Logger { return sess.logger }

// Browserless identifies live sessions by trackingId. Reusing a slot number can
// collide while the previous provider session is still closing. The attempt is
// both unique and directly searchable across all three services' logs.
func (sess *Session) browserTrackingID() string {
	if sess.attemptID != "" {
		// Browserless v2.57.3 requires fewer than 32 characters. A URL-safe,
		// 128-bit attempt fingerprint fits while preserving per-attempt identity.
		fingerprint := sha256.Sum256([]byte(sess.attemptID))
		return base64.RawURLEncoding.EncodeToString(fingerprint[:16])
	}
	return strconv.Itoa(sess.Id) // Standalone diagnostics have no harvest attempt.
}

// RobotsUserAgent returns the configured crawler identifier used for robots evaluation.
func (sess *Session) RobotsUserAgent() string {
	return sess.browserConfig.GetUserAgent()
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
		if sess.recorderCertificateSPKI == "" {
			return "", fmt.Errorf("recorder certificate SPKI is required when proxying is enabled")
		}
		// Send Chromium's own background HTTP(S) requests to an unused local
		// port so they fail instead of entering the crawl recordings.
		// startBrowserSession creates a separate browser context for the page
		// being crawled and configures it to use the real recorderproxy.
		proxy := "http://127.0.0.1:1"
		args = append(args,
			"--proxy-server="+proxy,
			"--proxy-bypass-list=<-loopback>",
			"--ignore-certificate-errors-spki-list="+sess.recorderCertificateSPKI,
		)
	}

	launch := launch{
		Headless: true,
		Args:     args,
	}

	b, err := json.Marshal(launch)
	if err != nil {
		return "", err
	}

	q := browserWsEndpoint.Query()

	q.Set("timeout", strconv.Itoa(sess.browserTimeout))
	q.Set("trackingId", sess.browserTrackingID())
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

func (sess *Session) startBrowserSession(ctx context.Context) (context.Context, func(), error) {
	log := sess.logger

	browserCtx, browserCancel := context.WithCancelCause(ctx)
	sess.browserCancel = browserCancel
	allocatorContext, allocatorCancel := newRemoteAllocator(browserCtx, sess.browserWsEndpoint)
	browserOptions := []chromedp.ContextOption{chromedp.WithErrorf(chromedpErrorf(log))}
	if sess.proxyHost != "" {
		// Route the page being crawled through its recorderproxy slot.
		// Chromium's background services keep the unused proxy configured
		// in compileBrowserWebsocketEndpoint.
		browserOptions = append(browserOptions, chromedp.WithNewBrowserContext(func(params *target.CreateBrowserContextParams) {
			params.ProxyServer = "http://" + sess.proxyHost + ":" + strconv.Itoa(sess.proxyPort+sess.Id-1)
			params.ProxyBypassList = "<-loopback>"
		}))
	}
	cdpCtx, cdpCancel := chromedp.NewContext(allocatorContext, browserOptions...)
	sess.ctx = cdpCtx
	cleanup := func() {
		_ = sess.stopChildren()
		browserCancel(nil)
		cdpCancel()
		allocatorCancel()
	}

	version, err := chromedp.CallBrowser(sess.ctx, browser.GetVersion, cdp.Empty{})
	if err != nil {
		return nil, cleanup, fmt.Errorf("failed to start browser: %w", err)
	}
	sess.browserVersion = version.Product

	sess.UserAgent = strings.ReplaceAll(version.UserAgent, "HeadlessChrome", "Chrome")
	if sess.browserConfig.UserAgent != "" {
		sess.UserAgent += " " + sess.browserConfig.UserAgent
	}

	log.Log(context.Background(), LevelTrace, "Browser version selected", "version", version.Product, "userAgent", sess.UserAgent, "browserTrackingId", sess.browserTrackingID())

	if err := sess.initListeners(cdpCtx); err != nil {
		return nil, cleanup, err
	}

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

	if err := sess.configureHarvestTarget(sess.ctx); err != nil {
		return nil, cleanup, err
	}

	// run task list
	if err := chromedp.Do(sess.ctx,
		cdpAction(serviceworker.Enable, cdp.Empty{}),
		chromedp.Emulate(deviceInfo),
		chromedp.Func(prepareRemotePage),
		cdpAction(network.SetCookies, network.SetCookiesParams{Cookies: sess.getCookieParams(sess.RequestedUrl)}),
		cdpAction(runtime.Enable, cdp.Empty{}),
	); err != nil {
		return nil, cleanup, err
	}

	return cdpCtx, cleanup, nil
}

// prepareRemotePage establishes the page lifecycle state expected by fetch
// scripts. Remote browser backends may leave a newly created target in the
// background, which suppresses visibility-dependent browser behavior.
func prepareRemotePage(ctx context.Context, t *chromedp.Target) error {
	if _, err := cdp.Call(ctx, t, page.Enable, page.EnableParams{}); err != nil {
		return fmt.Errorf("enable page domain: %w", err)
	}
	if _, err := cdp.Call(ctx, t, page.SetLifecycleEventsEnabled, page.SetLifecycleEventsEnabledParams{Enabled: true}); err != nil {
		return fmt.Errorf("enable load events: %w", err)
	}
	if _, err := cdp.Call(ctx, t, page.BringToFront, cdp.Empty{}); err != nil {
		return fmt.Errorf("bring remote page to front: %w", err)
	}
	return nil
}

func (sess *Session) navigate(loadCtx context.Context) error {
	tree, err := chromedp.Call(loadCtx, page.GetFrameTree, cdp.Empty{})
	if err != nil {
		return err
	}
	sess.pageLoad.start(tree.FrameTree.Frame.ID)
	// Only the root target needs interception. Child documents cannot be main.
	if _, err := chromedp.Call(loadCtx, fetch.Enable, fetch.EnableParams{Patterns: []*fetch.RequestPattern{
		{URLPattern: "http://*", ResourceType: network.ResourceTypeDocument, RequestStage: fetch.RequestStageRequest},
		{URLPattern: "https://*", ResourceType: network.ResourceTypeDocument, RequestStage: fetch.RequestStageRequest},
	}}); err != nil {
		return err
	}
	res, err := chromedp.Call(loadCtx, page.Navigate, page.NavigateParams{URL: sess.RequestedUrl.Uri, TransitionType: page.TransitionTypeOther})
	if err != nil {
		return err
	}
	sess.mainFrameID = res.FrameID
	sess.pageLoad.selectNavigation(res.FrameID, res.LoaderID)
	if res.ErrorText != "" && res.ErrorText != "net::ERR_ABORTED" {
		return fmt.Errorf("navigation: %s", res.ErrorText)
	}
	return nil
}

func (sess *Session) runOnLoadBehavior(loadCtx context.Context) error {
	if err := loadCtx.Err(); err != nil {
		return err
	}

	if err := sess.executeScripts(loadCtx, configV1.BrowserScript_ON_LOAD); err != nil {
		sess.logger.Warn("Failed to execute scripts", "phase", configV1.BrowserScript_ON_LOAD.String(), "error", err)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return err
		}
	}
	return loadCtx.Err()
}

func (sess *Session) getCookieParams(uri *frontierV1.QueuedUri) []*network.CookieParam {
	log := sess.logger

	if cookieCount := len(uri.GetCookies()); cookieCount > 0 {
		log.Log(context.Background(), LevelTrace, "Restoring browser cookies", "cookieCount", cookieCount)
	}
	cookies := make([]*network.CookieParam, len(uri.GetCookies()))
	for i, c := range uri.GetCookies() {
		cookies[i] = &network.CookieParam{
			Name:     c.Name,
			Value:    c.Value,
			URL:      uri.Uri,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HTTPOnly: c.HttpOnly,
			SameSite: network.CookieSameSite(c.SameSite),
			Expires:  cdp.TimeSinceEpoch(c.Expires),
		}
	}
	return cookies
}

func (sess *Session) waitForNetworkIdle(ctx context.Context, maxIdleTime time.Duration) error {
	return sess.networkTracker.waitForIdle(ctx, networkSettleIdleTime(maxIdleTime))
}

func (sess *Session) waitForBrowserReadiness(ctx context.Context, maxIdleTime time.Duration) error {
	reason, err := waitForReadiness(ctx, networkSettleIdleTime(maxIdleTime), maxIdleTime,
		sess.networkTracker.updates, true, func() readinessSnapshot {
			state := sess.networkTracker.snapshot()
			sess.childrenMu.Lock()
			state.requiredPending += sess.childSetups
			sess.childrenMu.Unlock()
			return state
		})
	sess.readinessReason = reason
	return err
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

func pageloadTimeoutError(uri, phase string) error {
	detail := "Pageload timed out. Url: " + uri
	if phase != "" {
		detail = fmt.Sprintf("Pageload timed out while waiting for %s. Url: %s", phase, uri)
	}
	return bcerrors.New(-5004, "Runtime exceeded", detail)
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
