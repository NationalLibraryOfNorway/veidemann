package session

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

// This backend exercises a real remote WebSocket connection without Docker.
func TestRemoteBrowserEventOrderAndShutdown(t *testing.T) {
	testRemoteBrowserEventOrderAndShutdown(t, false)
}

func TestRemoteBrowserEventOrderBeforeDetach(t *testing.T) {
	testRemoteBrowserEventOrderAndShutdown(t, true)
}

func testRemoteBrowserEventOrderAndShutdown(t *testing.T, detach bool) {
	t.Helper()
	connected := make(chan string, 1)
	closed := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			t.Errorf("upgrade browser connection: %v", err)
			return
		}
		defer func() {
			if err := conn.Close(); err != nil {
				t.Errorf("close browser connection: %v", err)
			}
		}()
		defer func() { closed <- struct{}{} }()
		connected <- r.URL.RequestURI()
		for {
			data, _, err := wsutil.ReadClientData(conn)
			if err != nil {
				return
			}
			var cmd struct {
				ID        int64           `json:"id"`
				SessionID string          `json:"sessionId"`
				Method    string          `json:"method"`
				Params    json.RawMessage `json:"params"`
			}
			if err := json.Unmarshal(data, &cmd); err != nil {
				t.Errorf("decode browser command: %v", err)
				return
			}
			detachAfterReply := false
			result := json.RawMessage(`{}`)
			switch cmd.Method {
			case "Browser.getVersion":
				result = json.RawMessage(`{"product":"Chrome/test","userAgent":"HeadlessChrome/test"}`)
			case "Target.createTarget":
				result = json.RawMessage(`{"targetId":"root"}`)
			case "Target.attachToTarget":
				result = json.RawMessage(`{"sessionId":"root-session"}`)
			case "Runtime.evaluate":
				result = json.RawMessage(`{"result":{"type":"object","className":"Window"}}`)
				var params runtime.EvaluateParams
				if err := json.Unmarshal(cmd.Params, &params); err != nil {
					t.Errorf("decode evaluate parameters: %v", err)
					return
				}
				if params.Expression == "emit-events" {
					detachAfterReply = detach
					for _, event := range []string{
						`{"method":"Network.loadingFinished","sessionId":"other-session","params":{"requestId":"other"}}`,
						`{"method":"Network.loadingFinished","params":{"requestId":"browser"}}`,
						`{"method":"Page.frameStartedLoading","sessionId":"root-session","params":{"frameId":"frame"}}`,
						`{"method":"Network.requestWillBeSent","sessionId":"root-session","params":{"requestId":"req","request":{"url":"https://example.com/","method":"GET"},"initiator":{"type":"parser"},"type":"Document","frameId":"frame"}}`,
						`{"method":"Network.loadingFinished","sessionId":"root-session","params":{"requestId":"req"}}`,
						`{"method":"Page.frameStoppedLoading","sessionId":"root-session","params":{"frameId":"frame"}}`,
					} {
						if err := wsutil.WriteServerText(conn, []byte(event)); err != nil {
							return
						}
					}
				}
			}
			response, err := json.Marshal(struct {
				ID        int64           `json:"id"`
				SessionID string          `json:"sessionId,omitempty"`
				Result    json.RawMessage `json:"result"`
			}{cmd.ID, cmd.SessionID, result})
			if err != nil {
				t.Errorf("encode browser response: %v", err)
				return
			}
			if err := wsutil.WriteServerText(conn, response); err != nil {
				return
			}
			if detachAfterReply {
				if err := wsutil.WriteServerText(conn, []byte(`{"method":"Target.detachedFromTarget","params":{"sessionId":"root-session","targetId":"root"}}`)); err != nil {
					return
				}
			}
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	endpoint := "ws" + strings.TrimPrefix(server.URL, "http") + "/?trackingId=7&launch=browserless"
	allocatorCtx, cancelAllocator := newRemoteAllocator(ctx, endpoint)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx)
	defer cancelBrowser()
	version, err := chromedp.CallBrowser(browserCtx, browser.GetVersion, cdp.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if version.Product != "Chrome/test" {
		t.Fatalf("browser version = %q", version.Product)
	}
	if got := <-connected; got != "/?trackingId=7&launch=browserless" {
		t.Fatalf("browser endpoint changed: %q", got)
	}

	sess := &Session{networkTracker: newNetworkActivityTracker(), frameLoads: newFrameLoadTracker()}
	sess.startAcceptingRequests()
	if err := sess.initListeners(browserCtx); err != nil {
		t.Fatal(err)
	}
	if sess.rootTargetID != "root" {
		t.Fatalf("root target = %q", sess.rootTargetID)
	}
	var delivered, canceled atomic.Int32
	if err := listenTarget(browserCtx, func(ev any) {
		if _, ok := ev.(*network.EventLoadingFinished); ok {
			delivered.Add(1)
		}
	}); err != nil {
		t.Fatal(err)
	}
	listenerCtx, cancelListener := context.WithCancel(browserCtx)
	defer cancelListener()
	if err := listenTarget(listenerCtx, func(any) { canceled.Add(1) }); err != nil {
		t.Fatal(err)
	}
	cancelListener()

	detached, err := chromedp.Detached(browserCtx)
	if err != nil {
		t.Fatal(err)
	}
	res, err := chromedp.Call(browserCtx, runtime.Evaluate, runtime.EvaluateParams{Expression: "emit-events"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result == nil || res.Result.ClassName != "Window" {
		t.Fatalf("command reply lost: %+v", res)
	}
	if detach {
		select {
		case <-detached:
		case <-ctx.Done():
			t.Fatal("target did not detach")
		}
	}
	if got := delivered.Load(); got != 1 {
		t.Fatalf("delivered %d completion events, want only the root session event", got)
	}
	if got := canceled.Load(); got != 0 {
		t.Fatalf("canceled listener received %d events", got)
	}
	request, found := sess.RequestSnapshot("req")
	if !found || !request.GotComplete {
		t.Fatalf("request events were reordered: request=%+v, found=%v", request, found)
	}
	if active := sess.frameLoads.Snapshot(); len(active) != 0 {
		t.Fatalf("frame events were reordered: active=%v", active)
	}
	if err := chromedp.Cancel(browserCtx); err != nil {
		t.Fatal(err)
	}
	a := chromedp.FromContext(allocatorCtx).Allocator.(*remoteAllocator)
	a.Wait()
	select {
	case <-closed:
	case <-ctx.Done():
		t.Fatal("remote browser connection was not closed")
	}
}

func TestListenTargetPropagatesRegistrationError(t *testing.T) {
	if err := listenTarget(t.Context(), func(any) {}); !errors.Is(err, chromedp.ErrInvalidContext) {
		t.Fatalf("listenTarget() error = %v, want ErrInvalidContext", err)
	}
}
