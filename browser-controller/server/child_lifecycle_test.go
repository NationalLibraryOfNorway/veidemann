//go:build integration

package server

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/screenshotwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/testutil"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	recorderLogger "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
)

type childDiagnostics struct {
	slog.Handler
	mu      *sync.Mutex
	records *[]map[string]string
	attrs   []slog.Attr
}

func (h *childDiagnostics) Enabled(_ context.Context, level slog.Level) bool {
	return level >= session.LevelTrace
}
func (h *childDiagnostics) WithAttrs(attrs []slog.Attr) slog.Handler {
	copy := *h
	copy.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &copy
}
func (h *childDiagnostics) Handle(_ context.Context, r slog.Record) error {
	values := map[string]string{"msg": r.Message, "level": r.Level.String()}
	for _, a := range h.attrs {
		values[a.Key] = a.Value.String()
	}
	r.Attrs(func(a slog.Attr) bool { values[a.Key] = a.Value.String(); return true })
	h.mu.Lock()
	*h.records = append(*h.records, values)
	h.mu.Unlock()
	return h.Handler.WithAttrs(h.attrs).Handle(context.Background(), r)
}

func captureFixtureDiagnostics(t *testing.T) (*sync.Mutex, *[]map[string]string) {
	t.Helper()
	var records []map[string]string
	mu := &sync.Mutex{}
	previous, previousRecorder := slog.Default(), recorderLogger.Log
	log := slog.New(&childDiagnostics{Handler: previous.Handler(), mu: mu, records: &records})
	slog.SetDefault(log)
	recorderLogger.Log = recorderLogger.New(log)
	t.Cleanup(func() { slog.SetDefault(previous); recorderLogger.Log = previousRecorder })
	return mu, &records
}

// The two sites use different hosts, not merely ports, so site isolation must
// create an OOPIF. A nested frame and a worker must run after their sessions are
// configured, then publish a link that the root extraction must see.
func TestChildTargetLifecycleArtifacts(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(fmt.Sprintf("remove-%t", remove), func(t *testing.T) {
			var mainBase, childBase string
			origin := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get(constants.HeaderAttemptId) != "" || r.Header.Get(constants.HeaderMainDocument) != "" {
					t.Error("internal header reached child origin")
				}
				w.Header().Set("Cache-Control", "no-store")
				switch r.URL.Path {
				case "/main":
					w.Header().Set("Content-Type", "text/html")
					fmt.Fprintf(w, `<html><body>Root<iframe id="child" src="%s/child"></iframe><script>
    window.addEventListener('message',event=>{if(event.data==='nested-and-worker-loaded'){
      const a=document.createElement('a');a.href='/complete';a.textContent='Complete';document.body.append(a);
      if(%t)document.getElementById('child').remove();
    }});</script></body></html>`, childBase, remove)
				case "/child":
					w.Header().Set("Content-Type", "text/html")
					fmt.Fprintf(w, `<html><body>Child<img src="/child.svg"><iframe src="%s/nested"></iframe><script>
    let nested=false,worker=false;
    const done=()=>{if(nested&&worker)parent.postMessage('nested-and-worker-loaded','*')};
    window.addEventListener('message',event=>{if(event.data==='nested-loaded'){nested=true;done()}});
    const w=new Worker('/worker.js');w.onmessage=()=>{worker=true;w.terminate();done()};
    </script></body></html>`, mainBase)
				case "/nested":
					w.Header().Set("Content-Type", "text/html")
					fmt.Fprint(w, `<html>Nested<img src="/nested.svg" onload="parent.postMessage('nested-loaded','*')"></html>`)
				case "/worker.js":
					w.Header().Set("Content-Type", "application/javascript")
					fmt.Fprint(w, `fetch('/worker-data').then(r=>r.text()).then(()=>postMessage('loaded'))`)
				case "/worker-data":
					fmt.Fprint(w, "worker content")
				case "/child.svg", "/nested.svg":
					w.Header().Set("Content-Type", "image/svg+xml")
					fmt.Fprint(w, `<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"><rect width="8" height="8" fill="blue"/></svg>`)
				default:
					w.WriteHeader(http.StatusNoContent)
				}
			}))
			origin.Listener.Close()
			var err error
			origin.Listener, err = net.Listen("tcp", "0.0.0.0:0")
			if err != nil {
				t.Fatal(err)
			}
			_, port, _ := net.SplitHostPort(origin.Listener.Addr().String())
			mainBase = "http://" + net.JoinHostPort(localhost, port)
			childBase = "http://" + net.JoinHostPort("localhost", port)
			origin.Start()
			defer origin.Close()
			mu, records := captureFixtureDiagnostics(t)
			var screenshot []byte
			var metadata screenshotwriter.Metadata
			opts := append([]session.Option{}, sessionOptions...)
			opts = append(opts, session.WithScreenshotWriter(&testutil.ScreenshotWriterMock{WriteFunc: func(data []byte, m screenshotwriter.Metadata) error {
				screenshot = append([]byte(nil), data...)
				metadata = m
				return nil
			}}))
			conf := &configV1.ConfigObject{Spec: &configV1.ConfigObject_CrawlConfig{CrawlConfig: &configV1.CrawlConfig{
				BrowserConfigRef: &configV1.ConfigRef{Id: "browserConfig1"}, PolitenessRef: &configV1.ConfigRef{Id: "politenessConfig1"}, CollectionRef: &configV1.ConfigRef{Id: "collectionConfig1"}, Extra: &configV1.ExtraConfig{CreateScreenshot: true},
			}}}
			requested := mainBase + "/main"
			result, err := session.New(1, opts...).Fetch(t.Context(), &frontierV1.PageHarvestSpec{QueuedUri: &frontierV1.QueuedUri{Uri: requested, DiscoveryPath: "L", ExecutionId: "eid", JobExecutionId: "jid"}, CrawlConfig: conf})
			if err != nil {
				t.Fatalf("%v; child sessions: %v", err, records)
			}
			if !hasOutlink(result.Outlinks, mainBase+"/complete") {
				t.Fatalf("nested frame/worker did not finish: %v; sessions: %v", result, records)
			}
			page := recordedPages.latest(requested)
			if page == nil || page.WarcId == "" || metadata.CrawlLog.GetRequestedUri() != requested || metadata.CrawlLog.GetWarcId() != page.WarcId {
				t.Fatalf("wrong main artifacts: %v, %v", page, metadata)
			}
			if _, err := png.DecodeConfig(bytes.NewReader(screenshot)); err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{childBase + "/child", childBase + "/child.svg", mainBase + "/nested", mainBase + "/nested.svg", childBase + "/worker.js", childBase + "/worker-data"} {
				found := false
				for _, r := range page.Resource {
					if r.Uri == want && r.StatusCode == 200 && r.WarcId != "" {
						found = true
					}
				}
				if !found {
					t.Fatalf("missing child recording %s: %v", want, page.Resource)
				}
			}
			mu.Lock()
			defer mu.Unlock()
			summaries, recorderSummaries, configured, terminals := 0, 0, 0, 0
			for _, row := range *records {
				if row["level"] == "DEBUG" && (row["component"] == "recorderproxy" || row["component"] == "browser-controller") && row["msg"] != "Resource recording finished" && row["msg"] != "Browser request finished" {
					t.Fatalf("intermediate event at DEBUG: %v", row)
				}
				switch row["msg"] {
				case "Harvest finished":
					summaries++
					if row["outcome"] != "success" || row["screenshot"] != "stored" || row["harvestId"] == "" || row["slot"] != "1" || row["durationMs"] == "" {
						t.Fatal(row)
					}
				case "Recording finished":
					recorderSummaries++
					if row["pageLog"] != "stored" || row["outcome"] != "success" {
						t.Fatal(row)
					}
				case "Child session configured":
					configured++
					if row["paused"] != "false" || row["resumed"] != "true" {
						t.Fatal(row)
					}
				case "Resource recording finished":
					terminals++
				}
			}
			if summaries != 1 || recorderSummaries != 1 || configured < 2 || terminals < 7 {
				t.Fatalf("incomplete harvest diagnostics: summaries=%d recorder=%d configured=%d terminal=%d", summaries, recorderSummaries, configured, terminals)
			}
			iframeSessions := map[string]string{}
			worker := false
			for _, r := range *records {
				if r["msg"] != "Child session attached" {
					continue
				}
				if r["waitingForDebugger"] != "true" || r["cdpSessionId"] == "" || r["parentCdpSessionId"] == "" {
					t.Fatalf("incomplete attachment: %v", r)
				}
				if r["targetType"] == "iframe" {
					iframeSessions[r["cdpSessionId"]] = r["parentCdpSessionId"]
				}
				if r["targetType"] == "worker" {
					worker = true
				}
			}
			if len(iframeSessions) < 2 || !worker {
				t.Fatalf("site isolation/worker attachment missing: %v", records)
			}
			nested := false
			for _, parent := range iframeSessions {
				if _, ok := iframeSessions[parent]; ok {
					nested = true
				}
			}
			if !nested {
				t.Fatalf("missing recursive attachment: %v", records)
			}
		})
	}
}
