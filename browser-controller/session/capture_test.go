package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

// Drive capture over the actual CDP transport, with deterministic document
// changes between extraction and validation. No timers or live-site behavior.
func TestCaptureBundleConsistencyAndRequiredOperations(t *testing.T) {
	for _, scenario := range []string{"stable", "changes-once", "always-changes", "screenshot-error", "cookies-error", "missing-commit", "mismatched-loader", "history-changes-once", "missing-loader"} {
		t.Run(scenario, func(t *testing.T) {
			sess := newDefaultSession()
			sess.pageLoad.start("main")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, _, _, err := ws.UpgradeHTTP(r, w)
				if err != nil {
					t.Error(err)
					return
				}
				defer conn.Close()
				generation := 1
				committedGeneration := 0
				for {
					data, _, err := wsutil.ReadClientData(conn)
					if err != nil {
						return
					}
					var cmd struct {
						ID        int64  `json:"id"`
						SessionID string `json:"sessionId"`
						Method    string `json:"method"`
						Params    struct {
							Expression string `json:"expression"`
						} `json:"params"`
					}
					if err := json.Unmarshal(data, &cmd); err != nil {
						t.Error(err)
						return
					}
					result := any(map[string]any{})
					switch cmd.Method {
					case "Target.createTarget":
						result = map[string]any{"targetId": "root"}
					case "Target.attachToTarget":
						result = map[string]any{"sessionId": "session"}
					case "Page.getFrameTree":
						loader := cdp.LoaderID(fmt.Sprint(generation))
						if scenario == "history-changes-once" {
							loader = "1"
						}

						if scenario != "missing-commit" && committedGeneration != generation && !(scenario == "history-changes-once" && committedGeneration != 0) {
							committedGeneration = generation
							sess.pageLoad.navigated(&cdp.Frame{ID: "main", LoaderID: cdp.LoaderID(fmt.Sprint(generation)), URL: fmt.Sprintf("https://example.com/loaded-%d", generation)})
						}
						if scenario == "missing-loader" {
							loader = ""
						}
						if scenario == "mismatched-loader" {
							loader = "wrong"
						}
						result = map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "loaderId": string(loader), "url": fmt.Sprintf("https://example.com/%d", generation)}}}
					case "Page.captureScreenshot":
						result = map[string]any{"data": base64.StdEncoding.EncodeToString([]byte(fmt.Sprint(generation)))}
					case "Network.getCookies":
						result = map[string]any{"cookies": []any{map[string]any{"name": "generation", "value": fmt.Sprint(generation), "domain": "example.com", "path": "/"}}}
					case "Runtime.evaluate":
						result = map[string]any{"result": map[string]any{"type": "object", "className": "Window"}}
						if cmd.Params.Expression == "extract-test-links" {
							result = map[string]any{"result": map[string]any{"type": "object", "value": []string{fmt.Sprintf("https://example.com/link-%d", generation)}}}
							if scenario == "always-changes" || ((scenario == "changes-once" || scenario == "history-changes-once") && generation == 1) {
								generation++
							}
						}
					}
					response := map[string]any{"id": cmd.ID, "sessionId": cmd.SessionID, "result": result}
					if (scenario == "screenshot-error" && cmd.Method == "Page.captureScreenshot") || (scenario == "cookies-error" && cmd.Method == "Network.getCookies") {
						delete(response, "result")
						response["error"] = map[string]any{"code": -32000, "message": "required operation failed"}
					}
					encoded, err := json.Marshal(response)
					if err != nil {
						t.Error(err)
						return
					}
					if err := wsutil.WriteServerText(conn, encoded); err != nil {
						return
					}
				}
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			alloc, closeAlloc := newRemoteAllocator(ctx, "ws"+strings.TrimPrefix(server.URL, "http"))
			defer closeAlloc()
			browser, closeBrowser := chromedp.NewContext(alloc)
			defer closeBrowser()
			sess.mainFrameID = "main"
			sess.CrawlConfig = &configV1.CrawlConfig{Extra: &configV1.ExtraConfig{CreateScreenshot: true}}
			sess.browserConfig = &configV1.BrowserConfig{}
			sess.scripts = newSessionScripts()
			sess.scripts.scripts[configV1.BrowserScript_EXTRACT_OUTLINKS] = []*configV1.ConfigObject{{Spec: &configV1.ConfigObject_BrowserScript{BrowserScript: &configV1.BrowserScript{Script: "extract-test-links"}}}}
			bundle, err := sess.captureBundle(browser)
			if scenario == "missing-loader" || scenario == "missing-commit" || scenario == "mismatched-loader" || scenario == "always-changes" || strings.HasSuffix(scenario, "-error") {
				if err == nil {
					t.Fatalf("accepted failed capture: %+v", bundle)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want := "1"
			if scenario == "changes-once" || scenario == "history-changes-once" {
				want = "2"
			}
			committed := want
			if scenario == "history-changes-once" {
				committed = "1"
			}
			if bundle.committedURL != "https://example.com/loaded-"+committed || bundle.uri != "https://example.com/"+want || string(bundle.screenshot) != want || len(bundle.cookies) != 1 || bundle.cookies[0].Value != want || len(bundle.outlinks) != 1 || bundle.outlinks[0] != "https://example.com/link-"+want {
				t.Fatalf("mixed artifact generations: %+v", bundle)
			}
		})
	}
}
