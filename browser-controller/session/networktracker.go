package session

import (
	"context"
	"sync"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/syncx"
	browserurl "github.com/NationalLibraryOfNorway/veidemann/browser-controller/url"
	"github.com/chromedp/cdproto/network"
)

type networkActivityTracker struct {
	mu         sync.Mutex
	inflight   map[network.RequestID]network.ResourceType
	lastChange time.Time
	updates    chan struct{}
}

func newNetworkActivityTracker() *networkActivityTracker {
	return &networkActivityTracker{
		inflight:   make(map[network.RequestID]network.ResourceType),
		lastChange: time.Now(),
		updates:    make(chan struct{}, 1),
	}
}

func shouldTrackNetworkRequest(ev *network.EventRequestWillBeSent) bool {
	if ev == nil || ev.Request == nil {
		return false
	}
	return shouldTrackObservedRequest(ev.Type, ev.Request.URL, ev.FrameID != "" || ev.LoaderID != "")
}

func shouldTrackObservedRequest(resourceType network.ResourceType, rawURL string, hasContext bool) bool {
	if !hasContext {
		return false
	}
	return shouldTrackNetworkResource(resourceType, rawURL)
}

func shouldTrackNetworkResource(resourceType network.ResourceType, rawURL string) bool {
	switch resourceType {
	case network.ResourceTypeDocument,
		network.ResourceTypeStylesheet,
		network.ResourceTypeImage,
		network.ResourceTypeMedia,
		network.ResourceTypeFont,
		network.ResourceTypeScript,
		network.ResourceTypeTextTrack:
		// These resource types directly affect what becomes visible after scrolling.
		// Background XHR/Fetch/Other traffic is handled separately by crawl-log
		// completion and should not pin the render-settle loop.
	case network.ResourceTypeWebSocket,
		network.ResourceTypeEventSource,
		network.ResourceTypePing,
		network.ResourceTypeCSPViolationReport,
		network.ResourceTypePreflight,
		network.ResourceTypeXHR,
		network.ResourceTypeFetch,
		network.ResourceTypeManifest,
		network.ResourceTypeSignedExchange,
		network.ResourceTypeFedCM,
		network.ResourceTypeOther,
		network.ResourceTypePrefetch:
		return false
	default:
		return false
	}

	if browserurl.IsBrowserLocal(rawURL) {
		return false
	}
	return true
}

func (t *networkActivityTracker) noteRequestStart(ev *network.EventRequestWillBeSent) {
	if ev == nil || ev.Request == nil {
		return
	}
	t.noteObservedRequestStart(ev.Type, ev.Request.URL, ev.FrameID != "" || ev.LoaderID != "", ev.RequestID)
}

func (t *networkActivityTracker) noteObservedRequestStart(resourceType network.ResourceType, rawURL string, hasContext bool, networkID network.RequestID) {
	if !shouldTrackObservedRequest(resourceType, rawURL, hasContext) {
		return
	}

	t.mu.Lock()
	if networkID != "" {
		t.inflight[networkID] = resourceType
	}
	t.lastChange = time.Now()
	t.mu.Unlock()
	t.notify()
}

func (t *networkActivityTracker) noteRequestData(requestID network.RequestID) {
	if requestID == "" {
		return
	}

	t.mu.Lock()
	if _, ok := t.inflight[requestID]; !ok {
		t.mu.Unlock()
		return
	}
	t.lastChange = time.Now()
	t.mu.Unlock()
	t.notify()
}

func (t *networkActivityTracker) noteRequestDone(requestID network.RequestID) {
	if requestID == "" {
		return
	}

	t.mu.Lock()
	if _, ok := t.inflight[requestID]; !ok {
		t.mu.Unlock()
		return
	}
	delete(t.inflight, requestID)
	t.lastChange = time.Now()
	t.mu.Unlock()
	t.notify()
}

func (t *networkActivityTracker) waitForIdle(ctx context.Context, idleTime time.Duration) error {
	if t == nil || idleTime <= 0 {
		return nil
	}
	waitStart := time.Now()

	for {
		if err := contextWaitError(ctx.Err()); err != nil {
			return err
		}

		t.mu.Lock()
		lastChange := t.lastChange
		updates := t.updates
		t.mu.Unlock()

		quietSince := lastChange
		if quietSince.Before(waitStart) {
			quietSince = waitStart
		}
		idleFor := time.Since(quietSince)
		if idleFor >= idleTime {
			return nil
		}

		waitTimer := time.NewTimer(idleTime - idleFor)

		select {
		case <-ctx.Done():
			waitTimer.Stop()
			return contextWaitError(ctx.Err())
		case <-updates:
			waitTimer.Stop()
		case <-waitTimer.C:
			return nil
		}
	}
}

func (t *networkActivityTracker) notify() {
	select {
	case t.updates <- struct{}{}:
	default:
	}
}

func contextWaitError(err error) error {
	switch err {
	case nil:
		return nil
	case context.DeadlineExceeded:
		return syncx.ErrExceededMaxTime
	case context.Canceled:
		return syncx.ErrCancelled
	default:
		return err
	}
}
