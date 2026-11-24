package recorderproxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
)

func discoveryRequest(method, uri, ref, dest string) *http.Request {
	req := httptest.NewRequest(method, uri, nil)
	req.Header.Set("Referer", ref)
	req.Header.Set("Sec-Fetch-Dest", dest)
	return req
}
func redirectResponse(status int, location string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Location": []string{location}}}
}

func TestDiscoveryAncestryRules(t *testing.T) {
	type step struct {
		uri, ref, dest string
		main           bool
		want, location string
	}
	for _, tc := range []struct {
		name  string
		steps []step
	}{
		{"fragment-normalization", []step{
			{uri: "https://EXAMPLE.com:443/index#one", main: true, want: "L"},
			{uri: "https://example.com/index#two", main: true, want: "L"},
			{uri: "https://example.com/image", ref: "https://example.com/index#three", want: "LE"},
		}},
		{"redirect-chain-and-embeds", []step{
			{uri: "https://example.com/start", main: true, want: "L", location: "middle"},
			{uri: "https://example.com/middle", main: true, want: "LR", location: "/final#section"},
			{uri: "https://example.com/final", main: true, want: "LRR"},
			{uri: "https://example.com/style", ref: "https://example.com/final", dest: "style", want: "LRRE", location: "/sheet"},
			{uri: "https://example.com/sheet", ref: "https://example.com/final", dest: "style", want: "LRRER"},
			{uri: "https://example.com/image", ref: "https://example.com/sheet", dest: "image", want: "LRRERE"},
		}},
		{"same-url-iframe-and-ambiguous-referrer", []step{
			{uri: "https://example.com/index", main: true, want: "L"},
			{uri: "https://example.com/index", ref: "https://example.com/index", want: "LE"},
			{uri: "https://example.com/image", ref: "https://example.com/index", want: "LE"},
		}},
		{"nested-embeds", []step{
			{uri: "https://example.com/index", main: true, want: "L"},
			{uri: "https://example.com/frame", ref: "https://example.com/index", want: "LE"},
			{uri: "https://example.com/style", ref: "https://example.com/frame", want: "LEE"},
			{uri: "https://example.com/image", ref: "https://example.com/style", want: "LEEE"},
		}},
		{"fallbacks-after-redirect", []step{
			{uri: "https://example.com/start", main: true, want: "L", location: "/final"},
			{uri: "https://example.com/final", main: true, want: "LR"},
			{uri: "https://example.com/missing", want: "LRE"},
			{uri: "https://example.com/malformed", ref: "://bad", want: "LRE"},
			{uri: "https://example.com/origin-only", ref: "https://example.com/", want: "LRE"},
			{uri: "https://example.com/changed-history", ref: "https://example.com/new-route?q=1", dest: "empty", want: "LRX"},
			{uri: "https://example.com/script-image", dest: "image", want: "LRE"},
		}},
		{"client-navigation-reload", []step{
			{uri: "https://example.com/start", main: true, want: "L", location: "/final"},
			{uri: "https://example.com/final", main: true, want: "LR"},
			{uri: "https://example.com/client", main: true, want: "LRX"},
			{uri: "https://example.com/client#fragment", main: true, want: "LRX"},
			{uri: "https://example.com/client?q=2", main: true, want: "LRXX"},
		}},
		{"before-main", []step{
			{uri: "https://example.com/early", dest: "empty", want: "LX"},
			{uri: "https://example.com/index", main: true, want: "L"},
		}},
		{"same-path-repeated-parents", []step{
			{uri: "https://example.com/index", main: true, want: "L"},
			{uri: "https://example.com/frame", ref: "https://example.com/index", want: "LE"},
			{uri: "https://example.com/frame", ref: "https://example.com/index", want: "LE"},
			{uri: "https://example.com/image", ref: "https://example.com/frame", want: "LEE"},
		}},
		{"same-url-redirect-once", []step{
			{uri: "https://example.com/index", main: true, want: "L", location: "/index"},
			{uri: "https://example.com/index", main: true, want: "LR"},
			{uri: "https://example.com/index", main: true, want: "LR"},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var d discoveryIndex
			for i, s := range tc.steps {
				tr := d.admit(uint64(i+1), discoveryRequest("GET", s.uri, s.ref, s.dest), s.uri, s.main, "L")
				if tr.path != s.want {
					t.Fatalf("step %d: %s path=%s want=%s", i, s.uri, tr.path, s.want)
				}
				if s.location != "" {
					d.response(tr, redirectResponse(302, s.location))
				}
			}
		})
	}
}

func TestDiscoveryRedirectCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name, method, nextMethod, dest, nextDest, location string
		status                                             int
		main, nextMain, match                              bool
	}{
		{"301-post", "POST", "GET", "", "", "/next", 301, true, true, true},
		{"302-post", "POST", "GET", "", "", "/next", 302, true, true, true},
		{"303-put", "PUT", "GET", "", "", "/next", 303, true, true, true},
		{"303-head", "HEAD", "HEAD", "", "", "/next", 303, true, true, true},
		{"307-post", "POST", "POST", "", "", "/next", 307, true, true, true},
		{"308-post", "POST", "POST", "", "", "/next", 308, true, true, true},
		{"wrong-method", "POST", "GET", "", "", "/next", 307, true, true, false},
		{"wrong-role", "GET", "GET", "", "", "/next", 302, true, false, false},
		{"wrong-dest", "GET", "GET", "image", "style", "/next", 302, false, false, false},
		{"missing-dest", "GET", "GET", "image", "", "/next", 302, false, false, true},
		{"not-redirect", "GET", "GET", "", "", "/next", 200, true, true, false},
		{"304", "GET", "GET", "", "", "/next", 304, true, true, false},
		{"empty-location", "GET", "GET", "", "", "", 302, true, true, false},
		{"invalid-location", "GET", "GET", "", "", "http://[bad", 302, true, true, false},
		{"non-network", "GET", "GET", "", "", "data:text/plain,hi", 302, true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var d discoveryIndex
			source := "https://example.com/start"
			parent := d.admit(1, discoveryRequest(tc.method, source, "", tc.dest), source, tc.main, "L")
			d.response(parent, redirectResponse(tc.status, tc.location))
			target := "https://example.com/next"
			next := d.admit(2, discoveryRequest(tc.nextMethod, target, "", tc.nextDest), target, tc.nextMain, "L")
			if (next.path == parent.path+"R") != tc.match {
				t.Fatalf("parent=%s next=%s match=%v", parent.path, next.path, tc.match)
			}
		})
	}
}

func TestDiscoveryAmbiguousRedirectsDiscarded(t *testing.T) {
	var d discoveryIndex
	for i, uri := range []string{"https://example.com/one", "https://example.com/two"} {
		parent := d.admit(uint64(i+1), discoveryRequest("GET", uri, "", "image"), uri, false, "L")
		d.response(parent, redirectResponse(302, "/target"))
	}
	for i := 0; i < 2; i++ {
		uri := "https://example.com/target"
		tr := d.admit(uint64(i+3), discoveryRequest("GET", uri, "", "image"), uri, false, "L")
		if tr.path != "LE" {
			t.Fatalf("ambiguous candidate reused: %s", tr.path)
		}
	}
}

func TestDiscoveryIndependentOfCompletionAndClearedAtFinish(t *testing.T) {
	c, p := testControl()
	spec := beginSpec("discovery-attempt-0001")
	spec.DiscoveryPath = "L"
	_, err := c.BeginHarvest(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	a := p.attempt
	add := func(uri, ref string, main bool) *rpcontext.RecordContext {
		req := discoveryRequest("GET", uri, ref, "")
		req = req.WithContext(rpcontext.RecordProxyDataAware(t.Context()))
		if main {
			req.Header.Set(constants.HeaderMainDocument, spec.AttemptId)
		}
		rc := rpcontext.NewRecordContext()
		if !a.admit(rc, func() { a.lifecycle.addRecord(rc); rc.Init(1, nil, req, req.URL) }) {
			t.Fatal("not admitted")
		}
		return rc
	}
	parent := add("https://example.com/start", "", true)
	parent.ObserveResponse(redirectResponse(302, "/final"))
	parent.ObserveResponse(redirectResponse(302, "/final")) // Duplicate observation is harmless.
	main := add("https://example.com/final", "", true)
	child := add("https://example.com/image", "https://example.com/final", false)
	if child.CrawlLogSnapshot().DiscoveryPath != "LRE" {
		t.Fatal(child.CrawlLogSnapshot())
	}
	// Children finish first. No outcome has been required to establish their ancestry.
	for _, rc := range []*rpcontext.RecordContext{child, main, parent} {
		if err := rc.SaveCrawlLog(); err != nil {
			t.Fatal(err)
		}
	}
	for i, want := range []string{"LRE", "LR", "L"} {
		if a.results[i].log.DiscoveryPath != want {
			t.Fatalf("terminal path changed: %v", a.results[i].log)
		}
	}
	a.startFinish(nil, nil)
	<-a.done
	if a.discovery.byURL != nil || a.discovery.pending != nil || a.discovery.main != nil {
		t.Fatal("index retained")
	}
	main.ObserveResponse(redirectResponse(302, "/late"))
	if a.discovery.pending != nil {
		t.Fatal("late response revived index")
	}
	next := beginSpec("discovery-attempt-0002")
	next.AdmissionToken = p.nextToken
	if _, err := c.BeginHarvest(t.Context(), next); err != nil {
		t.Fatal(err)
	}
	if p.attempt.discovery.main != nil {
		t.Fatal("ancestry leaked into next attempt")
	}
}

func TestDiscoveryAvailableOnImmediateCancellation(t *testing.T) {
	c, p := testControl()
	spec := beginSpec("discovery-cancel-attempt")
	spec.DiscoveryPath = "LR"
	if _, err := c.BeginHarvest(t.Context(), spec); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	req := discoveryRequest("GET", "https://example.com/image", "https://example.com/#raw", "image")
	req = req.WithContext(rpcontext.RecordProxyDataAware(canceled))
	rc := rpcontext.NewRecordContext()
	a := p.attempt
	a.admit(rc, func() { a.lifecycle.addRecord(rc); rc.Init(1, nil, req, req.URL) })
	wait, stop := context.WithTimeout(t.Context(), time.Second)
	defer stop()
	if err := a.lifecycle.wait(wait); err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.results) != 1 {
		t.Fatalf("results=%v", a.results)
	}
	cl := a.results[0].log
	if cl.DiscoveryPath != "LRE" || cl.Referrer != "https://example.com/#raw" || cl.RequestedUri != req.URL.String() || cl.Error == nil {
		t.Fatalf("cancellation lost ancestry or raw URLs: %v", cl)
	}
}
