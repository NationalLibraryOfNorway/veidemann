package session

import (
	"errors"
	"testing"

	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/requests"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/syncx"
	"github.com/chromedp/cdproto/target"
)

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

func TestFindRootRequestReuseCandidateAcceptsOtherDuplicate(t *testing.T) {
	registry := requests.NewRegistry(syncx.NewWaitGroup(t.Context()))
	root, added := registry.GetOrAddRequest(&requests.Request{
		Method:       "GET",
		Url:          "https://www.nb.no/",
		RequestId:    "interception-job-1.0",
		NetworkId:    "237.1",
		ResourceType: "Document",
	})
	if !added {
		t.Fatal("root request was not added")
	}

	sess := &Session{
		RequestedUrl: &frontierV1.QueuedUri{Uri: "https://www.nb.no/"},
		Requests:     registry,
	}

	reused := sess.findRootRequestReuseCandidate(&requests.Request{
		Method:       "GET",
		Url:          "https://www.nb.no/",
		RequestId:    "interception-job-35.0",
		ResourceType: "Other",
	})
	if reused != root {
		t.Fatal("root duplicate was not reused")
	}
}

func TestFindRootRequestReuseCandidateAcceptsCompletedRoot(t *testing.T) {
	registry := requests.NewRegistry(syncx.NewWaitGroup(t.Context()))
	root, added := registry.GetOrAddRequest(&requests.Request{
		Method:       "GET",
		Url:          "https://www.nb.no/",
		RequestId:    "interception-job-1.0",
		NetworkId:    "239.1",
		ResourceType: "Document",
		GotComplete:  true,
	})
	if !added {
		t.Fatal("root request was not added")
	}

	sess := &Session{
		RequestedUrl: &frontierV1.QueuedUri{Uri: "https://www.nb.no/"},
		Requests:     registry,
	}

	reused := sess.findRootRequestReuseCandidate(&requests.Request{
		Method:       "GET",
		Url:          "https://www.nb.no/",
		RequestId:    "interception-job-35.0",
		ResourceType: "Other",
	})
	if reused != root {
		t.Fatal("completed root duplicate was not reused")
	}
}

func TestRollbackPausedRequestRemovesAddedNonInitialRequest(t *testing.T) {
	registry := requests.NewRegistry(syncx.NewWaitGroup(t.Context()))
	_, added := registry.GetOrAddRequest(&requests.Request{
		Method:       "GET",
		Url:          "https://www.nb.no/",
		RequestId:    "interception-job-1.0",
		NetworkId:    "240.1",
		ResourceType: "Document",
	})
	if !added {
		t.Fatal("initial request was not added")
	}
	orphan, added := registry.GetOrAddRequest(&requests.Request{
		Method:       "GET",
		Url:          "https://www.nb.no/image.jpg",
		RequestId:    "interception-job-2.0",
		NetworkId:    "240.2",
		ResourceType: "Image",
	})
	if !added {
		t.Fatal("orphan request was not added")
	}

	sess := &Session{Requests: registry}
	sess.rollbackPausedRequest(orphan, true)

	count := 0
	registry.Walk(func(req *requests.Request) {
		count++
		if req == orphan {
			t.Fatal("orphan request was not removed")
		}
	})
	if count != 1 {
		t.Fatalf("request count = %d, want 1", count)
	}
}

func TestRollbackPausedRequestKeepsInitialRequest(t *testing.T) {
	registry := requests.NewRegistry(syncx.NewWaitGroup(t.Context()))
	initial, added := registry.GetOrAddRequest(&requests.Request{
		Method:       "GET",
		Url:          "https://www.nb.no/",
		RequestId:    "interception-job-1.0",
		NetworkId:    "241.1",
		ResourceType: "Document",
	})
	if !added {
		t.Fatal("initial request was not added")
	}

	sess := &Session{Requests: registry}
	sess.rollbackPausedRequest(initial, true)

	count := 0
	registry.Walk(func(req *requests.Request) {
		count++
		if req != initial {
			t.Fatalf("unexpected request left in registry: %#v", req)
		}
	})
	if count != 1 {
		t.Fatalf("request count = %d, want 1", count)
	}
}
