package session

import (
	"errors"
	"testing"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/requests"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/url"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/target"
)

func TestNetworkRequestRegistrationIgnoresBrowserLocalURLs(t *testing.T) {
	registry := requests.NewRegistry(nil)
	sess := &Session{
		Requests:       registry,
		networkTracker: newNetworkActivityTracker(),
	}
	sess.startAcceptingRequests()

	sess.onNetworkEventRequestWillBeSent(t.Context(), &network.EventRequestWillBeSent{
		RequestID: network.RequestID("local-1"),
		Request:   &network.Request{URL: "data:image/png;base64,AAAA", Method: "GET"},
		Initiator: &network.Initiator{Type: network.InitiatorTypeParser},
		Type:      network.ResourceTypeImage,
	}, 1)

	if got := registry.InitialRequest(); got != nil {
		t.Fatalf("browser-local request was registered: %#v", got)
	}
}

func TestNetworkRequestRegistrationPreservesCanonicalMerge(t *testing.T) {
	registry := requests.NewRegistry(nil)
	sess := &Session{
		Requests:       registry,
		networkTracker: newNetworkActivityTracker(),
	}
	sess.startAcceptingRequests()

	sess.onNetworkEventRequestWillBeSent(t.Context(), &network.EventRequestWillBeSent{
		RequestID: network.RequestID("network-1"),
		Request:   &network.Request{URL: "https://example.com/image.png", Method: "GET"},
		Initiator: &network.Initiator{Type: network.InitiatorTypeParser},
		Type:      network.ResourceTypeImage,
	}, 1)

	fromNetwork := registry.InitialRequest()
	if fromNetwork == nil {
		t.Fatal("network request was not registered")
	}
	fromFetch, added := registry.GetOrAddRequest(&requests.Request{
		ID:             "network-1",
		FetchRequestID: "interception-1",
		NetworkID:      "network-1",
		URL:            "https://example.com/image.png",
		Method:         "GET",
		ResourceType:   "Image",
	})
	if added {
		t.Fatal("Fetch observation created a duplicate registry entry")
	}
	if fromFetch != fromNetwork {
		t.Fatal("Fetch observation did not merge with the canonical Network request")
	}
	if fromFetch.FetchRequestID != "interception-1" {
		t.Fatalf("FetchRequestID = %q, want interception-1", fromFetch.FetchRequestID)
	}
}

func TestRequestFromFetchPausedIdentifiesBrowserLocalURL(t *testing.T) {
	ev := &fetch.EventRequestPaused{
		RequestID:    fetch.RequestID("interception-1"),
		NetworkID:    network.RequestID("network-1"),
		Request:      &network.Request{URL: "blob:https://example.com/id", Method: "GET"},
		ResourceType: network.ResourceTypeScript,
	}

	if !url.IsBrowserLocal(ev.Request.URL) {
		t.Fatal("browser-local Fetch request was not identified before registration")
	}
}

func TestLoadingFailedThenRecorderCompletionKeepsRecorderCrawlLog(t *testing.T) {
	registry := requests.NewRegistry(nil)
	req, added := registry.GetOrAddRequest(&requests.Request{
		ID:           "request-1",
		NetworkID:    "request-1",
		URL:          "https://example.com/",
		Method:       "GET",
		ResourceType: "Document",
		GotNew:       true,
	})
	if !added {
		t.Fatal("initial request was not added")
	}

	sess := &Session{
		Requests:       registry,
		networkTracker: newNetworkActivityTracker(),
	}
	sess.onNetworkEventLoadingFailed(t.Context(), &network.EventLoadingFailed{
		RequestID: network.RequestID(req.ID),
		Type:      network.ResourceTypeDocument,
		ErrorText: "net::ERR_CONNECTION_REFUSED",
	}, 1)

	if !req.GotComplete {
		t.Fatal("loadingFailed did not complete the initial request")
	}
	if req.CrawlLog != nil {
		t.Fatalf("loadingFailed fabricated a crawl log: %#v", req.CrawlLog)
	}

	recorderLog := &logV1.CrawlLog{RequestedUri: req.URL, StatusCode: -2}
	registry.CompleteRequest(req.ID, recorderLog, false)

	if req.CrawlLog != recorderLog {
		t.Fatalf("recorder crawl log was not retained: got %#v, want %#v", req.CrawlLog, recorderLog)
	}
	if registry.InitialRequest() != req || !registry.InitialRequest().GotComplete {
		t.Fatal("initial request was not complete after recorder completion")
	}
}

func TestMarkTargetInitialized(t *testing.T) {
	sess := &Session{}

	if got := sess.markTargetInitialized(target.ID("child-1")); !got {
		t.Fatal("first target initialization was rejected")
	}

	if got := sess.markTargetInitialized(target.ID("child-1")); got {
		t.Fatal("duplicate target initialization was accepted")
	}

	if got := sess.markTargetInitialized(target.ID("child-2")); !got {
		t.Fatal("second unique target initialization was rejected")
	}
}

func TestMarkTargetInitializedRejectsRootTargetSecondTime(t *testing.T) {
	sess := &Session{}
	if got := sess.markTargetInitialized(target.ID("root-target")); !got {
		t.Fatal("root target initialization was rejected")
	}

	if got := sess.markTargetInitialized(target.ID("root-target")); got {
		t.Fatal("listener target was accepted for child target initialization")
	}
}

func TestIsInvalidInterceptionIDError(t *testing.T) {
	if !isInvalidInterceptionIDError(errors.New("Invalid InterceptionId. (-32602)")) {
		t.Fatal("expected Invalid InterceptionId error to be classified as benign")
	}
	if isInvalidInterceptionIDError(errors.New("other failure")) {
		t.Fatal("unexpected benign classification for unrelated error")
	}
}

func TestShouldInitChildTarget(t *testing.T) {
	tests := []struct {
		name       string
		targetType string
		want       bool
	}{
		{name: "iframe", targetType: "iframe", want: true},
		{name: "page", targetType: "page", want: false},
		{name: "worker", targetType: "worker", want: false},
		{name: "service worker", targetType: "service_worker", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldInitChildTarget(tc.targetType); got != tc.want {
				t.Fatalf("shouldInitChildTarget(%q) = %v, want %v", tc.targetType, got, tc.want)
			}
		})
	}
}

func TestShouldTrackFrameLifecycle(t *testing.T) {
	sess := &Session{rootTargetID: target.ID("root-target")}

	tests := []struct {
		name      string
		targetID  target.ID
		wantTrack bool
	}{
		{name: "root target", targetID: target.ID("root-target"), wantTrack: true},
		{name: "child iframe target", targetID: target.ID("child-iframe"), wantTrack: false},
		{name: "missing target", targetID: "", wantTrack: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sess.shouldTrackFrameLifecycleTarget(tc.targetID); got != tc.wantTrack {
				t.Fatalf("shouldTrackFrameLifecycle(%q) = %v, want %v", tc.targetID, got, tc.wantTrack)
			}
		})
	}
}

func TestRollbackPausedRequestRemovesAddedNonInitialRequest(t *testing.T) {
	registry := requests.NewRegistry(nil)
	_, added := registry.GetOrAddRequest(&requests.Request{
		Method:         "GET",
		URL:            "https://www.nb.no/",
		ID:             "240.1",
		FetchRequestID: "interception-job-1.0",
		NetworkID:      "240.1",
		ResourceType:   "Document",
	})
	if !added {
		t.Fatal("initial request was not added")
	}
	orphan, added := registry.GetOrAddRequest(&requests.Request{
		Method:         "GET",
		URL:            "https://www.nb.no/image.jpg",
		ID:             "240.2",
		FetchRequestID: "interception-job-2.0",
		NetworkID:      "240.2",
		ResourceType:   "Image",
	})
	if !added {
		t.Fatal("orphan request was not added")
	}

	sess := &Session{Requests: registry}
	sess.rollbackPausedRequest(orphan, true)

	if registry.RemoveRequest(orphan) {
		t.Fatal("orphan request was not removed")
	}
	if got := registry.InitialRequest(); got == nil || got.ID == orphan.ID {
		t.Fatalf("unexpected initial request after rollback: %#v", got)
	}
}

func TestRollbackPausedRequestRemovesAddedInitialRequest(t *testing.T) {
	registry := requests.NewRegistry(nil)
	initial, added := registry.GetOrAddRequest(&requests.Request{
		Method:         "GET",
		URL:            "https://www.nb.no/",
		ID:             "241.1",
		FetchRequestID: "interception-job-1.0",
		NetworkID:      "241.1",
		ResourceType:   "Document",
	})
	if !added {
		t.Fatal("initial request was not added")
	}

	sess := &Session{Requests: registry}
	sess.rollbackPausedRequest(initial, added)

	if got := registry.InitialRequest(); got != nil {
		t.Fatalf("request remained after rollback: %#v", got)
	}
}
