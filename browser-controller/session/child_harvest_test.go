package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/screenshotwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/testutil"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"google.golang.org/grpc"
)

type childHarvestConfig struct{}

func (childHarvestConfig) GetConfigObject(_ context.Context, ref *configV1.ConfigRef) (*configV1.ConfigObject, error) {
	if ref.Id == "browser" {
		return &configV1.ConfigObject{Spec: &configV1.ConfigObject_BrowserConfig{BrowserConfig: &configV1.BrowserConfig{PageLoadTimeoutMs: 5000, MaxInactivityTimeMs: 1}}}, nil
	}
	return &configV1.ConfigObject{}, nil
}
func (childHarvestConfig) GetScripts(context.Context, *configV1.BrowserConfig) ([]*configV1.ConfigObject, error) {
	return nil, nil
}

type childHarvestRecorder struct {
	recorderReply
	begins     int
	lostBegin  bool
	recoveries int
}

func (r *childHarvestRecorder) BeginHarvest(context.Context, *recorderV1.BeginHarvestRequest, ...grpc.CallOption) (*recorderV1.BeginHarvestReply, error) {
	r.begins++
	if r.lostBegin {
		return nil, errors.New("lost begin reply")
	}
	return &recorderV1.BeginHarvestReply{NextToken: "next-token"}, nil
}

func (r *childHarvestRecorder) RecoverSlot(context.Context, *recorderV1.RecoverSlotRequest, ...grpc.CallOption) (*recorderV1.RecoverSlotReply, error) {
	r.recoveries++
	return &recorderV1.RecoverSlotReply{NextToken: "recovered-token"}, nil
}

// Drive the public harvest path, including recorder finalization, over a mock
// WebSocket backend. The iframe has no execution context until it is resumed.
func TestChildSetupFailureFinalizesHarvest(t *testing.T) {
	for _, scenario := range []string{"lost-begin", "setup-failure", "detachment", "late-setup-failure", "cancelled", "screenshot-failure", "finalization-failure"} {
		t.Run(scenario, func(t *testing.T) {
			fetchCtx, cancelFetch := context.WithCancel(t.Context())
			defer cancelFetch()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, _, _, err := ws.UpgradeHTTP(r, w)
				if err != nil {
					t.Error(err)
					return
				}
				defer func() { _ = conn.Close() }()
				emit := func(method, session string, params any) {
					raw, _ := json.Marshal(map[string]any{"method": method, "sessionId": session, "params": params})
					if err := wsutil.WriteServerText(conn, raw); err != nil {
						t.Errorf("send %s event: %v", method, err)
					}
				}
				attach := func() {
					emit("Target.attachedToTarget", "root-session", map[string]any{"sessionId": "child-session", "targetInfo": map[string]any{"targetId": "child-target", "type": "iframe"}, "waitingForDebugger": true})
				}
				navigated := false
				stopped := false
				captureFrameReads := 0
				for {
					raw, _, err := wsutil.ReadClientData(conn)
					if err != nil {
						return
					}
					var cmd struct {
						ID        int64  `json:"id"`
						SessionID string `json:"sessionId"`
						Method    string `json:"method"`
					}
					if err := json.Unmarshal(raw, &cmd); err != nil {
						t.Error(err)
						return
					}
					result := any(map[string]any{})
					switch cmd.Method {
					case "Browser.getVersion":
						result = map[string]any{"product": "Chrome/test", "userAgent": "Chrome/test"}
					case "Target.createTarget":
						result = map[string]any{"targetId": "root-target"}
					case "Target.attachToTarget":
						if cmd.SessionID != "" {
							t.Error("attached to child a second time")
						}
						result = map[string]any{"sessionId": "root-session"}
					case "Runtime.evaluate":
						result = map[string]any{"result": map[string]any{"type": "object", "className": "Window"}}
					case "Page.getFrameTree":
						if stopped {
							captureFrameReads++
							if scenario == "late-setup-failure" && captureFrameReads == 2 {
								attach()
							}
						}
						result = map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "loaderId": "loader", "url": "https://example.com/"}}}
					case "Page.navigate":
						navigated = true
						emit("Page.frameNavigated", "root-session", map[string]any{"frame": map[string]any{"id": "main", "loaderId": "loader", "url": "https://example.com/"}})
						emit("Page.lifecycleEvent", "root-session", map[string]any{"frameId": "main", "loaderId": "loader", "name": "load"})
						result = map[string]any{"frameId": "main", "loaderId": "loader"}
						if scenario == "setup-failure" || scenario == "detachment" {
							attach()
						}
						if scenario == "cancelled" {
							cancelFetch()
						}
					case "Page.stopLoading":
						stopped = true
					}
					response := map[string]any{"id": cmd.ID, "sessionId": cmd.SessionID, "result": result}
					if cmd.SessionID == "child-session" {
						if !navigated || cmd.Method != "Network.enable" {
							t.Errorf("unexpected child command %s", cmd.Method)
						}
						if scenario == "detachment" {
							emit("Target.detachedFromTarget", "root-session", map[string]any{"sessionId": "child-session"})
							continue
						}
						delete(response, "result")
						response["error"] = map[string]any{"code": -32000, "message": "original child setup error"}
					}
					encoded, _ := json.Marshal(response)
					if wsutil.WriteServerText(conn, encoded) != nil {
						return
					}
				}
			}))
			defer server.Close()
			host, port, _ := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
			portNum, _ := strconv.Atoi(port)
			client := &childHarvestRecorder{recorderReply: recorderReply{reply: &recorderV1.FinishHarvestReply{Reusable: true, Main: &logV1.CrawlLog{WarcId: "main-recording", StatusCode: 200}}}}
			client.lostBegin = scenario == "lost-begin"
			sess := New(1, WithConfigAdapter(childHarvestConfig{}), WithBrowserHost(host), WithBrowserPort(portNum), WithRecorder(client), WithAdmissionToken("fixture"))
			logs := captureLogs(sess, slog.LevelDebug)
			if scenario == "screenshot-failure" {
				sess.screenShotWriter = &testutil.ScreenshotWriterMock{WriteFunc: func([]byte, screenshotwriter.Metadata) error {
					return errors.New("injected screenshot persistence failure")
				}}
			}
			if scenario == "finalization-failure" {
				client.err = errors.New("injected recorder finalization failure")
				client.reply = nil
			}
			result, err := sess.Fetch(fetchCtx, &frontierV1.PageHarvestSpec{QueuedUri: &frontierV1.QueuedUri{Uri: "https://example.com/"}, CrawlConfig: &configV1.ConfigObject{Spec: &configV1.ConfigObject_CrawlConfig{CrawlConfig: &configV1.CrawlConfig{BrowserConfigRef: &configV1.ConfigRef{Id: "browser"}, PolitenessRef: &configV1.ConfigRef{Id: "politeness"}, Extra: &configV1.ExtraConfig{CreateScreenshot: scenario == "screenshot-failure"}}}}})
			if client.begins != 1 || client.request == nil {
				t.Fatal("recorder was not finalized")
			}
			switch scenario {
			case "lost-begin":
				if err == nil || client.recoveries != 1 || sess.NextToken() != "recovered-token" || client.request.PageUri != nil {
					t.Fatal("ambiguous Begin failed recovery", err)
				}
			case "detachment":
				if err != nil || result == nil || client.request.PageUri == nil {
					t.Fatalf("detachment failed page: %v", err)
				}
			case "cancelled":
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			case "screenshot-failure", "finalization-failure":
				if err == nil || !strings.Contains(err.Error(), "injected") {
					t.Fatal(err)
				}
			default:
				for _, detail := range []string{"child-target", "child-session", "iframe", "Network.enable", "original child setup error"} {
					if err == nil || !strings.Contains(err.Error(), detail) {
						t.Fatalf("missing %s: %v", detail, err)
					}
				}
				if client.request.PageUri != nil {
					t.Fatal("setup failure accepted artifacts")
				}
			}
			if sess.Reusable() != (scenario != "finalization-failure") {
				t.Fatal(fmt.Sprint("incorrect final slot state: ", err))
			}
			summaries := logs.matching(t, "Harvest finished")
			if len(summaries) != 1 {
				t.Fatal(summaries)
			}
			summary := summaries[0]
			if summary["harvestId"] == "" || summary["slot"] != float64(1) || summary["pageUrl"] != "https://example.com/" || summary["component"] != "browser-controller" || summary["durationMs"] == nil || summary["duration"] != nil {
				t.Fatal(summary)
			}
			expectedOutcome := "failure"
			if scenario == "detachment" {
				expectedOutcome = "success"
			}
			if scenario == "cancelled" {
				expectedOutcome = "canceled"
			}
			if summary["outcome"] != expectedOutcome {
				t.Fatal(summary)
			}
			if scenario == "screenshot-failure" && (summary["phase"] != "screenshot_persistence" || summary["screenshot"] != "failed") {
				t.Fatal(summary)
			}
			if scenario == "finalization-failure" && summary["phase"] != "recorder_finalization" {
				t.Fatal(summary)
			}
			if strings.Contains(scenario, "setup-failure") && summary["phase"] != "child_setup" {
				t.Fatal(summary)
			}
		})
	}
}

func TestChildSetupPreventsReadinessAndCaptureDrainIsBounded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sess, root, wire, _ := childHarness(t)
		sess.onTargetAttached(root, attached("child", "iframe"), 1)
		<-wire.commands
		ctx, cancel := context.WithTimeout(t.Context(), 1500*time.Millisecond)
		defer cancel()
		// Without pending setup this would report quiet after one second.
		if err := sess.waitForBrowserReadiness(ctx, 10*time.Millisecond); !errors.Is(err, errCompletionIdleTimeout) {
			t.Fatalf("readiness ignored pending child setup: %v", err)
		}
		if err := sess.drainChildren(ctx, true); !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "drain child setup") {
			t.Fatalf("unbounded capture drain: %v", err)
		}
	})
}
