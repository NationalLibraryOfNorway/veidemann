package session

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

func TestDocumentRequestHeaders(t *testing.T) {
	sess := newDefaultSession()
	sess.attemptID = "current-attempt"
	sess.pageLoad.start("root")
	for _, frame := range []cdp.FrameID{"root", "child", ""} {
		ev := &fetch.EventRequestPaused{FrameID: frame, ResourceType: network.ResourceTypeDocument, Request: &network.Request{Headers: network.Headers{
			"x-veidemann-main-document": "forged", constants.HeaderAttemptId: sess.attemptID, "Cookie": "session=abc", "Accept": "text/html",
		}}}
		headers := make(http.Header)
		for _, h := range sess.documentRequestHeaders(ev) {
			headers.Add(h.Name, h.Value)
		}
		want := ""
		if frame == "root" {
			want = sess.attemptID
		}
		if headers.Get(constants.HeaderMainDocument) != want || len(headers.Values(constants.HeaderMainDocument)) > 1 {
			t.Fatalf("frame=%s headers=%v", frame, headers)
		}
		if headers.Get("Cookie") != "session=abc" || headers.Get("Accept") != "text/html" || headers.Get(constants.HeaderAttemptId) != sess.attemptID {
			t.Fatalf("lost headers: %v", headers)
		}
		if ev.Request.Headers["x-veidemann-main-document"] != "forged" {
			t.Fatal("mutated event")
		}
	}
}

func TestDocumentContinuationFailure(t *testing.T) {
	for _, scenario := range []string{"success", "expired", "failure", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, _, _, err := ws.UpgradeHTTP(r, w)
				if err != nil {
					t.Error(err)
					return
				}
				defer func() { _ = conn.Close() }()
				for {
					data, _, err := wsutil.ReadClientData(conn)
					if err != nil {
						return
					}
					var cmd struct {
						ID        int64                       `json:"id"`
						SessionID string                      `json:"sessionId"`
						Method    string                      `json:"method"`
						Params    fetch.ContinueRequestParams `json:"params"`
					}
					if err := json.Unmarshal(data, &cmd); err != nil {
						t.Error(err)
						return
					}
					result := any(map[string]any{})
					switch cmd.Method {
					case "Runtime.evaluate":
						result = map[string]any{"result": map[string]any{"type": "object", "className": "Window"}}
					case "Target.createTarget":
						result = map[string]any{"targetId": "root"}
					case "Target.attachToTarget":
						result = map[string]any{"sessionId": "root-session"}
					case "Fetch.continueRequest":
						if cmd.Params.RequestID != "paused" {
							t.Errorf("wrong request: %v", cmd.Params)
						}
					}
					response := map[string]any{"id": cmd.ID, "sessionId": cmd.SessionID, "result": result}
					if cmd.Method == "Fetch.continueRequest" && scenario != "success" {
						delete(response, "result")
						message := "Unsafe header"
						if scenario == "expired" {
							message = "Invalid InterceptionId."
						}
						response["error"] = map[string]any{"code": -32602, "message": message}
					}
					encoded, _ := json.Marshal(response)
					if err := wsutil.WriteServerText(conn, encoded); err != nil {
						return
					}
				}
			}))
			defer server.Close()
			timeout, stop := context.WithTimeout(t.Context(), 3*time.Second)
			defer stop()
			parent, cancel := context.WithCancelCause(timeout)
			defer cancel(nil)
			alloc, closeAlloc := newRemoteAllocator(parent, "ws"+strings.TrimPrefix(server.URL, "http"))
			defer closeAlloc()
			ctx, closeBrowser := chromedp.NewContext(alloc)
			defer closeBrowser()
			if err := chromedp.Do(ctx); err != nil {
				t.Fatal(err)
			}
			sess := newDefaultSession()
			sess.attemptID = "attempt"
			sess.browserCancel = cancel
			sess.pageLoad.start("root")
			if scenario == "cancelled" {
				cancel(context.Canceled)
			}
			sess.continueDocumentRequest(ctx, &fetch.EventRequestPaused{RequestID: "paused", FrameID: "root", ResourceType: network.ResourceTypeDocument, Request: &network.Request{Headers: network.Headers{}}})
			cause := context.Cause(parent)
			switch scenario {
			case "failure":
				if cause == nil || !strings.Contains(cause.Error(), "continue document request") {
					t.Fatalf("lost continuation error: %v", cause)
				}
			case "cancelled":
				if !errors.Is(cause, context.Canceled) {
					t.Fatalf("changed cancellation: %v", cause)
				}
			default:
				if cause != nil {
					t.Fatalf("harmless continuation ended harvest: %v", cause)
				}
			}
		})
	}
}
