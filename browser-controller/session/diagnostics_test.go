package session

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/chromedp/cdproto"
	jsonv2 "github.com/chromedp/cdproto/cdp/jsonv2"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/target"
)

type capturedLogs struct {
	mu   sync.Mutex
	data bytes.Buffer
}

func (c *capturedLogs) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.data.Write(p)
}
func (c *capturedLogs) records(t *testing.T) []map[string]any {
	t.Helper()
	c.mu.Lock()
	raw := append([]byte(nil), c.data.Bytes()...)
	c.mu.Unlock()
	decoder := json.NewDecoder(bytes.NewReader(raw))
	var records []map[string]any
	for {
		var rawRecord json.RawMessage
		err := decoder.Decode(&rawRecord)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		keys := json.NewDecoder(bytes.NewReader(rawRecord))
		if _, err := keys.Token(); err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for keys.More() {
			key, err := keys.Token()
			if err != nil {
				t.Fatal(err)
			}
			field := key.(string)
			if seen[field] {
				t.Fatalf("duplicate JSON key %q in %s", field, rawRecord)
			}
			seen[field] = true
			var value json.RawMessage
			if err := keys.Decode(&value); err != nil {
				t.Fatal(err)
			}
		}
		var record map[string]any
		if err := json.Unmarshal(rawRecord, &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}
func (c *capturedLogs) matching(t *testing.T, message string) []map[string]any {
	var found []map[string]any
	for _, record := range c.records(t) {
		if record["msg"] == message {
			found = append(found, record)
		}
	}
	return found
}
func captureLogs(sess *Session, level slog.Level) *capturedLogs {
	logs := &capturedLogs{}
	sess.logger = slog.New(slog.NewJSONHandler(logs, &slog.HandlerOptions{Level: level})).With("component", "browser-controller", "slot", sess.Id)
	return logs
}
func diagnosticContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, childContextKey{}, &childSession{id: target.SessionID(id)})
}

func TestRequestDiagnosticsKeepSessionsAndRedirectsIndependent(t *testing.T) {
	sess := New(1)
	logs := captureLogs(sess, LevelTrace)
	root, child := diagnosticContext(t.Context(), "root"), diagnosticContext(t.Context(), "child")
	request := func(ctx context.Context, url string, redirect *network.Response) {
		sess.onNetworkEventRequestWillBeSent(ctx, &network.EventRequestWillBeSent{RequestID: "same", FrameID: "frame", Type: network.ResourceTypeImage, Request: &network.Request{URL: url, Method: "GET"}, RedirectResponse: redirect}, 1)
	}
	request(root, "https://example.com/first", nil)
	request(child, "https://child.example/image", nil)
	request(root, "https://example.com/final", &network.Response{URL: "https://example.com/first", Status: 302})
	sess.onNetworkEventResponseReceived(root, &network.EventResponseReceived{RequestID: "same", Response: &network.Response{Status: 403}}, 1)
	sess.onNetworkEventLoadingFailed(root, &network.EventLoadingFailed{RequestID: "same", ErrorText: "net::ERR_FAILED"}, 1)
	failed := logs.matching(t, "Browser request finished")
	if len(failed) != 1 || failed[0]["url"] != "https://example.com/final" || failed[0]["httpStatus"] != float64(403) || failed[0]["cdpSessionId"] != "root" || failed[0]["outcome"] != "failed" {
		t.Fatal(failed)
	}
	redirects := logs.matching(t, "Browser request redirected")
	if len(redirects) != 1 || redirects[0]["url"] != "https://example.com/first" || redirects[0]["redirectUrl"] != "https://example.com/final" {
		t.Fatal(redirects)
	}
	if sess.networkTracker.snapshot().requiredPending != 1 {
		t.Fatal("diagnostics changed child readiness")
	}
	sess.forgetRequests("child")
	sess.networkTracker.removeSession("child")
	if len(sess.diagnostics.requests) != 0 || sess.networkTracker.snapshot().requiredPending != 0 {
		t.Fatal("detached metadata survived")
	}
	// An unmatched failure must still expose its session, ID, and original error.
	sess.observeRequestEnd(child, "missing", "net::ERR_ABORTED", true)
	if got := logs.matching(t, "Browser request finished"); len(got) != 2 || got[1]["browserRequestId"] != "missing" || got[1]["outcome"] != "canceled" {
		t.Fatal(got)
	}
}

func TestRequestDiagnosticsBoundsAndLevelFiltering(t *testing.T) {
	for _, level := range []slog.Level{slog.LevelInfo, slog.LevelDebug, LevelTrace} {
		t.Run(level.String(), func(t *testing.T) {
			sess := New(1)
			logs := captureLogs(sess, level)
			ctx := diagnosticContext(t.Context(), "root")
			for i, url := range []string{"data:image/png;base64,PRIVATE_IMAGE_PAYLOAD", "https://example.com/" + strings.Repeat("æ", 600)} {
				id := network.RequestID(string(rune('a' + i)))
				sess.observeRequest(ctx, &network.EventRequestWillBeSent{RequestID: id, FrameID: "frame", Type: network.ResourceTypeImage, Request: &network.Request{URL: url, Method: "GET"}})
				sess.observeResponse(ctx, &network.EventResponseReceived{RequestID: id, Response: &network.Response{Status: 200}})
				sess.observeRequestEnd(ctx, id, "", false)
			}
			starts := logs.matching(t, "Browser request started")
			if (level == LevelTrace && len(starts) != 2) || (level != LevelTrace && len(starts) != 0) {
				t.Fatal("incorrect start event level", starts)
			}
			rows := logs.matching(t, "Browser request finished")
			if level == slog.LevelInfo {
				if len(rows) != 0 {
					t.Fatal(rows)
				}
				return
			}
			if level == slog.LevelDebug && len(logs.records(t)) != 2 {
				t.Fatal("DEBUG contains intermediate events", logs.records(t))
			}
			if len(rows) != 2 || rows[0]["url"] != "data:[omitted]" || len(rows[1]["url"].(string)) > 512 {
				t.Fatal(rows)
			}
		})
	}
}

func TestProgressSnapshotsAreBoundedAndStopBeforeSummary(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sess := New(2)
		logs := captureLogs(sess, slog.LevelInfo)
		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		defer cancel()
		sess.setPhase(ctx, "readiness", "Waiting for page activity to settle")
		for i := 0; i < 8; i++ {
			sess.onNetworkEventRequestWillBeSent(diagnosticContext(ctx, "root"), &network.EventRequestWillBeSent{RequestID: network.RequestID(string(rune('a' + i))), FrameID: "main", Type: network.ResourceTypeImage, Request: &network.Request{URL: "https://example.com/image", Method: "GET"}}, 1)
		}
		sess.onNetworkEventDataReceived(diagnosticContext(ctx, "root"), &network.EventDataReceived{RequestID: "a"}, 1)
		stop := sess.startProgress(time.Now())
		time.Sleep(25 * time.Second)
		stop()
		progress := logs.matching(t, "Harvest still in progress")
		if len(progress) != 2 || progress[0]["phase"] != "readiness" || progress[0]["remainingMs"] != float64(20000) || len(progress[0]["oldestPendingRequests"].([]any)) != 5 || progress[0]["pendingRequests"] != float64(8) {
			t.Fatal(progress)
		}
		sess.readinessReason = "required_request_stalled"
		sess.logHarvestResult(ctx, time.Now(), nil, 3, 512, 4, "stored")
		time.Sleep(10 * time.Second)
		rows := logs.records(t)
		if rows[len(rows)-1]["msg"] != "Harvest finished" || len(logs.matching(t, "Harvest finished")) != 1 {
			t.Fatal(rows)
		}
	})
}

func TestTeardownFailureAppearsInTerminalSummary(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sess := New(3)
		logs := captureLogs(sess, slog.LevelInfo)
		start := time.Now()
		release := make(chan struct{})
		sess.setPhase(t.Context(), "browser_teardown", "")
		err := sess.closeBrowser(func() { <-release })
		close(release)
		synctest.Wait()
		sess.failPhase("browser_teardown")
		sess.logHarvestResult(t.Context(), start, err, 5, 1234, 2, "stored")
		rows := logs.matching(t, "Harvest finished")
		if len(rows) != 1 || rows[0]["outcome"] != "failure" || rows[0]["phase"] != "browser_teardown" || rows[0]["reusable"] != false || rows[0]["durationMs"] != float64(5000) {
			t.Fatal(rows)
		}
	})
}

func TestUnusedBrowserEventsAreSilentAndMalformedConsumedEventsWarn(t *testing.T) {
	sess := New(1)
	logs := captureLogs(sess, slog.LevelDebug)
	ctx := context.WithValue(t.Context(), harvestLogContextKey{}, sess.logger.With("harvestId", "event-attempt"))
	var events targetEvents
	events.listen(ctx, "root", func(any) { t.Error("malformed/unused event was delivered") })
	for _, method := range []cdproto.MethodType{"Page.frameScheduledNavigation", "Page.frameClearedScheduledNavigation"} {
		events.dispatch(&cdproto.Message{SessionID: "root", Method: method, Params: jsonv2.Value(`{}`)})
	}
	if len(logs.records(t)) != 0 {
		t.Fatal("unused events produced noise")
	}
	events.dispatch(&cdproto.Message{SessionID: "root", Method: cdproto.EventNetworkLoadingFinished, Params: jsonv2.Value(`{"requestId":{}}`)})
	rows := logs.matching(t, "Malformed browser event")
	if len(rows) != 1 || rows[0]["level"] != "WARN" || rows[0]["cdpSessionId"] != "root" || rows[0]["harvestId"] != "event-attempt" {
		t.Fatal(rows)
	}
}

func TestStopReasonSurvivesCaptureAndFailureSummary(t *testing.T) {
	sess := New(1)
	logs := captureLogs(sess, slog.LevelInfo)
	for _, reason := range []string{"quiet", "media_allowance_elapsed", "required_request_stalled", "page_load_deadline"} {
		sess.readinessReason = reason
		sess.logHarvestResult(t.Context(), time.Now(), nil, 1, 200, 0, "disabled")
	}
	rows := logs.matching(t, "Harvest finished")
	for i, reason := range []string{"quiet", "media_allowance_elapsed", "required_request_stalled", "page_load_deadline"} {
		if rows[i]["stoppingReason"] != reason {
			t.Fatal(rows)
		}
	}
	sess.failPhase("capture_screenshot")
	sess.logHarvestResult(t.Context(), time.Now(), errors.New("screenshot unavailable"), 1, 200, 0, "not_captured")
	last := logs.matching(t, "Harvest finished")[4]
	if last["phase"] != "capture_screenshot" || last["errorCode"] != float64(-5) {
		t.Fatal(last)
	}
}

func TestBrowserTrackingIDFollowsAttemptInsteadOfReusingSlot(t *testing.T) {
	sess := New(1)
	previous := ""
	for _, attempt := range []string{"first-attempt-uuid", "second-attempt-uuid"} {
		sess.attemptID = attempt
		endpoint, err := sess.compileBrowserWebsocketEndpoint()
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := url.Parse(endpoint)
		if err != nil {
			t.Fatal(err)
		}
		tracking := parsed.Query().Get("trackingId")
		if len(tracking) != 22 || tracking == previous || tracking == "1" {
			t.Fatal(tracking)
		}
		previous = tracking
	}
}
