package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	bcerrors "github.com/NationalLibraryOfNorway/veidemann/browser-controller/errors"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/frontier"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/screenshotwriter"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func WithAdmissionToken(token string) Option {
	return newFuncOption(func(s *Session) { s.admissionToken = token })
}
func (sess *Session) NextToken() string { return sess.admissionToken }

func (sess *Session) recoverSlot(ctx context.Context) error {
	reply, err := sess.recorder.RecoverSlot(ctx, &recorderV1.RecoverSlotRequest{ProxyId: int32(sess.Id)})
	if err != nil {
		return err
	}
	if reply.GetNextToken() == "" {
		return errors.New("recorder recovery returned no admission token")
	}
	sess.admissionToken = reply.NextToken
	return nil
}

func WithRecorder(client recorderV1.RecorderControlClient) Option {
	return newFuncOption(func(s *Session) { s.recorder = client })
}
func WithCaptureTimeout(timeout time.Duration) Option {
	return newFuncOption(func(s *Session) { s.captureTimeout = timeout })
}
func WithPersistenceTimeout(timeout time.Duration) Option {
	return newFuncOption(func(s *Session) { s.persistenceTimeout = timeout })
}

var errUnusableMain = errors.New("unusable main page")

type artifactBundle struct {
	uri          string // Current URL, including same-document changes, for referrers.
	committedURL string // Network URL of the observed root document, for the recorder.
	screenshot   []byte
	cookies      []*frontierV1.Cookie
	outlinks     []string
}

// Reusable reports whether both recorder and browser teardown isolated this slot.
func (sess *Session) Reusable() bool { return !sess.poisoned }

func (sess *Session) Fetch(ctx context.Context, phs *frontierV1.PageHarvestSpec) (result *frontier.RenderResult, err error) {
	start := time.Now()
	sess.attemptID = uuid.NewString()
	sess.logger = sess.logger.With("harvestId", sess.attemptID, "pageUrl", boundedLogURL(phs.GetQueuedUri().GetUri()))
	sess.readinessReason = "setup"
	var recorded int32
	var recordedBytes int64
	var outlinkCount int
	screenshotStatus := "disabled"
	if phs.GetCrawlConfig().GetCrawlConfig().GetExtra().GetCreateScreenshot() {
		screenshotStatus = "not_captured"
	}
	sess.setPhase(ctx, "configuration", "Harvest started")
	stopProgress := sess.startProgress(start)
	defer func() {
		stopProgress()
		sess.diagnostics.mu.Lock()
		clear(sess.diagnostics.requests)
		sess.diagnostics.mu.Unlock()
		sess.logHarvestResult(ctx, start, err, recorded, recordedBytes, outlinkCount, screenshotStatus)
	}()
	maxTotal, maxIdle, err := sess.loadFetchConfig(ctx, phs)
	if err != nil {
		return nil, err
	}
	sess.logger.Info("Harvest budgets selected", "browsingTimeoutMs", maxTotal.Milliseconds(), "inactivityTimeoutMs", maxIdle.Milliseconds(), "captureTimeoutMs", sess.captureTimeout.Milliseconds(), "finalizationTimeoutMs", sess.completionDrainTimeout.Milliseconds())
	sess.setPhase(ctx, "browser_setup", "Connecting to browser")
	cdpCtx, cleanup, err := sess.startBrowserSession(ctx)
	defer func() {
		if err != nil {
			sess.failPhase(sess.currentPhase())
		}
		teardownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		sess.setPhase(teardownCtx, "browser_teardown", "Closing browser")
		closeErr := sess.closeBrowser(cleanup)
		if closeErr != nil {
			sess.failPhase("browser_teardown")
		}
		err = preferFailure(err, closeErr)
	}()
	if err != nil {
		return nil, err
	}
	sess.logger.Info("Browser ready", "browserVersion", sess.browserVersion, "browserTrackingId", sess.browserTrackingID())
	beginCtx, cancelBegin := context.WithTimeout(ctx, 10*time.Second)
	sess.setPhase(beginCtx, "recorder_begin", "Starting recording")
	if sess.admissionToken == "" {
		if err := sess.recoverSlot(beginCtx); err != nil {
			cancelBegin()
			sess.poisoned = true
			return nil, err
		}
	}
	begin, beginErr := sess.recorder.BeginHarvest(beginCtx, &recorderV1.BeginHarvestRequest{
		ProxyId: int32(sess.Id), AttemptId: sess.attemptID, AdmissionToken: sess.admissionToken,
		RequestedUri: sess.RequestedUrl.Uri, Referrer: sess.RequestedUrl.Referrer,
		DiscoveryPath: sess.RequestedUrl.DiscoveryPath, CrawlExecutionId: sess.RequestedUrl.ExecutionId,
		JobExecutionId: sess.RequestedUrl.JobExecutionId, CollectionRef: sess.CrawlConfig.CollectionRef,
		Politeness: sess.PolitenessConfig, UserAgent: sess.RobotsUserAgent(),
		FinalizationTimeoutMs: (sess.completionDrainTimeout - 5*time.Second).Milliseconds(),
	})
	cancelBegin()
	if beginErr == nil && begin.GetNextToken() == "" {
		beginErr = errors.New("recorder Begin returned no admission token")
	}
	if beginErr == nil {
		sess.admissionToken = begin.NextToken
	}
	var bundle artifactBundle
	var cutoff bool
	browseErr := beginErr
	if beginErr == nil {
		loadCtx, cancelLoad := context.WithTimeout(cdpCtx, maxTotal)
		bundle, cutoff, browseErr = sess.browseAndCapture(ctx, cdpCtx, loadCtx, maxIdle)
		cancelLoad()
	}
	outlinkCount = len(bundle.outlinks)
	if screenshotStatus != "disabled" && browseErr == nil {
		screenshotStatus = "captured"
	}
	if browseErr != nil {
		sess.failPhase(sess.currentPhase())
	}
	if childErr := sess.stopChildren(); childErr != nil && !errors.Is(browseErr, childErr) {
		browseErr = errors.Join(browseErr, childErr)
	}
	if cause := context.Cause(cdpCtx); cause != nil && !errors.Is(browseErr, cause) {
		browseErr = errors.Join(browseErr, cause)
	}
	var pageURI *string
	var links []string
	if browseErr == nil {
		pageURI = &bundle.committedURL
		links = bundle.outlinks
	}
	// Even an interrupted Begin reply can have opened the attempt. Always finish
	// after issuing Begin, including browsing failures and recovered browser panics.
	finalCtx, cancelFinal := context.WithTimeout(context.WithoutCancel(ctx), sess.completionDrainTimeout)
	sess.setPhase(finalCtx, "recorder_finalization", "Finalizing recorder work")
	reply, finishErr := sess.finishHarvest(finalCtx, pageURI, links)
	if beginErr != nil {
		if recoverErr := sess.recoverSlot(finalCtx); recoverErr != nil {
			sess.poisoned = true
			finishErr = errors.Join(finishErr, recoverErr)
		} else {
			sess.poisoned = false
		}
	}
	cancelFinal()
	if reply != nil {
		recorded, recordedBytes = reply.UriCount, reply.BytesDownloaded
		sess.logger.Info("Recorder finalization completed", "recordedResources", recorded, "recordedBytes", recordedBytes, "mainWarcId", reply.GetMain().GetWarcId(), "reusable", reply.Reusable, "failure", reply.Failure)
	}
	if finishErr != nil {
		sess.diagnostics.mu.Lock()
		sess.diagnostics.failurePhase = "recorder_finalization"
		sess.diagnostics.mu.Unlock()
		return nil, preferFailure(browseErr, finishErr)
	}
	if browseErr != nil {
		if cutoff && errors.Is(browseErr, errUnusableMain) {
			return nil, pageloadTimeoutError(sess.RequestedUrl.Uri, "acceptable main capture")
		}
		return nil, browseErr
	}
	sess.setPhase(ctx, "validate_recording", "")
	main := reply.Main
	if reply.MainCached {
		return nil, bcerrors.New(-4100, "Already seen", "Main document was found in cache")
	}
	if main.GetWarcId() == "" || main.GetError() != nil || main.GetStatusCode() < 200 {
		if cutoff {
			return nil, pageloadTimeoutError(sess.RequestedUrl.Uri, "acceptable main capture")
		}
		return nil, fmt.Errorf("no recorded main capture for %s", bundle.uri)
	}
	if sess.CrawlConfig.GetExtra().GetCreateScreenshot() {
		persistCtx, cancelPersist := context.WithTimeout(context.WithoutCancel(ctx), sess.persistenceTimeout)
		sess.setPhase(persistCtx, "screenshot_persistence", "Persisting screenshot")
		screenshotStatus = "failed"
		if main.IpAddress == "" {
			sess.logger.Warn("Missing IP address for screenshot metadata, using 127.0.0.1")
		}
		defer cancelPersist()
		if err := sess.screenShotWriter.Write(persistCtx, bundle.screenshot, screenshotwriter.Metadata{CrawlConfig: sess.CrawlConfig, CrawlLog: main, BrowserConfig: sess.browserConfig, BrowserVersion: sess.browserVersion}); err != nil {
			return nil, fmt.Errorf("persist screenshot: %w", err)
		}
		screenshotStatus = "stored"
		sess.logger.Info("Screenshot stored", "bytes", len(bundle.screenshot), "width", sess.browserConfig.WindowWidth, "height", sess.browserConfig.WindowHeight, "mainWarcId", main.WarcId)
	}
	return bundle.renderResult(sess.RequestedUrl, reply, time.Since(start)), nil
}

func (bundle *artifactBundle) renderResult(request *frontierV1.QueuedUri, reply *recorderV1.FinishHarvestReply, elapsed time.Duration) *frontier.RenderResult {
	result := &frontier.RenderResult{PageFetchTimeMs: elapsed.Milliseconds(), UriCount: reply.UriCount, BytesDownloaded: reply.BytesDownloaded}
	for _, uri := range bundle.outlinks {
		result.Outlinks = append(result.Outlinks, &frontierV1.QueuedUri{ExecutionId: request.ExecutionId, JobExecutionId: request.JobExecutionId, Uri: uri, Referrer: bundle.uri, DiscoveryPath: reply.Main.DiscoveryPath + "L", Cookies: bundle.cookies, DiscoveredTimeStamp: timestamppb.Now()})
	}
	return result
}

// Browser cancellation may wait for remote target cleanup. Bound that wait and
// stop the worker group if the browser cannot be isolated.
func (sess *Session) closeBrowser(cleanup func()) error {
	closed := make(chan struct{})
	go func() { cleanup(); close(closed) }()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-closed:
		return nil
	case <-timer.C:
		sess.poisoned = true
		return fmt.Errorf("browser shutdown timed out")
	}
}

func (sess *Session) browseAndCapture(parent, cdpCtx, loadCtx context.Context, maxIdle time.Duration) (bundle artifactBundle, cutoff bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = recoverFetchError(r)
		}
	}()
	sess.setPhase(loadCtx, "document_scripts", "Installing document scripts")
	if err = sess.registerNewDocumentScripts(loadCtx); err != nil {
		return
	}
	sess.readinessReason = "navigation"
	sess.setPhase(loadCtx, "navigation", "Navigating to page")
	err = sess.navigate(loadCtx)
	if err == nil {
		sess.setPhase(loadCtx, "main_document_load", "Waiting for main document load")
		err = sess.waitForPageLoad(loadCtx)
		if err == nil {
			sess.logger.Info("Main document loaded", "frameId", sess.mainFrameID)
		}
	}
	if err == nil {
		sess.networkTracker.startMediaAllowance(sess.mediaSettleTimeout)
		sess.setPhase(loadCtx, "on_load_scripts", "Running page scripts")
		err = sess.runOnLoadBehavior(loadCtx)
	}
	if err == nil {
		sess.setPhase(loadCtx, "readiness", "Waiting for page activity to settle")
		err = sess.waitForBrowserReadiness(loadCtx, maxIdle)
	}
	cutoff = (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, errCompletionIdleTimeout)) && parent.Err() == nil
	if errors.Is(err, context.DeadlineExceeded) && parent.Err() == nil {
		sess.readinessReason = "page_load_deadline"
	}
	if err != nil && !cutoff {
		sess.readinessReason = "operational_failure"
		if parent.Err() == context.Canceled {
			sess.readinessReason = "canceled"
		} else if parent.Err() == context.DeadlineExceeded {
			sess.readinessReason = "harvest_deadline"
		}
	}
	sess.diagnostics.mu.Lock()
	sess.diagnostics.collectionPhase = sess.diagnostics.phase
	sess.diagnostics.mu.Unlock()
	attrs := []any{"phase", sess.currentPhase(), "stoppingReason", sess.readinessReason, "captureAllowed", err == nil || cutoff}
	if cutoff {
		attrs = append(attrs, sess.pendingDiagnostics()...)
	}
	sess.logger.Info("Collection stopped", attrs...)
	if err != nil && !cutoff {
		return
	}
	captureCtx, cancel := context.WithTimeout(context.WithoutCancel(cdpCtx), sess.captureTimeout)
	defer cancel()
	sess.setPhase(captureCtx, "capture_stop_loading", "Capturing page artifacts")
	if _, err = chromedp.Call(captureCtx, page.StopLoading, cdp.Empty{}); err != nil {
		err = fmt.Errorf("capture stop loading: %w", err)
		return
	}
	sess.setPhase(captureCtx, "child_setup_drain", "")
	if err = sess.drainChildren(captureCtx, false); err != nil {
		return
	}
	bundle, err = sess.captureBundle(captureCtx)
	if err == nil {
		sess.logger.Info("Page artifacts captured", "outlinks", len(bundle.outlinks), "screenshotBytes", len(bundle.screenshot))
	}
	if err == nil {
		sess.setPhase(captureCtx, "child_setup_drain", "")
		err = sess.drainChildren(captureCtx, true)
	}
	return
}

func preferFailure(original, final error) error {
	if final == nil {
		return original
	}
	if original == nil {
		return final
	}
	// Specific recorder errors take precedence over generic CDP navigation errors.
	var specific bcerrors.FetchError
	if errors.As(final, &specific) {
		return errors.Join(final, original)
	}
	if errors.As(original, &specific) && specific.CommonsError().Code == -5004 {
		return errors.Join(bcerrors.New(-5, "Finalization failed", final.Error()), final, original)
	}
	return errors.Join(original, final)
}

func (sess *Session) finishHarvest(ctx context.Context, pageURI *string, links []string) (*recorderV1.FinishHarvestReply, error) {
	header := &recorderV1.FinishHarvestHeader{ProxyId: int32(sess.Id), AttemptId: sess.attemptID, PageUri: pageURI}
	reply, err := recorderV1.UploadFinish(ctx, sess.recorder, header, links)
	// The recorder may already have committed a finish whose reply was lost.
	if status.Code(err) == codes.Unavailable && ctx.Err() == nil {
		reply, err = recorderV1.UploadFinish(ctx, sess.recorder, header, links)
	}
	if err != nil {
		sess.poisoned = true
		return nil, fmt.Errorf("finish harvest: %w", err)
	}
	sess.poisoned = !reply.Reusable
	if failure := reply.Failure; failure != nil {
		return reply, bcerrors.New(failure.Code, failure.Msg, failure.Detail)
	}
	return reply, nil
}

func (sess *Session) captureBundle(ctx context.Context) (artifactBundle, error) {
	// A navigation during extraction invalidates every artifact, including cookies.
	// Retry the complete bundle, never combine artifacts from different documents.
	for retry := 0; retry < 2; retry++ {
		bundle := artifactBundle{}
		sess.setPhase(ctx, "capture_main_frame", "")
		before, err := chromedp.Call(ctx, page.GetFrameTree, cdp.Empty{})
		if err != nil {
			return bundle, fmt.Errorf("capture main frame: %w", err)
		}
		frame := before.FrameTree.Frame
		bundle.uri = frame.URL + frame.URLFragment
		if frame.URL == "" || strings.HasPrefix(frame.URL, "about:") || strings.HasPrefix(frame.URL, "chrome-error:") {
			return bundle, fmt.Errorf("%w: %s", errUnusableMain, frame.URL)
		}
		if sess.mainFrameID != "" && frame.ID != sess.mainFrameID {
			return bundle, fmt.Errorf("main frame changed")
		}
		var associated bool
		bundle.committedURL, associated = sess.pageLoad.documentURL(frame)
		if !associated {
			continue
		}
		if sess.CrawlConfig.GetExtra().GetCreateScreenshot() {
			capture := page.CaptureScreenshotParams{Format: page.CaptureScreenshotFormatPng}
			if sess.browserConfig.WindowWidth > 0 && sess.browserConfig.WindowHeight > 0 {
				capture.Clip = &page.Viewport{Width: float64(sess.browserConfig.WindowWidth), Height: float64(sess.browserConfig.WindowHeight), Scale: 1}
				capture.CaptureBeyondViewport = new(true)
			}
			sess.setPhase(ctx, "capture_screenshot", "")
			image, e := chromedp.Call(ctx, page.CaptureScreenshot, capture)
			if e != nil {
				return bundle, fmt.Errorf("capture screenshot: %w", e)
			}
			bundle.screenshot = image.Data
		}
		sess.setPhase(ctx, "capture_cookies", "")
		cookies, e := chromedp.Call(ctx, network.GetCookies, network.GetCookiesParams{})
		if e != nil {
			return bundle, fmt.Errorf("capture cookies: %w", e)
		}
		for _, c := range cookies.Cookies {
			bundle.cookies = append(bundle.cookies, &frontierV1.Cookie{Name: c.Name, Value: c.Value, Domain: c.Domain, Path: c.Path, Expires: float64(c.Expires), Size: int32(c.Size), HttpOnly: c.HTTPOnly, Secure: c.Secure, Session: c.Session, SameSite: c.SameSite.String()})
		}
		sess.setPhase(ctx, "extract_outlinks", "")
		for _, script := range sess.scripts.Get(configV1.BrowserScript_EXTRACT_OUTLINKS) {
			raw, e := evaluateScript(ctx, script.GetBrowserScript().GetScript())
			if e != nil {
				return bundle, fmt.Errorf("extract outlinks: %w", e)
			}
			var links []string
			if raw != nil {
				if e = json.Unmarshal(raw, &links); e != nil {
					return bundle, fmt.Errorf("decode outlinks: %w", e)
				}
			}
			for _, link := range links {
				link = strings.Trim(strings.TrimSpace(link), "\"\\")
				if link != "" && link != bundle.uri {
					bundle.outlinks = append(bundle.outlinks, link)
				}
			}
		}
		sess.setPhase(ctx, "validate_capture", "")
		after, e := chromedp.Call(ctx, page.GetFrameTree, cdp.Empty{})
		if e != nil {
			return bundle, fmt.Errorf("validate captured main frame: %w", e)
		}
		committedURL, associated := sess.pageLoad.documentURL(after.FrameTree.Frame)
		if associated && committedURL == bundle.committedURL && after.FrameTree.Frame.ID == frame.ID && after.FrameTree.Frame.LoaderID == frame.LoaderID && after.FrameTree.Frame.URL == frame.URL && after.FrameTree.Frame.URLFragment == frame.URLFragment {
			return bundle, nil
		}
	}
	return artifactBundle{}, fmt.Errorf("page changed or committed document association missing during both capture attempts")
}
