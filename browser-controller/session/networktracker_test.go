package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
)

func TestShouldTrackNetworkRequest(t *testing.T) {
	tests := []struct {
		name string
		ev   *network.EventRequestWillBeSent
		want bool
	}{
		{
			name: "tracks frame-associated image request",
			ev: &network.EventRequestWillBeSent{
				RequestID: network.RequestID("req-1"),
				Type:      network.ResourceTypeImage,
				FrameID:   cdp.FrameID("frame-1"),
				Request:   &network.Request{URL: "https://example.com/image.jpg"},
			},
			want: true,
		},
		{
			name: "tracks worker-associated script request with loader id",
			ev: &network.EventRequestWillBeSent{
				RequestID: network.RequestID("req-2"),
				Type:      network.ResourceTypeScript,
				LoaderID:  cdp.LoaderID("loader-1"),
				Request:   &network.Request{URL: "https://example.com/chunk.js"},
			},
			want: true,
		},
		{
			name: "ignores background xhr requests",
			ev: &network.EventRequestWillBeSent{
				RequestID: network.RequestID("req-3"),
				Type:      network.ResourceTypeXHR,
				FrameID:   cdp.FrameID("frame-1"),
				Request:   &network.Request{URL: "https://example.com/xhr"},
			},
			want: false,
		},
		{
			name: "ignores requests without frame or loader",
			ev: &network.EventRequestWillBeSent{
				RequestID: network.RequestID("req-4"),
				Type:      network.ResourceTypeImage,
				Request:   &network.Request{URL: "https://example.com/image.jpg"},
			},
			want: false,
		},
		{
			name: "ignores websocket traffic",
			ev: &network.EventRequestWillBeSent{
				RequestID: network.RequestID("req-5"),
				Type:      network.ResourceTypeWebSocket,
				FrameID:   cdp.FrameID("frame-1"),
				Request:   &network.Request{URL: "wss://example.com/socket"},
			},
			want: false,
		},
		{
			name: "ignores chrome scheme requests",
			ev: &network.EventRequestWillBeSent{
				RequestID: network.RequestID("req-6"),
				Type:      network.ResourceTypeImage,
				FrameID:   cdp.FrameID("frame-1"),
				Request:   &network.Request{URL: "chrome://settings"},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldTrackNetworkRequest(tc.ev); got != tc.want {
				t.Fatalf("shouldTrackNetworkRequest() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestShouldTrackObservedRequest(t *testing.T) {
	tests := []struct {
		name         string
		resourceType network.ResourceType
		url          string
		hasContext   bool
		want         bool
	}{
		{name: "tracks contextual image", resourceType: network.ResourceTypeImage, url: "https://example.com/image.jpg", hasContext: true, want: true},
		{name: "ignores contextual xhr", resourceType: network.ResourceTypeXHR, url: "https://example.com/xhr", hasContext: true, want: false},
		{name: "ignores missing context", resourceType: network.ResourceTypeImage, url: "https://example.com/image.jpg", hasContext: false, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldTrackObservedRequest(tc.resourceType, tc.url, tc.hasContext); got != tc.want {
				t.Fatalf("shouldTrackObservedRequest() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNetworkActivityTrackerWaitForIdleReturnsAfterQuietPeriod(t *testing.T) {
	tracker := newNetworkActivityTracker()
	tracker.noteRequestStart(&network.EventRequestWillBeSent{
		RequestID: network.RequestID("req-1"),
		Type:      network.ResourceTypeImage,
		FrameID:   cdp.FrameID("frame-1"),
		Request:   &network.Request{URL: "https://example.com/image.jpg"},
	})

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- tracker.waitForIdle(ctx, 20*time.Millisecond)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("waitForIdle() error = %v, want nil", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("waitForIdle did not complete after a quiet period")
	}
}

func TestNetworkActivityTrackerWaitForIdleExtendsOnDataReceived(t *testing.T) {
	tracker := newNetworkActivityTracker()
	tracker.noteRequestStart(&network.EventRequestWillBeSent{
		RequestID: network.RequestID("req-1"),
		Type:      network.ResourceTypeImage,
		FrameID:   cdp.FrameID("frame-1"),
		Request:   &network.Request{URL: "https://example.com/image.jpg"},
	})

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- tracker.waitForIdle(ctx, 20*time.Millisecond)
	}()

	time.Sleep(10 * time.Millisecond)
	tracker.noteRequestData(network.RequestID("req-1"))

	select {
	case err := <-done:
		t.Fatalf("waitForIdle returned too early after data activity: %v", err)
	case <-time.After(15 * time.Millisecond):
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("waitForIdle() error = %v, want nil", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("waitForIdle did not complete after data activity went quiet")
	}
}

func TestNetworkActivityTrackerWaitForIdleExtendsOnObservedRequestWithoutID(t *testing.T) {
	tracker := newNetworkActivityTracker()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- tracker.waitForIdle(ctx, 20*time.Millisecond)
	}()

	time.Sleep(10 * time.Millisecond)
	tracker.noteObservedRequestStart(network.ResourceTypeImage, "https://example.com/image.jpg", true, "")

	select {
	case err := <-done:
		t.Fatalf("waitForIdle returned too early after observed request activity: %v", err)
	case <-time.After(15 * time.Millisecond):
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("waitForIdle() error = %v, want nil", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("waitForIdle did not complete after observed request activity went quiet")
	}
}

func TestNetworkActivityTrackerWaitForIdleReturnsContextDeadline(t *testing.T) {
	tracker := newNetworkActivityTracker()
	tracker.noteRequestStart(&network.EventRequestWillBeSent{
		RequestID: network.RequestID("req-1"),
		Type:      network.ResourceTypeImage,
		FrameID:   cdp.FrameID("frame-1"),
		Request:   &network.Request{URL: "https://example.com/image.jpg"},
	})

	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Millisecond)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		defer close(stopped)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				tracker.noteRequestData(network.RequestID("req-1"))
			}
		}
	}()

	err := tracker.waitForIdle(ctx, 20*time.Millisecond)
	<-stopped
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waitForIdle() error = %v, want %v", err, context.DeadlineExceeded)
	}
}
