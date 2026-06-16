package requests

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/syncx"
)

func TestGetOrAddRequestDeduplicatesByNetworkID(t *testing.T) {
	registry := NewRegistry(syncx.NewWaitGroup(context.Background()))

	first, added := registry.GetOrAddRequest(&Request{
		RequestId:    "interception-job-1.0",
		NetworkId:    "238.39",
		Method:       "GET",
		Url:          "https://example.com/a.js",
		ResourceType: "Script",
	})
	if !added {
		t.Fatal("first request was not added")
	}

	second, added := registry.GetOrAddRequest(&Request{
		RequestId:    "interception-job-2.0",
		NetworkId:    "238.39",
		Method:       "GET",
		Url:          "https://example.com/a.js",
		ResourceType: "Script",
	})
	if added {
		t.Fatal("duplicate network ID was added as a new request")
	}
	if second != first {
		t.Fatal("duplicate network ID did not reuse the original request")
	}

	count := 0
	registry.Walk(func(*Request) {
		count++
	})
	if count != 1 {
		t.Fatalf("request count = %d, want 1", count)
	}
}

func TestGetOrAddRequestDeduplicatesByRequestID(t *testing.T) {
	registry := NewRegistry(syncx.NewWaitGroup(context.Background()))

	first, added := registry.GetOrAddRequest(&Request{
		RequestId:    "interception-job-1.0",
		Method:       "GET",
		Url:          "https://example.com/",
		ResourceType: "Document",
	})
	if !added {
		t.Fatal("first request was not added")
	}

	second, added := registry.GetOrAddRequest(&Request{
		RequestId:    "interception-job-1.0",
		Method:       "GET",
		Url:          "https://example.com/",
		ResourceType: "Document",
	})
	if added {
		t.Fatal("duplicate request ID was added as a new request")
	}
	if second != first {
		t.Fatal("duplicate request ID did not reuse the original request")
	}

	count := 0
	registry.Walk(func(*Request) {
		count++
	})
	if count != 1 {
		t.Fatalf("request count = %d, want 1", count)
	}
}

func TestMatchCrawlLogsIgnoresNonBlockingPingRequests(t *testing.T) {
	registry := NewRegistry(syncx.NewWaitGroup(context.Background()))
	registry.AddRequest(&Request{
		RequestId:    "interception-job-1.0",
		Method:       "POST",
		Url:          "https://metrics.example/ping",
		ResourceType: "Ping",
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
			registry := NewRegistry(syncx.NewWaitGroup(context.Background()))
			registry.AddRequest(&Request{
				RequestId:    "interception-job-1.0",
				Method:       "GET",
				Url:          "https://example.com/background",
				ResourceType: tc.resourceType,
			})

			if !registry.MatchCrawlLogs() {
				t.Fatalf("non-blocking %s request should not keep crawlLog matching unresolved", tc.resourceType)
			}
		})
	}
}

func TestRemoveRequestRemovesTrackedRequest(t *testing.T) {
	registry := NewRegistry(syncx.NewWaitGroup(context.Background()))
	kept, added := registry.GetOrAddRequest(&Request{
		RequestId:    "interception-job-1.0",
		Method:       "GET",
		Url:          "https://example.com/",
		ResourceType: "Document",
	})
	if !added {
		t.Fatal("initial request was not added")
	}
	removed, added := registry.GetOrAddRequest(&Request{
		RequestId:    "interception-job-2.0",
		Method:       "GET",
		Url:          "https://example.com/image.png",
		ResourceType: "Image",
	})
	if !added {
		t.Fatal("second request was not added")
	}

	if !registry.RemoveRequest(removed) {
		t.Fatal("expected request removal to succeed")
	}

	count := 0
	registry.Walk(func(req *Request) {
		count++
		if req != kept {
			t.Fatalf("unexpected request left in registry: %#v", req)
		}
	})
	if count != 1 {
		t.Fatalf("request count = %d, want 1", count)
	}
}

func TestRequestBlocksPageCompletion(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		want         bool
	}{
		{name: "document blocks", resourceType: "Document", want: true},
		{name: "ping does not block", resourceType: "Ping", want: false},
		{name: "event source does not block", resourceType: "EventSource", want: false},
		{name: "xhr does not block", resourceType: "XHR", want: false},
		{name: "fetch does not block", resourceType: "Fetch", want: false},
		{name: "websocket does not block", resourceType: "WebSocket", want: false},
		{name: "image still blocks", resourceType: "Image", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &Request{ResourceType: tc.resourceType}
			if got := req.BlocksPageCompletion(); got != tc.want {
				t.Fatalf("BlocksPageCompletion() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBuildCrawlLogMatchSnapshotSummarizesRequests(t *testing.T) {
	crawlLog := &logV1.CrawlLog{}
	snapshot := buildCrawlLogMatchSnapshot([]*Request{
		{RequestId: "interception-job-1.0", ResourceType: "Document", CrawlLog: crawlLog},
		{RequestId: "interception-job-2.0", ResourceType: "Image"},
		{RequestId: "interception-job-3.0", ResourceType: "XHR"},
		{RequestId: "interception-job-4.0", ResourceType: "Script", CrawlLog: crawlLog},
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
	if snapshot.ignoredCount != 1 {
		t.Fatalf("ignoredCount = %d, want 1", snapshot.ignoredCount)
	}
	if want := []string{"interception-job-2.0"}; !reflect.DeepEqual(snapshot.missingRequestIDs, want) {
		t.Fatalf("missingRequestIDs = %v, want %v", snapshot.missingRequestIDs, want)
	}
}

func TestBuildCrawlLogMatchSnapshotLimitsMissingIDs(t *testing.T) {
	requests := make([]*Request, 0, maxLoggedMissingRequestIDs+2)
	for i := 0; i < maxLoggedMissingRequestIDs+2; i++ {
		requests = append(requests, &Request{RequestId: fmt.Sprintf("interception-job-%d.0", i+1), ResourceType: "Image"})
	}

	snapshot := buildCrawlLogMatchSnapshot(requests)
	if snapshot.unresolvedCount != maxLoggedMissingRequestIDs+2 {
		t.Fatalf("unresolvedCount = %d, want %d", snapshot.unresolvedCount, maxLoggedMissingRequestIDs+2)
	}
	if len(snapshot.missingRequestIDs) != maxLoggedMissingRequestIDs {
		t.Fatalf("len(missingRequestIDs) = %d, want %d", len(snapshot.missingRequestIDs), maxLoggedMissingRequestIDs)
	}
}
