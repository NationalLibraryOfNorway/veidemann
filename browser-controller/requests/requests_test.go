package requests

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
)

func TestBoundedURLForLog(t *testing.T) {
	t.Run("short URL", func(t *testing.T) {
		const rawURL = "https://example.com/image.png"
		got, length := BoundedURLForLog(rawURL)
		if got != rawURL {
			t.Fatalf("BoundedURLForLog() = %q, want %q", got, rawURL)
		}
		if length != len(rawURL) {
			t.Fatalf("length = %d, want %d", length, len(rawURL))
		}
	})

	t.Run("long URL", func(t *testing.T) {
		rawURL := "https://example.com/" + strings.Repeat("a", maxLoggedURLBytes)
		got, length := BoundedURLForLog(rawURL)
		if len(got) > maxLoggedURLBytes {
			t.Fatalf("bounded URL length = %d, want at most %d", len(got), maxLoggedURLBytes)
		}
		if !strings.HasSuffix(got, "…") {
			t.Fatalf("bounded URL %q does not end with an ellipsis", got)
		}
		if length != len(rawURL) {
			t.Fatalf("length = %d, want %d", length, len(rawURL))
		}
	})

	t.Run("unicode boundary", func(t *testing.T) {
		rawURL := strings.Repeat("a", maxLoggedURLBytes-4) + "øtail"
		got, length := BoundedURLForLog(rawURL)
		if !strings.HasSuffix(got, "…") {
			t.Fatalf("bounded URL %q does not end with an ellipsis", got)
		}
		if !utf8.ValidString(got) {
			t.Fatalf("bounded URL is not valid UTF-8: %q", got)
		}
		if length != len(rawURL) {
			t.Fatalf("length = %d, want %d", length, len(rawURL))
		}
	})
}

func TestResourceLogAttrs(t *testing.T) {
	req := &Request{
		ID:             "network-1",
		FetchRequestID: "fetch-1",
		NetworkID:      "network-1",
		URL:            "https://example.com/image.png",
		Method:         "GET",
		ResourceType:   "Image",
		Initiator:      "parser",
		GotNew:         true,
		GotComplete:    true,
		CrawlLog: &logV1.CrawlLog{
			StatusCode: 200,
			WarcId:     "warc-1",
		},
	}

	attrs := attrsByName(resourceLogAttrs(req))
	for name, want := range map[string]any{
		"requestId":      "network-1",
		"fetchRequestId": "fetch-1",
		"networkId":      "network-1",
		"url":            req.URL,
		"urlLength":      len(req.URL),
		"hasCrawlLog":    true,
		"statusCode":     int32(200),
		"warcId":         "warc-1",
	} {
		if got := attrs[name]; !reflect.DeepEqual(got, want) {
			t.Errorf("attribute %q = %#v, want %#v", name, got, want)
		}
	}

	missing := attrsByName(resourceLogAttrs(&Request{ID: "missing-1", URL: req.URL}))
	if got := missing["hasCrawlLog"]; got != false {
		t.Errorf("hasCrawlLog = %#v, want false", got)
	}
	if _, ok := missing["statusCode"]; ok {
		t.Error("missing resource unexpectedly has statusCode")
	}
}

func attrsByName(attrs []any) map[string]any {
	result := make(map[string]any, len(attrs)/2)
	for i := 0; i < len(attrs); i += 2 {
		result[attrs[i].(string)] = attrs[i+1]
	}
	return result
}

func TestGetOrAddRequestDeduplicatesByID(t *testing.T) {
	registry := NewRegistry(nil)

	first, added := registry.GetOrAddRequest(&Request{
		ID:             "238.39",
		FetchRequestID: "interception-job-1.0",
		NetworkID:      "238.39",
		Method:         "GET",
		URL:            "https://example.com/a.js",
		ResourceType:   "Script",
	})
	if !added {
		t.Fatal("first request was not added")
	}

	second, added := registry.GetOrAddRequest(&Request{
		ID:             "238.39",
		FetchRequestID: "interception-job-2.0",
		NetworkID:      "238.39",
		Method:         "GET",
		URL:            "https://example.com/a.js",
		ResourceType:   "Script",
	})
	if added {
		t.Fatal("duplicate network ID was added as a new request")
	}
	if second != first {
		t.Fatal("duplicate network ID did not reuse the original request")
	}

	if count := len(registry.requests); count != 1 {
		t.Fatalf("request count = %d, want 1", count)
	}
}

func TestRootRequestSnapshotResolvesRedirectRoot(t *testing.T) {
	registry := NewRegistry(nil)
	initial := &Request{
		ID:           "network-1",
		URL:          "https://example.com/old",
		ResourceType: "Document",
	}
	redirect := &Request{
		ID:              "network-2",
		URL:             "https://example.com/new",
		ResourceType:    "Document",
		Redirected:      true,
		RedirectFromURL: initial.URL,
	}
	registry.AddRequest(initial)
	registry.AddRequest(redirect)

	root := registry.RootRequestSnapshot()

	if root == nil || root.ID != redirect.ID {
		t.Fatal("redirect target was not selected as the root request")
	}
	if root == redirect {
		t.Fatal("root snapshot retained pointers into the live registry")
	}
	if redirect.RedirectParent != nil {
		t.Fatal("root snapshot mutated the live redirect relationship")
	}
	if initial.CrawlLog != nil || redirect.CrawlLog != nil {
		t.Fatal("root snapshot mutated the live registry")
	}
}

func TestFinalizeResponsesReturnsIndependentSnapshot(t *testing.T) {
	registry := NewRegistry(nil)
	initialLog := &logV1.CrawlLog{WarcId: "warc-1", Size: 10}
	initial := &Request{
		ID:           "network-1",
		URL:          "https://example.com/",
		Method:       "GET",
		ResourceType: "Document",
		GotNew:       true,
		GotComplete:  true,
		CrawlLog:     initialLog,
	}
	late := &Request{
		ID:           "network-2",
		URL:          "https://example.com/late.js",
		Method:       "GET",
		ResourceType: "Script",
		GotNew:       true,
	}
	registry.AddRequest(initial)
	registry.AddRequest(late)

	snapshot := registry.FinalizeResponses(&frontierV1.QueuedUri{DiscoveryPath: "L"})
	if snapshot.InitialRequest == initial || snapshot.RootRequest == initial {
		t.Fatal("final snapshot retained pointers into the live registry")
	}
	if snapshot.InitialRequest.CrawlLog == initialLog {
		t.Fatal("final snapshot retained the live crawl log")
	}
	if got := snapshot.InitialRequest.CrawlLog.GetDiscoveryPath(); got != "L" {
		t.Fatalf("initial discovery path = %q, want L", got)
	}
	if snapshot.Requests[1].CrawlLog != nil {
		t.Fatal("unfinished request unexpectedly had a crawl log in the snapshot")
	}

	lateLog := &logV1.CrawlLog{WarcId: "warc-2", Size: 20}
	registry.CompleteRequest(late.ID, lateLog, false)
	initialLog.Size = 99
	if snapshot.Requests[1].CrawlLog != nil {
		t.Fatal("late completion mutated the finalized snapshot")
	}
	if got := snapshot.InitialRequest.CrawlLog.GetSize(); got != 10 {
		t.Fatalf("snapshot crawl log size = %d, want 10", got)
	}

	next := registry.FinalizeResponses(&frontierV1.QueuedUri{DiscoveryPath: "L"})
	if next.Requests[1].CrawlLog == nil {
		t.Fatal("completion before finalization was not included")
	}
}

func TestMatchCrawlLogsIgnoresNonBlockingPingRequests(t *testing.T) {
	registry := NewRegistry(nil)
	registry.AddRequest(&Request{
		ID:             "238.39",
		FetchRequestID: "interception-job-1.0",
		Method:         "POST",
		URL:            "https://metrics.example/ping",
		ResourceType:   "Ping",
	})

	if !registry.MatchCrawlLogs() {
		t.Fatal("non-blocking ping request should not keep crawlLog matching unresolved")
	}
}

func TestMatchCrawlLogsIgnoresNonBlockingBackgroundRequests(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
	}{
		{name: "event source", resourceType: "EventSource"},
		{name: "xhr", resourceType: "XHR"},
		{name: "fetch", resourceType: "Fetch"},
		{name: "websocket", resourceType: "WebSocket"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewRegistry(nil)
			registry.AddRequest(&Request{
				ID:             "211.11",
				FetchRequestID: "interception-job-1.0",
				Method:         "GET",
				URL:            "https://example.com/background",
				ResourceType:   tc.resourceType,
			})

			if !registry.MatchCrawlLogs() {
				t.Fatalf("non-blocking %s request should not keep crawlLog matching unresolved", tc.resourceType)
			}
		})
	}
}

func TestRemoveRequestRemovesTrackedRequest(t *testing.T) {
	registry := NewRegistry(nil)
	kept, added := registry.GetOrAddRequest(&Request{
		ID:             "240.1",
		FetchRequestID: "interception-job-1.0",
		Method:         "GET",
		URL:            "https://example.com/",
		ResourceType:   "Document",
	})
	if !added {
		t.Fatal("initial request was not added")
	}
	removed, added := registry.GetOrAddRequest(&Request{
		ID:             "240.2",
		FetchRequestID: "interception-job-2.0",
		Method:         "GET",
		URL:            "https://example.com/image.png",
		ResourceType:   "Image",
	})
	if !added {
		t.Fatal("second request was not added")
	}

	if !registry.RemoveRequest(removed) {
		t.Fatal("expected request removal to succeed")
	}

	if count := len(registry.requests); count != 1 {
		t.Fatalf("request count = %d, want 1", count)
	}
	if registry.requests[0] != kept {
		t.Fatalf("unexpected request left in registry: %#v", registry.requests[0])
	}
}
func TestRequestBlocksPageCompletion(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		gotNew       bool
		want         bool
	}{
		{name: "document blocks when new", resourceType: "Document", gotNew: true, want: true},
		{name: "document does not block when not new", resourceType: "Document", gotNew: false, want: false},

		{name: "ping does not block", resourceType: "Ping", gotNew: true, want: false},
		{name: "event source does not block", resourceType: "EventSource", gotNew: true, want: false},
		{name: "xhr does not block", resourceType: "XHR", gotNew: true, want: false},
		{name: "fetch does not block", resourceType: "Fetch", gotNew: true, want: false},
		{name: "websocket does not block", resourceType: "WebSocket", gotNew: true, want: false},

		{name: "image blocks when new", resourceType: "Image", gotNew: true, want: true},
		{name: "image does not block when not new", resourceType: "Image", gotNew: false, want: false},

		{name: "script blocks when new", resourceType: "Script", gotNew: true, want: true},
		{name: "stylesheet blocks when new", resourceType: "Stylesheet", gotNew: true, want: true},
		{name: "font blocks when new", resourceType: "Font", gotNew: true, want: true},
		{name: "media blocks when new", resourceType: "Media", gotNew: true, want: true},
		{name: "text track blocks when new", resourceType: "TextTrack", gotNew: true, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &Request{
				ResourceType: tc.resourceType,
				GotNew:       tc.gotNew,
			}

			if got := req.BlocksPageCompletion(); got != tc.want {
				t.Fatalf("BlocksPageCompletion() = %v, want %v", got, tc.want)
			}
		})
	}
}
func TestBuildCrawlLogMatchSnapshotSummarizesRequests(t *testing.T) {
	crawlLog := &logV1.CrawlLog{}

	snapshot := buildCrawlLogMatchSnapshot([]*Request{
		{
			ID:             "network-1",
			FetchRequestID: "interception-job-1.0",
			NetworkID:      "network-1",
			URL:            "https://example.com/",
			ResourceType:   "Document",
			GotNew:         true,
			GotComplete:    true,
			CrawlLog:       crawlLog,
		},
		{
			ID:             "network-2",
			FetchRequestID: "interception-job-2.0",
			NetworkID:      "network-2",
			URL:            "https://example.com/image.png",
			ResourceType:   "Image",
			GotNew:         true,
		},
		{
			ID:             "network-3",
			FetchRequestID: "interception-job-3.0",
			NetworkID:      "network-3",
			URL:            "https://example.com/api",
			ResourceType:   "XHR",
			GotNew:         true,
		},
		{
			ID:             "network-4",
			FetchRequestID: "interception-job-4.0",
			NetworkID:      "network-4",
			URL:            "https://example.com/app.js",
			ResourceType:   "Script",
			GotNew:         true,
			GotComplete:    true,
			CrawlLog:       crawlLog,
		},
		{
			ID:             "network-5",
			FetchRequestID: "interception-job-5.0",
			NetworkID:      "network-5",
			URL:            "https://example.com/not-yet-seen.png",
			ResourceType:   "Image",
			GotNew:         false,
		},
	})

	if snapshot.blockingCount != 3 {
		t.Fatalf("blockingCount = %d, want 3", snapshot.blockingCount)
	}
	if snapshot.resolvedCount != 2 {
		t.Fatalf("resolvedCount = %d, want 2", snapshot.resolvedCount)
	}
	if snapshot.unresolvedCount != 1 {
		t.Fatalf("unresolvedCount = %d, want 1", snapshot.unresolvedCount)
	}
	if snapshot.ignoredCount != 2 {
		t.Fatalf("ignoredCount = %d, want 2", snapshot.ignoredCount)
	}

	want := []missingRequestSummary{
		{
			ID:             "network-2",
			FetchRequestID: "interception-job-2.0",
			NetworkID:      "network-2",
			URL:            "https://example.com/image.png",
			URLLength:      len("https://example.com/image.png"),
			ResourceType:   "Image",
			GotNew:         true,
			GotComplete:    false,
			FromCache:      false,
		},
	}

	if !reflect.DeepEqual(snapshot.missingRequests, want) {
		t.Fatalf("missingRequests = %#v, want %#v", snapshot.missingRequests, want)
	}
}
func TestBuildCrawlLogMatchSnapshotLimitsMissingRequests(t *testing.T) {
	requests := make([]*Request, 0, maxLoggedMissingRequestIDs+2)

	for i := 0; i < maxLoggedMissingRequestIDs+2; i++ {
		n := i + 1

		requests = append(requests, &Request{
			ID:             fmt.Sprintf("network-%d", n),
			FetchRequestID: fmt.Sprintf("interception-job-%d.0", n),
			NetworkID:      fmt.Sprintf("network-%d", n),
			URL:            fmt.Sprintf("https://example.com/image-%d.png", n),
			ResourceType:   "Image",
			GotNew:         true,
		})
	}

	snapshot := buildCrawlLogMatchSnapshot(requests)

	if snapshot.unresolvedCount != maxLoggedMissingRequestIDs+2 {
		t.Fatalf("unresolvedCount = %d, want %d", snapshot.unresolvedCount, maxLoggedMissingRequestIDs+2)
	}

	if len(snapshot.missingRequests) != maxLoggedMissingRequestIDs {
		t.Fatalf("len(missingRequests) = %d, want %d", len(snapshot.missingRequests), maxLoggedMissingRequestIDs)
	}

	for i, missing := range snapshot.missingRequests {
		n := i + 1

		want := missingRequestSummary{
			ID:             fmt.Sprintf("network-%d", n),
			FetchRequestID: fmt.Sprintf("interception-job-%d.0", n),
			NetworkID:      fmt.Sprintf("network-%d", n),
			URL:            fmt.Sprintf("https://example.com/image-%d.png", n),
			URLLength:      len(fmt.Sprintf("https://example.com/image-%d.png", n)),
			ResourceType:   "Image",
			GotNew:         true,
			GotComplete:    false,
			FromCache:      false,
		}

		if !reflect.DeepEqual(missing, want) {
			t.Fatalf("missingRequests[%d] = %#v, want %#v", i, missing, want)
		}
	}
}
