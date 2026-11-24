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

func TestFailedSubresourcePreservesArtifacts(t *testing.T) {
	mu, diagnostics := captureFixtureDiagnostics(t)
	dead, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	_, deadPort, _ := net.SplitHostPort(dead.Addr().String())
	_ = dead.Close()
	imageURI := fmt.Sprintf("http://%s:%s/dead.png", localhost, deadPort)
	origin := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><img src="%s"><a href="/link">Link</a></html>`, imageURI)
	}))
	_ = origin.Listener.Close()
	origin.Listener, err = net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	origin.Start()
	defer origin.Close()
	_, port, _ := net.SplitHostPort(origin.Listener.Addr().String())
	base := fmt.Sprintf("http://%s:%s", localhost, port)
	requested := base + "/index.html"
	var screenshot []byte
	var metadata screenshotwriter.Metadata
	writes := 0
	options := append([]session.Option{}, sessionOptions...)
	options = append(options, session.WithScreenshotWriter(&testutil.ScreenshotWriterMock{
		WriteFunc: func(data []byte, meta screenshotwriter.Metadata) error {
			writes++
			screenshot = append([]byte(nil), data...)
			metadata = meta
			return nil
		},
	}))
	conf := &configV1.ConfigObject{Spec: &configV1.ConfigObject_CrawlConfig{CrawlConfig: &configV1.CrawlConfig{
		BrowserConfigRef: &configV1.ConfigRef{Id: "browserConfig1"},
		PolitenessRef:    &configV1.ConfigRef{Id: "politenessConfig1"},
		CollectionRef:    &configV1.ConfigRef{Id: "collectionConfig1"},
		Extra:            &configV1.ExtraConfig{CreateScreenshot: true},
	}}}
	sess := session.New(1, options...)
	result, err := sess.Fetch(t.Context(), &frontierV1.PageHarvestSpec{
		QueuedUri:   &frontierV1.QueuedUri{Uri: requested, DiscoveryPath: "L", ExecutionId: "eid", JobExecutionId: "jid"},
		CrawlConfig: conf,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hasOutlink(result.Outlinks, base+"/link") || result.UriCount == 0 || result.BytesDownloaded <= 0 || !sess.Reusable() {
		t.Fatalf("harvest result lost: %v", result)
	}
	page := recordedPages.latest(requested)
	if page == nil {
		t.Fatal("missing page log")
	}
	if writes != 1 || metadata.CrawlLog.GetWarcId() != page.WarcId || metadata.CrawlLog.GetRequestedUri() != requested {
		t.Fatalf("screenshot not attached to main: writes=%d metadata=%v", writes, metadata)
	}
	if _, err := png.DecodeConfig(bytes.NewReader(screenshot)); err != nil {
		t.Fatal("invalid screenshot", err)
	}
	var recorded int32
	failed := 0
	for _, resource := range page.Resource {
		if resource.WarcId != "" {
			recorded++
		}
		if resource.Uri == imageURI {
			failed++
			if resource.GetError().GetCode() != -2 || resource.WarcId != "" {
				t.Fatal(resource)
			}
		}
	}
	if failed != 1 || result.UriCount != recorded {
		t.Fatalf("failed images=%d count=%d resources=%v", failed, result.UriCount, page.Resource)
	}
	mu.Lock()
	defer mu.Unlock()
	summary, browserFailure, recordingFailure := false, false, false
	for _, row := range *diagnostics {
		if row["msg"] == "Harvest finished" && row["outcome"] == "success" && (row["browserRequestsFailed"] != "0" || row["browserHttpErrors"] != "0") {
			summary = true
		}
		if (row["msg"] == "Browser request finished" && (row["outcome"] == "failed" || row["httpStatus"] == "503")) && row["url"] == imageURI && row["cdpSessionId"] != "" {
			browserFailure = true
		}
		if row["msg"] == "Resource recording finished" && row["url"] == imageURI && row["outcome"] == "failed" && row["warcId"] == "" {
			recordingFailure = true
		}
	}
	if !summary || !browserFailure || !recordingFailure {
		t.Fatalf("missing failure diagnostics: summary=%t browser=%t recorder=%t", summary, browserFailure, recordingFailure)
	}
}
