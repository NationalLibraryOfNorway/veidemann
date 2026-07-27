package server

import (
	"context"
	"testing"

	browsercontrollerV2 "github.com/NationalLibraryOfNorway/veidemann/api/browsercontroller/v2"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/requests"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/syncx"
)

func TestRegisterResourceCancelsReleasedSession(t *testing.T) {
	server := NewApiServer(session.NewRegistry(2), nil, nil)
	reply, err := server.RegisterResource(context.Background(), &browsercontrollerV2.RegisterResourceRequest{
		ProxyId:   1,
		RequestId: "network-1",
		Method:    "GET",
		Uri:       "https://example.com/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.GetCancel() != cancelledByBrowserController {
		t.Fatalf("cancel reason = %q, want %q", reply.GetCancel(), cancelledByBrowserController)
	}
}

func TestCompleteResourceAcceptsLateBrowserCancellationForReleasedSession(t *testing.T) {
	server := NewApiServer(session.NewRegistry(2), nil, nil)
	_, err := server.CompleteResource(context.Background(), &browsercontrollerV2.CompleteResourceRequest{
		ProxyId:   1,
		RequestId: "network-1",
		CrawlLog: &logV1.CrawlLog{
			StatusCode:   canceledByBrowserStatusCode,
			Method:       "GET",
			RequestedUri: "https://example.com/events",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRegisterResourceRejectsStaleExecution(t *testing.T) {
	registry := session.NewRegistry(2)
	active, err := registry.GetNextAvailable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Release(active)
	active.RequestedUrl = &frontierV1.QueuedUri{ExecutionId: "active"}

	server := NewApiServer(registry, nil, nil)
	reply, err := server.RegisterResource(context.Background(), &browsercontrollerV2.RegisterResourceRequest{
		ProxyId:          int32(active.Id),
		RequestId:        "network-1",
		Method:           "GET",
		Uri:              "https://example.com/",
		CrawlExecutionId: "stale",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.GetCancel() != cancelledByBrowserController {
		t.Fatalf("cancel reason = %q, want %q", reply.GetCancel(), cancelledByBrowserController)
	}
}

func TestCompleteResourceRejectsStaleExecution(t *testing.T) {
	registry := session.NewRegistry(2)
	active, err := registry.GetNextAvailable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Release(active)
	active.RequestedUrl = &frontierV1.QueuedUri{ExecutionId: "active"}
	active.Requests = requests.NewRegistry(syncx.NewWaitGroup(t.Context()))
	request := &requests.Request{ID: "network-1", URL: "https://example.com/"}
	active.Requests.AddRequest(request)

	server := NewApiServer(registry, nil, nil)
	_, err = server.CompleteResource(context.Background(), &browsercontrollerV2.CompleteResourceRequest{
		ProxyId:   int32(active.Id),
		RequestId: request.ID,
		CrawlLog: &logV1.CrawlLog{
			ExecutionId:  "stale",
			StatusCode:   200,
			Method:       "GET",
			RequestedUri: request.URL,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.CrawlLog != nil {
		t.Fatal("stale completion mutated the active session")
	}
}
