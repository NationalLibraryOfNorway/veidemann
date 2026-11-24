//go:build integration

package server

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/screenshotwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/testutil"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
)

func TestSameURLIframeArtifacts(t *testing.T) {
	for _, scenario := range []string{"http", "https", "cross-site-redirect"} {
		t.Run(scenario, func(t *testing.T) {
			const parentHTML = `<html><iframe src="/index.html"></iframe><a href="/link">Link</a></html>`
			const childHTML = `<html>Child</html>`
			var committed string
			origin := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get(constants.HeaderMainDocument) != "" || r.Header.Get(constants.HeaderAttemptId) != "" {
					t.Error("internal header reached origin")
				}
				if r.URL.Path == "/start" {
					http.Redirect(w, r, committed, http.StatusFound)
					return
				}
				if r.URL.Path != "/index.html" {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				w.Header().Set("Content-Type", "text/html")
				w.Header().Set("Cache-Control", "no-store")
				if strings.Contains(r.Header.Get("Cookie"), "child=yes") {
					fmt.Fprint(w, childHTML)
				} else {
					w.Header().Set("Set-Cookie", "child=yes; Path=/")
					fmt.Fprint(w, parentHTML)
				}
			}))
			_ = origin.Listener.Close()
			var err error
			origin.Listener, err = net.Listen("tcp", "0.0.0.0:0")
			if err != nil {
				t.Fatal(err)
			}
			scheme := "http"
			if scenario == "https" {
				scheme = "https"
				origin.StartTLS()
			} else {
				origin.Start()
			}
			defer origin.Close()
			_, port, _ := net.SplitHostPort(origin.Listener.Addr().String())
			base := fmt.Sprintf("%s://%s:%s", scheme, localhost, port)
			requested := base + "/index.html"
			committed = requested
			if scenario == "cross-site-redirect" {
				requested = base + "/start"
				committed = fmt.Sprintf("http://localhost:%s/index.html", port)
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
				BrowserConfigRef: &configV1.ConfigRef{Id: "browserConfig1"}, PolitenessRef: &configV1.ConfigRef{Id: "politenessConfig1"}, CollectionRef: &configV1.ConfigRef{Id: "collectionConfig1"}, Extra: &configV1.ExtraConfig{CreateScreenshot: true},
			}}}
			sess := session.New(1, options...)
			result, err := sess.Fetch(t.Context(), &frontierV1.PageHarvestSpec{QueuedUri: &frontierV1.QueuedUri{Uri: requested, DiscoveryPath: "L", ExecutionId: "eid", JobExecutionId: "jid"}, CrawlConfig: conf})
			if err != nil {
				t.Fatal(err)
			}
			page := recordedPages.latest(requested)
			parentWARC := fmt.Sprintf("warcid_1_%x", sha1.Sum([]byte(parentHTML)))
			childWARC := fmt.Sprintf("warcid_1_%x", sha1.Sum([]byte(childHTML)))
			if page == nil || page.WarcId != parentWARC {
				t.Fatalf("page selected child: %v", page)
			}
			parentCount, childCount := 0, 0
			for _, resource := range page.Resource {
				if resource.Uri == committed && resource.WarcId == parentWARC {
					parentCount++
				}
				if resource.Uri == committed && resource.WarcId == childWARC {
					childCount++
				}
			}
			if parentCount != 1 || childCount != 1 {
				t.Fatalf("missing distinct same-URL recordings: %v", page.Resource)
			}
			if writes != 1 || metadata.CrawlLog.GetWarcId() != parentWARC || metadata.CrawlLog.GetRequestedUri() != committed {
				t.Fatalf("wrong screenshot metadata: %v", metadata)
			}
			if _, err := png.DecodeConfig(bytes.NewReader(screenshot)); err != nil {
				t.Fatal(err)
			}
			if result.UriCount < 2 || result.BytesDownloaded <= 0 || !hasOutlink(result.Outlinks, strings.TrimSuffix(committed, "index.html")+"link") {
				t.Fatalf("missing artifacts: %v", result)
			}
		})
	}
}
