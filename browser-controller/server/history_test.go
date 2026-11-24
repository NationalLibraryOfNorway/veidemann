//go:build integration

package server

import (
	"bytes"
	"fmt"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/screenshotwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/testutil"
)

func TestHistoryDocumentArtifacts(t *testing.T) {
	for _, scenario := range []struct {
		name, script, current string
		redirect, failed      bool
	}{
		{name: "push-path", script: `history.pushState({}, '', '/new-route')`, current: "/new-route"},
		{name: "push-query", script: `history.pushState({}, '', '?query=push')`, current: "/index.html?query=push"},
		{name: "replace-path", script: `history.replaceState({}, '', '/new-route')`, current: "/new-route"},
		{name: "replace-query", script: `history.replaceState({}, '', '?query=replace')`, current: "/index.html?query=replace"},
		{name: "redirect-history", redirect: true, script: `history.pushState({}, '', '/new-route?q=1')`, current: "/new-route?q=1"},
		{name: "reload-history", script: `if (!sessionStorage.reloaded) { sessionStorage.reloaded = 'yes'; location.reload(); } else { history.replaceState({}, '', '/after-reload'); }`, current: "/after-reload"},
		{name: "fragment", script: `location.hash = 'section'`, current: "/index.html#section"},
		{name: "later-failed-navigation", script: `history.pushState({}, '', '/new-route'); location.href = '/failed'`, failed: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			origin := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/failed" {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err == nil {
						conn.Close()
					}
					return
				}
				if scenario.redirect && r.URL.Path == "/index.html" {
					http.Redirect(w, r, "/final", http.StatusFound)
					return
				}
				w.Header().Set("Content-Type", "text/html")
				w.Header().Set("Cache-Control", "no-store")
				fmt.Fprintf(w, `<html><script>%s;</script><a href="/link">Link</a></html>`, scenario.script)
			}))
			_ = origin.Listener.Close()
			var err error
			origin.Listener, err = net.Listen("tcp", "0.0.0.0:0")
			if err != nil {
				t.Fatal(err)
			}
			origin.Start()
			defer origin.Close()
			_, port, _ := net.SplitHostPort(origin.Listener.Addr().String())
			base := fmt.Sprintf("http://%s:%s", localhost, port)
			requested := base + "/index.html"
			committed := requested
			if scenario.redirect {
				committed = base + "/final"
			}
			var screenshot []byte
			var metadata screenshotwriter.Metadata
			writes := 0
			options := append([]session.Option{}, sessionOptions...)
			options = append(options, session.WithScreenshotWriter(&testutil.ScreenshotWriterMock{WriteFunc: func(data []byte, meta screenshotwriter.Metadata) error {
				writes++
				screenshot = append([]byte(nil), data...)
				metadata = meta
				return nil
			}}))
			conf := &configV1.ConfigObject{Spec: &configV1.ConfigObject_CrawlConfig{CrawlConfig: &configV1.CrawlConfig{
				BrowserConfigRef: &configV1.ConfigRef{Id: "browserConfig1"},
				PolitenessRef:    &configV1.ConfigRef{Id: "politenessConfig1"},
				CollectionRef:    &configV1.ConfigRef{Id: "collectionConfig1"},
				Extra:            &configV1.ExtraConfig{CreateScreenshot: true},
			}}}
			sess := session.New(1, options...)
			result, err := sess.Fetch(t.Context(), &frontierV1.PageHarvestSpec{QueuedUri: &frontierV1.QueuedUri{Uri: requested, DiscoveryPath: "L", ExecutionId: "eid", JobExecutionId: "jid"}, CrawlConfig: conf})
			if scenario.failed {
				if err == nil || writes != 0 || recordedPages.latest(requested) != nil {
					t.Fatalf("failed navigation reused old success: result=%v err=%v writes=%d", result, err, writes)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !hasOutlink(result.Outlinks, base+"/link") || result.UriCount == 0 || result.BytesDownloaded <= 0 {
				t.Fatalf("missing harvest artifacts: %v", result)
			}
			for _, link := range result.Outlinks {
				if link.Referrer != base+scenario.current {
					t.Fatalf("referrer=%q want %q", link.Referrer, base+scenario.current)
				}
			}
			page := recordedPages.latest(requested)
			if page == nil {
				t.Fatal("missing page log")
			}
			if writes != 1 || metadata.CrawlLog.GetWarcId() == "" || metadata.CrawlLog.GetWarcId() != page.WarcId || metadata.CrawlLog.GetRequestedUri() != committed {
				t.Fatalf("screenshot attached to wrong document: writes=%d metadata=%v page=%v", writes, metadata, page)
			}
			if _, err := png.DecodeConfig(bytes.NewReader(screenshot)); err != nil {
				t.Fatal("invalid screenshot", err)
			}
		})
	}
}
