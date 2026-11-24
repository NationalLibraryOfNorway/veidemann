package session

import (
	"context"
	"github.com/chromedp/cdproto/target"
	"strings"
	"sync"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/requests"
	browserurl "github.com/NationalLibraryOfNorway/veidemann/browser-controller/url"
	"github.com/chromedp/cdproto/network"
)

type networkActivityTracker struct {
	mu                 sync.Mutex
	inflight           map[network.RequestID]network.ResourceType
	lastRequiredChange time.Time
	lastMediaChange    time.Time
	mediaDeadline      time.Time
	updates            chan struct{}
}

func newNetworkActivityTracker() *networkActivityTracker {
	return &networkActivityTracker{
		inflight:           make(map[network.RequestID]network.ResourceType),
		lastRequiredChange: time.Now(),
		updates:            make(chan struct{}, 1),
	}
}

func shouldTrackObservedRequest(resourceType network.ResourceType, rawURL string, hasContext bool) bool {
	if !hasContext {
		return false
	}

	if requests.ClassifyReadiness(resourceType.String()) == requests.Background {
		return false
	}

	return !browserurl.IsBrowserLocal(rawURL)
}

func (t *networkActivityTracker) noteObservedRequestStart(resourceType network.ResourceType, rawURL string, hasContext bool, networkID network.RequestID) {
	if !shouldTrackObservedRequest(resourceType, rawURL, hasContext) {
		return
	}

	t.mu.Lock()
	if networkID != "" {
		t.inflight[networkID] = resourceType
	}
	t.noteActivityLocked(resourceType)
	t.mu.Unlock()
	t.notify()
}

func (t *networkActivityTracker) noteRequestData(requestID network.RequestID) {
	if requestID == "" {
		return
	}

	t.mu.Lock()
	resourceType, ok := t.inflight[requestID]
	if !ok {
		t.mu.Unlock()
		return
	}
	t.noteActivityLocked(resourceType)
	t.mu.Unlock()
	t.notify()
}

func (t *networkActivityTracker) noteRequestDone(requestID network.RequestID) {
	if requestID == "" {
		return
	}

	t.mu.Lock()
	resourceType, ok := t.inflight[requestID]
	if !ok {
		t.mu.Unlock()
		return
	}
	delete(t.inflight, requestID)
	t.noteActivityLocked(resourceType)
	t.mu.Unlock()
	t.notify()
}

func (t *networkActivityTracker) noteActivityLocked(resourceType network.ResourceType) {
	switch requests.ClassifyReadiness(resourceType.String()) {
	case requests.Required:
		t.lastRequiredChange = time.Now()
	case requests.BoundedMedia:
		t.lastMediaChange = time.Now()
	}
}

func (t *networkActivityTracker) startMediaAllowance(duration time.Duration) {
	t.mu.Lock()
	// A single allowance is shared across script waits, settlement, and capture.
	if t.mediaDeadline.IsZero() {
		t.mediaDeadline = time.Now().Add(duration)
	}
	t.mu.Unlock()
	t.notify()
}

func (t *networkActivityTracker) snapshot() readinessSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	pending := 0
	for _, typ := range t.inflight {
		if requests.ClassifyReadiness(typ.String()) == requests.Required {
			pending++
		}
	}
	return readinessSnapshot{requiredPending: pending, requiredLast: t.lastRequiredChange, mediaLast: t.lastMediaChange, mediaDeadline: t.mediaDeadline}
}

func (t *networkActivityTracker) waitForIdle(ctx context.Context, idleTime time.Duration) error {
	if idleTime <= 0 {
		return nil
	}
	_, err := waitForReadiness(ctx, idleTime, 0, t.updates, false, t.snapshot)
	return err
}

func (t *networkActivityTracker) notify() {
	select {
	case t.updates <- struct{}{}:
	default:
	}
}

// Detachment is terminal for every request owned by that session.
func (t *networkActivityTracker) removeSession(id target.SessionID) {
	t.mu.Lock()
	for request := range t.inflight {
		if strings.HasPrefix(string(request), string(id)+":") {
			delete(t.inflight, request)
		}
	}
	t.lastRequiredChange = time.Now()
	t.mu.Unlock()
	t.notify()
}
