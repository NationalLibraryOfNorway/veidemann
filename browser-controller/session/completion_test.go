/*
 * Copyright 2020 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package session

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/requests"
)

func TestSignalCompletionActivityCoalesces(t *testing.T) {
	activity := make(chan struct{}, 1)

	signalCompletionActivity(nil)
	signalCompletionActivity(activity)
	signalCompletionActivity(activity)

	select {
	case <-activity:
	default:
		t.Fatal("completion activity was not signaled")
	}
	select {
	case <-activity:
		t.Fatal("duplicate completion activity was not coalesced")
	default:
	}
}

func TestSessionSignalCompletionActivityPreservesContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	sess := &Session{
		ctx:                ctx,
		completionActivity: make(chan struct{}, 1),
	}

	if err := sess.signalCompletionActivity(); !errors.Is(err, context.Canceled) {
		t.Fatalf("SignalCompletionActivity() error = %v, want %v", err, context.Canceled)
	}
	select {
	case <-sess.completionActivity:
		t.Fatal("canceled session signaled completion activity")
	default:
	}
}

func TestSessionRegisterResourceOnlySignalsBlockingRequests(t *testing.T) {
	sess := &Session{completionActivity: make(chan struct{}, 1)}
	sess.ObserveRequest(requests.Request{ID: "xhr", ResourceType: "XHR"})
	sess.ObserveRequest(requests.Request{ID: "document", ResourceType: "Document"})

	if _, err := sess.RegisterResource("xhr"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-sess.completionActivity:
		t.Fatal("non-blocking request signaled completion activity")
	default:
	}

	if _, err := sess.RegisterResource("document"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-sess.completionActivity:
	default:
		t.Fatal("blocking request did not signal completion activity")
	}
}

func TestSessionRejectResourceSignalsEveryFoundRequest(t *testing.T) {
	sess := &Session{completionActivity: make(chan struct{}, 1)}
	sess.ObserveRequest(requests.Request{ID: "xhr", ResourceType: "XHR"})

	found, err := sess.RejectResource("xhr")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("rejected request was not found")
	}
	select {
	case <-sess.completionActivity:
	default:
		t.Fatal("robots rejection did not signal completion activity")
	}
}

func TestSessionRecordInitialCacheHitAbortsBeforeCompletionActivity(t *testing.T) {
	abortStarted := make(chan struct{})
	releaseAbort := make(chan struct{})
	released := false
	defer func() {
		if !released {
			close(releaseAbort)
		}
	}()

	sess := &Session{
		completionActivity: make(chan struct{}, 1),
		frameLoads:         newFrameLoadTracker(),
		loadCancel: func() {
			close(abortStarted)
			<-releaseAbort
		},
	}
	sess.frameLoads.Start("root")
	sess.ObserveRequest(requests.Request{
		ID:           "document",
		ResourceType: "Document",
		GotNew:       true,
	})

	type completionResult struct {
		result ResourceCompletionResult
		err    error
	}
	resultCh := make(chan completionResult, 1)
	go func() {
		result, err := sess.RecordResourceCompletion("document", &logV1.CrawlLog{WarcId: "warc-1"}, true)
		resultCh <- completionResult{result: result, err: err}
	}()

	select {
	case <-abortStarted:
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}

	select {
	case <-sess.completionActivity:
		t.Fatal("completion activity was signaled before abort finished")
	default:
	}
	if waitErr := sess.frameLoads.Wait(t.Context()); !errors.Is(waitErr, errInitialRequestCached) {
		t.Fatalf("frame load state during abort = %v, want %v", waitErr, errInitialRequestCached)
	}

	close(releaseAbort)
	released = true
	var completion completionResult
	select {
	case completion = <-resultCh:
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
	if completion.err != nil {
		t.Fatal(completion.err)
	}
	if !completion.result.Found || !completion.result.InitialCacheHit {
		t.Fatalf("completion result = %#v, want initial cache hit", completion.result)
	}
	select {
	case <-sess.completionActivity:
	default:
		t.Fatal("blocking cached completion did not signal activity after abort")
	}
}

func TestSessionRecordCompletionClonesCrawlLog(t *testing.T) {
	sess := &Session{completionActivity: make(chan struct{}, 1)}
	sess.ObserveRequest(requests.Request{ID: "xhr", ResourceType: "XHR", GotNew: true})
	crawlLog := &logV1.CrawlLog{WarcId: "warc-1", Size: 10}

	result, err := sess.RecordResourceCompletion("xhr", crawlLog, false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Found || result.InitialCacheHit {
		t.Fatalf("completion result = %#v", result)
	}
	crawlLog.Size = 99

	snapshot, found := sess.RequestSnapshot("xhr")
	if !found || snapshot.CrawlLog.GetSize() != 10 || snapshot.CrawlLog == crawlLog {
		t.Fatalf("session retained caller-owned crawl log: %#v", snapshot.CrawlLog)
	}
	select {
	case <-sess.completionActivity:
		t.Fatal("non-blocking completion signaled activity")
	default:
	}
}

func TestSessionCompleteOptionsSelectsFirstUnregisteredURLMatch(t *testing.T) {
	sess := &Session{completionActivity: make(chan struct{}, 1)}
	const uri = "https://example.com/options"
	sess.ObserveRequest(requests.Request{ID: "registered", URL: uri, ResourceType: "Document", GotNew: true})
	sess.ObserveRequest(requests.Request{ID: "options", URL: uri, ResourceType: "Document"})

	found, err := sess.CompleteOptionsResource(uri)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("OPTIONS request was not found")
	}
	registered, _ := sess.RequestSnapshot("registered")
	options, _ := sess.RequestSnapshot("options")
	if registered.GotComplete || !options.GotComplete {
		t.Fatalf("unexpected URL match: registered=%#v options=%#v", registered, options)
	}
	select {
	case <-sess.completionActivity:
		t.Fatal("unregistered OPTIONS completion signaled blocking activity")
	default:
	}
}

func TestSessionInitializesRequestRegistryOnceConcurrently(t *testing.T) {
	sess := &Session{}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("request-%d", i)
			sess.ObserveRequest(requests.Request{ID: id, ResourceType: "Image"})
			if snapshot, found := sess.RequestSnapshot(id); !found || snapshot.ID != id {
				t.Errorf("request snapshot = %#v, found=%v", snapshot, found)
			}
		}(i)
	}
	wg.Wait()
}

func TestWaitForCompletionActivity(t *testing.T) {
	t.Run("activity", func(t *testing.T) {
		activity := make(chan struct{}, 1)
		signalCompletionActivity(activity)

		if err := waitForCompletionActivity(t.Context(), activity, time.Hour); err != nil {
			t.Fatalf("waitForCompletionActivity() error = %v, want nil", err)
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		if err := waitForCompletionActivity(ctx, nil, time.Hour); !errors.Is(err, context.Canceled) {
			t.Fatalf("waitForCompletionActivity() error = %v, want %v", err, context.Canceled)
		}
	})

	t.Run("idle timeout", func(t *testing.T) {
		if err := waitForCompletionActivity(t.Context(), nil, time.Millisecond); !errors.Is(err, errCompletionIdleTimeout) {
			t.Fatalf("waitForCompletionActivity() error = %v, want %v", err, errCompletionIdleTimeout)
		}
	})
}

func TestWaitForSettled(t *testing.T) {
	t.Run("already complete", func(t *testing.T) {
		networkWaits := 0
		err := waitForSettled(
			t.Context(),
			time.Hour,
			nil,
			func(context.Context) error {
				networkWaits++
				return nil
			},
			func() bool { return true },
		)
		if err != nil {
			t.Fatalf("waitForSettled() error = %v, want nil", err)
		}
		if networkWaits != 1 {
			t.Fatalf("network waits = %d, want 1", networkWaits)
		}
	})

	t.Run("activity triggers another settle check", func(t *testing.T) {
		activity := make(chan struct{}, 1)
		networkWaits := 0
		completionChecks := 0
		signalCompletionActivity(activity)

		err := waitForSettled(
			t.Context(),
			time.Hour,
			activity,
			func(context.Context) error {
				networkWaits++
				return nil
			},
			func() bool {
				completionChecks++
				return completionChecks == 2
			},
		)
		if err != nil {
			t.Fatalf("waitForSettled() error = %v, want nil", err)
		}
		if networkWaits != 2 {
			t.Fatalf("network waits = %d, want 2", networkWaits)
		}
	})

	t.Run("idle timeout is terminal", func(t *testing.T) {
		networkWaits := 0
		err := waitForSettled(
			t.Context(),
			time.Millisecond,
			nil,
			func(context.Context) error {
				networkWaits++
				return nil
			},
			func() bool { return false },
		)
		if !errors.Is(err, errCompletionIdleTimeout) {
			t.Fatalf("waitForSettled() error = %v, want %v", err, errCompletionIdleTimeout)
		}
		if networkWaits != 1 {
			t.Fatalf("network waits = %d, want 1", networkWaits)
		}
	})

	t.Run("network wait error", func(t *testing.T) {
		wantErr := errors.New("network wait failed")
		err := waitForSettled(
			t.Context(),
			time.Hour,
			nil,
			func(context.Context) error { return wantErr },
			func() bool {
				t.Fatal("request completion was checked after a network error")
				return false
			},
		)
		if !errors.Is(err, wantErr) {
			t.Fatalf("waitForSettled() error = %v, want %v", err, wantErr)
		}
	})

	t.Run("context deadline", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
		defer cancel()

		err := waitForSettled(
			ctx,
			time.Hour,
			nil,
			func(ctx context.Context) error { return ctx.Err() },
			func() bool { return false },
		)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("waitForSettled() error = %v, want %v", err, context.DeadlineExceeded)
		}
	})
}
