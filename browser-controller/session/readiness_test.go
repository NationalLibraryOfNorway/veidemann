package session

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/chromedp/cdproto/network"
)

func TestReadinessIdleMediaDoesNotRequireEOF(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sess := testHarvestSession()
		observeBrowserRequest(sess, "video", "Media")
		start := time.Now()
		if err := sess.waitForBrowserReadiness(t.Context(), time.Second); err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); elapsed != time.Second {
			t.Fatalf("elapsed=%v, want 1s", elapsed)
		}
	})
}

func TestReadinessMediaRetriesShareOneAllowance(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sess := testHarvestSession()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go func() {
			for n := 0; ; n++ {
				select {
				case <-ctx.Done():
					return
				case <-time.After(200 * time.Millisecond):
					id := fmt.Sprintf("video-%d", n)
					observeBrowserRequest(sess, id, "Media")
					finishBrowserRequest(sess, id, -5011)
					// Even a repeated attempt to start the allowance cannot reset it.
					sess.networkTracker.startMediaAllowance(5 * time.Second)
				}
			}
		}()
		start := time.Now()
		if err := sess.waitForNetworkIdle(ctx, time.Second); err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); elapsed != 5*time.Second {
			t.Fatalf("script wait=%v, want 5s", elapsed)
		}
		if err := sess.waitForBrowserReadiness(ctx, time.Second); err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); elapsed != 6*time.Second {
			t.Fatalf("shared wait=%v, want 6s", elapsed)
		}
		cancel()
		synctest.Wait()
	})
}

func TestReadinessContinuousMediaReachesAllowance(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sess := testHarvestSession()
		observeBrowserRequest(sess, "video", "Media")
		sess.networkTracker.noteObservedRequestStart(network.ResourceTypeMedia, "https://example.com/video", true, "video")
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(200 * time.Millisecond):
					sess.networkTracker.noteRequestData("video")
				}
			}
		}()
		start := time.Now()
		if err := sess.waitForBrowserReadiness(ctx, time.Second); err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); elapsed != 5*time.Second {
			t.Fatalf("wait=%v, want 5s", elapsed)
		}
		cancel()
		synctest.Wait()
	})
}

func TestReadinessProgressResumesAfterQuietWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sess := testHarvestSession()
		observeBrowserRequest(sess, "image", "Image")
		sess.networkTracker.noteObservedRequestStart(network.ResourceTypeImage, "https://example.com/image", true, "image")
		go func() {
			time.Sleep(1200 * time.Millisecond)
			sess.onNetworkEventDataReceived(t.Context(), &network.EventDataReceived{RequestID: "image", EncodedDataLength: 10}, 1)
			time.Sleep(1200 * time.Millisecond)
			finishBrowserRequest(sess, "image", 200)
		}()
		start := time.Now()
		if err := sess.waitForBrowserReadiness(t.Context(), 500*time.Millisecond); err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); elapsed != 3400*time.Millisecond {
			t.Fatalf("wait=%v, want 3.4s", elapsed)
		}
	})
}

func TestReadinessStaleNotificationsDoNotExtendStall(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sess := testHarvestSession()
		observeBrowserRequest(sess, "script", "Script")
		go func() {
			for range 10 {
				time.Sleep(100 * time.Millisecond)
				sess.networkTracker.notify()
			}
		}()
		start := time.Now()
		err := sess.waitForBrowserReadiness(t.Context(), time.Second)
		if !errors.Is(err, errCompletionIdleTimeout) {
			t.Fatalf("error=%v", err)
		}
		if elapsed := time.Since(start); elapsed != 2*time.Second {
			t.Fatalf("wait=%v, want 2s", elapsed)
		}
	})
}

func TestReadinessExpiredLoadContextFailsAfterOnLoad(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	sess := testHarvestSession()
	if err := sess.runOnLoadBehavior(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("ON_LOAD error=%v", err)
	}
	if err := sess.waitForBrowserReadiness(ctx, time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
}

func TestReadinessShortMediaRetainsSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sess := testHarvestSession()
		observeBrowserRequest(sess, "finite-video", "Media")
		go func() { time.Sleep(200 * time.Millisecond); finishBrowserRequest(sess, "finite-video", 200) }()
		if err := sess.waitForBrowserReadiness(t.Context(), time.Second); err != nil {
			t.Fatal(err)
		}
	})
}

func TestReadinessDisabledMediaAndBackgroundDoNotExtendRequiredStall(t *testing.T) {
	for _, mediaAllowance := range []time.Duration{0, 5 * time.Second} {
		synctest.Test(t, func(t *testing.T) {
			sess := testHarvestSession()
			sess.networkTracker = newNetworkActivityTracker()
			sess.networkTracker.startMediaAllowance(mediaAllowance)
			observeBrowserRequest(sess, "script", "Script")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			go func() {
				for n := 0; ; n++ {
					select {
					case <-ctx.Done():
						return
					case <-time.After(100 * time.Millisecond):
						typ := "Fetch"
						if n%2 == 0 {
							typ = "Media"
						}
						id := fmt.Sprintf("background-%d", n)
						observeBrowserRequest(sess, id, typ)
						finishBrowserRequest(sess, id, 200)
					}
				}
			}()
			start := time.Now()
			if err := sess.waitForBrowserReadiness(ctx, time.Second); !errors.Is(err, errCompletionIdleTimeout) {
				t.Fatalf("error=%v", err)
			}
			if elapsed := time.Since(start); elapsed != 2*time.Second {
				t.Fatalf("required stall extended to %v", elapsed)
			}
			cancel()
			synctest.Wait()
		})
	}
}

func testHarvestSession() *Session {
	sess := newDefaultSession()
	sess.networkTracker.startMediaAllowance(sess.mediaSettleTimeout)
	return sess
}
func observeBrowserRequest(sess *Session, id, typ string) {
	sess.networkTracker.noteObservedRequestStart(network.ResourceType(typ), "https://example.com/"+id, true, network.RequestID(id))
}
func finishBrowserRequest(sess *Session, id string, _ int) {
	sess.networkTracker.noteRequestDone(network.RequestID(id))
}
