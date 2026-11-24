package session

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"sync"
	"time"

	bcerrors "github.com/NationalLibraryOfNorway/veidemann/browser-controller/errors"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/requests"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/target"
)

// Diagnostic observations never participate in recording or readiness decisions.
// Copies leave this mutex before logging; the ordered CDP reader updates state.
type harvestDiagnostics struct {
	mu                            sync.Mutex
	phase                         string
	phaseStart, deadline          time.Time
	failurePhase                  string
	requests                      map[network.RequestID]requestDiagnostic
	children                      map[string]int
	configured, resumed, detached int
	completed, failed, canceled   int
	httpErrors                    int
	collectionPhase               string
}

type requestDiagnostic struct {
	Session           target.SessionID     `json:"cdpSessionId"`
	ID                network.RequestID    `json:"browserRequestId"`
	URL               string               `json:"url"`
	Method            string               `json:"method"`
	Type              network.ResourceType `json:"resourceType"`
	Status            int64                `json:"httpStatus,omitempty"`
	started, progress time.Time
	required          bool
}

type pendingRequestDiagnostic struct {
	requestDiagnostic
	AgeMs  int64 `json:"ageMs"`
	IdleMs int64 `json:"idleMs"`
}

func newHarvestDiagnostics() *harvestDiagnostics {
	return &harvestDiagnostics{requests: make(map[network.RequestID]requestDiagnostic), children: make(map[string]int)}
}

func boundedLogURL(raw string) string { bounded, _ := requests.BoundedURLForLog(raw); return bounded }

func (sess *Session) setPhase(ctx context.Context, phase, message string) {
	sess.diagnostics.mu.Lock()
	sess.diagnostics.phase, sess.diagnostics.phaseStart = phase, time.Now()
	sess.diagnostics.deadline, _ = ctx.Deadline()
	sess.diagnostics.mu.Unlock()
	if message != "" {
		sess.logger.Info(message, "phase", phase)
	}
}

func (sess *Session) failPhase(phase string) {
	sess.diagnostics.mu.Lock()
	if sess.diagnostics.failurePhase == "" {
		sess.diagnostics.failurePhase = phase
	}
	sess.diagnostics.mu.Unlock()
}

func (sess *Session) currentPhase() string {
	sess.diagnostics.mu.Lock()
	defer sess.diagnostics.mu.Unlock()
	return sess.diagnostics.phase
}

func (sess *Session) pendingDiagnostics() []any {
	state := sess.networkTracker.snapshot()
	sess.childrenMu.Lock()
	setups := sess.childSetups
	sess.childrenMu.Unlock()
	now := time.Now()
	sess.diagnostics.mu.Lock()
	pending := make([]requestDiagnostic, 0)
	for _, request := range sess.diagnostics.requests {
		if request.required {
			pending = append(pending, request)
		}
	}
	sess.diagnostics.mu.Unlock()
	sort.Slice(pending, func(i, j int) bool {
		if pending[i].started.Equal(pending[j].started) {
			return string(pending[i].Session)+string(pending[i].ID) < string(pending[j].Session)+string(pending[j].ID)
		}
		return pending[i].started.Before(pending[j].started)
	})
	if len(pending) > 5 {
		pending = pending[:5]
	}
	oldest := make([]pendingRequestDiagnostic, 0, len(pending))
	for _, request := range pending {
		oldest = append(oldest, pendingRequestDiagnostic{requestDiagnostic: request, AgeMs: now.Sub(request.started).Milliseconds(), IdleMs: now.Sub(request.progress).Milliseconds()})
	}
	return []any{"pendingRequests", state.requiredPending, "pendingChildSetups", setups, "oldestPendingRequests", oldest}
}

// Keep progress alive through cancellation: capture/finalization have their own
// budgets. The caller joins it before emitting the terminal summary.
func (sess *Session) startProgress(start time.Time) func() {
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				sess.diagnostics.mu.Lock()
				phase, since, deadline := sess.diagnostics.phase, sess.diagnostics.phaseStart, sess.diagnostics.deadline
				sess.diagnostics.mu.Unlock()
				if phase == "" || time.Since(since) < 10*time.Second {
					continue
				}
				attrs := []any{"phase", phase, "elapsedMs", time.Since(start).Milliseconds(), "phaseElapsedMs", time.Since(since).Milliseconds()}
				if !deadline.IsZero() {
					attrs = append(attrs, "remainingMs", max(int64(0), time.Until(deadline).Milliseconds()))
				}
				attrs = append(attrs, sess.pendingDiagnostics()...)
				sess.logger.Info("Harvest still in progress", attrs...)
			}
		}
	}()
	return func() { close(stop); <-done }
}

func (sess *Session) logHarvestResult(parent context.Context, start time.Time, err error, recorded int32, bytes int64, outlinks int, screenshot string) {
	sess.diagnostics.mu.Lock()
	failurePhase := sess.diagnostics.failurePhase
	sess.diagnostics.mu.Unlock()
	outcome, level := "success", slog.LevelInfo
	if err != nil {
		outcome, level = "failure", slog.LevelError
		var specific bcerrors.FetchError
		if parent.Err() == context.Canceled && errors.Is(err, context.Canceled) && sess.Reusable() && failurePhase != "child_setup" && !errors.As(err, &specific) {
			outcome, level = "canceled", slog.LevelInfo
		}
	}
	reason := sess.readinessReason
	if reason == "setup" || reason == "navigation" || reason == "" {
		reason = "operational_failure"
		if outcome == "canceled" {
			reason = "canceled"
		} else if parent.Err() == context.DeadlineExceeded {
			reason = "harvest_deadline"
		}
	}
	sess.diagnostics.mu.Lock()
	phase := sess.diagnostics.failurePhase
	if phase == "" && err != nil {
		phase = sess.diagnostics.phase
	}
	children := make(map[string]int, len(sess.diagnostics.children))
	for kind, count := range sess.diagnostics.children {
		children[kind] = count
	}
	attrs := []any{"outcome", outcome, "stoppingReason", reason, "durationMs", time.Since(start).Milliseconds(),
		"recordedResources", recorded, "recordedBytes", bytes, "outlinks", outlinks, "screenshot", screenshot,
		"childSessions", children, "childrenConfigured", sess.diagnostics.configured, "childrenResumed", sess.diagnostics.resumed,
		"childrenDetached", sess.diagnostics.detached, "browserRequestsCompleted", sess.diagnostics.completed,
		"browserRequestsFailed", sess.diagnostics.failed, "browserHttpErrors", sess.diagnostics.httpErrors, "browserRequestsCanceled", sess.diagnostics.canceled, "reusable", sess.Reusable()}
	if sess.diagnostics.collectionPhase != "" {
		attrs = append(attrs, "collectionPhase", sess.diagnostics.collectionPhase)
	}
	sess.diagnostics.mu.Unlock()
	if err != nil {
		attrs = append(attrs, "phase", phase, "errorCode", bcerrors.CommonsError(err).Code, "error", err)
	}
	sess.logger.Log(context.Background(), level, "Harvest finished", attrs...)
}

func (sess *Session) observeRequest(ctx context.Context, ev *network.EventRequestWillBeSent) {
	now := time.Now()
	key := observedRequestID(ctx, ev.RequestID)
	request := requestDiagnostic{Session: sessionIDFromContext(ctx), ID: ev.RequestID, URL: boundedLogURL(ev.Request.URL), Method: ev.Request.Method, Type: ev.Type, started: now, progress: now,
		required: shouldTrackObservedRequest(ev.Type, ev.Request.URL, ev.FrameID != "" || ev.LoaderID != "") && requests.ClassifyReadiness(ev.Type.String()) == requests.Required}
	sess.diagnostics.mu.Lock()
	previous, exists := sess.diagnostics.requests[key]
	sess.diagnostics.requests[key] = request
	sess.diagnostics.mu.Unlock()
	if ev.RedirectResponse != nil {
		if !exists {
			previous = request
			previous.URL = boundedLogURL(ev.RedirectResponse.URL)
		}
		previous.Status = ev.RedirectResponse.Status
		sess.logRequest(LevelTrace, "Browser request redirected", previous, "redirectUrl", request.URL)
	}
	sess.logRequest(LevelTrace, "Browser request started", request)
}

func (sess *Session) observeResponse(ctx context.Context, ev *network.EventResponseReceived) {
	if ev.Response == nil {
		return
	}
	key := observedRequestID(ctx, ev.RequestID)
	sess.diagnostics.mu.Lock()
	request, ok := sess.diagnostics.requests[key]
	if ok {
		if request.Status < 400 && ev.Response.Status >= 400 {
			sess.diagnostics.httpErrors++
		}
		request.Status = ev.Response.Status
		request.progress = time.Now()
		sess.diagnostics.requests[key] = request
	}
	sess.diagnostics.mu.Unlock()
	if ok {
		sess.logRequest(LevelTrace, "Browser response received", request)
	}
}

func (sess *Session) observeData(ctx context.Context, id network.RequestID) {
	key := observedRequestID(ctx, id)
	sess.diagnostics.mu.Lock()
	if request, ok := sess.diagnostics.requests[key]; ok {
		request.progress = time.Now()
		sess.diagnostics.requests[key] = request
	}
	sess.diagnostics.mu.Unlock()
}

func (sess *Session) observeRequestEnd(ctx context.Context, id network.RequestID, failure string, canceled bool) {
	key := observedRequestID(ctx, id)
	sess.diagnostics.mu.Lock()
	request, ok := sess.diagnostics.requests[key]
	delete(sess.diagnostics.requests, key)
	if canceled {
		sess.diagnostics.canceled++
	} else if failure != "" {
		sess.diagnostics.failed++
	} else {
		sess.diagnostics.completed++
	}
	sess.diagnostics.mu.Unlock()
	if !ok {
		request = requestDiagnostic{Session: sessionIDFromContext(ctx), ID: id}
	}
	outcome := "completed"
	if canceled {
		outcome = "canceled"
	} else if failure != "" {
		outcome = "failed"
	} else if request.Status >= 400 {
		outcome = "http_error"
	}
	attrs := []any{"outcome", outcome}
	if failure != "" {
		attrs = append(attrs, "browserError", failure, "canceled", canceled)
	}
	sess.logRequest(slog.LevelDebug, "Browser request finished", request, attrs...)
}

func (sess *Session) logRequest(level slog.Level, message string, request requestDiagnostic, extra ...any) {
	if !sess.logger.Enabled(context.Background(), level) {
		return
	}
	attrs := []any{"cdpSessionId", request.Session, "browserRequestId", request.ID}
	if request.URL != "" {
		attrs = append(attrs, "url", request.URL)
	}
	if request.Method != "" {
		attrs = append(attrs, "method", request.Method)
	}
	if request.Type != "" {
		attrs = append(attrs, "resourceType", request.Type)
	}
	if request.Status >= 100 && request.Status <= 599 {
		attrs = append(attrs, "httpStatus", request.Status)
	}
	if !request.started.IsZero() {
		attrs = append(attrs, "durationMs", time.Since(request.started).Milliseconds())
	}
	attrs = append(attrs, extra...)
	sess.logger.Log(context.Background(), level, message, attrs...)
}

func (sess *Session) forgetRequests(session target.SessionID) {
	sess.diagnostics.mu.Lock()
	for key, request := range sess.diagnostics.requests {
		if request.Session == session {
			delete(sess.diagnostics.requests, key)
		}
	}
	sess.diagnostics.mu.Unlock()
}
