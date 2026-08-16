package session

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestFrameLoadTrackerStartFinishAndSnapshot(t *testing.T) {
	tracker := newFrameLoadTracker()

	if tracker.Start("") {
		t.Fatal("empty start was tracked")
	}
	if !tracker.Start("frame-b") {
		t.Fatal("first start was not tracked")
	}
	if !tracker.Start("frame-a") {
		t.Fatal("second frame start was not tracked")
	}
	if tracker.Start("frame-b") {
		t.Fatal("duplicate start was tracked")
	}
	if snapshot := tracker.Snapshot(); !slices.Equal(snapshot, []string{"frame-a", "frame-b"}) {
		t.Fatalf("snapshot = %v, want [frame-a frame-b]", snapshot)
	}

	if tracker.Finish("") {
		t.Fatal("empty finish was tracked")
	}
	if tracker.Finish("missing-frame") {
		t.Fatal("finish without start was tracked")
	}
	if !tracker.Finish("frame-a") {
		t.Fatal("finish after start was not tracked")
	}
	if tracker.Finish("frame-a") {
		t.Fatal("duplicate finish was tracked")
	}
	if snapshot := tracker.Snapshot(); !slices.Equal(snapshot, []string{"frame-b"}) {
		t.Fatalf("snapshot after finish = %v, want [frame-b]", snapshot)
	}
}

func TestFrameLoadTrackerAcceptsSequentialLoadsBeforeWait(t *testing.T) {
	tracker := newFrameLoadTracker()

	for cycle := range 2 {
		if !tracker.Start("frame-1") {
			t.Fatalf("cycle %d start was not tracked", cycle+1)
		}
		if !tracker.Finish("frame-1") {
			t.Fatalf("cycle %d finish was not tracked", cycle+1)
		}
	}

	if err := tracker.Wait(t.Context()); err != nil {
		t.Fatalf("Wait() error = %v, want nil", err)
	}
}

func TestFrameLoadTrackerWaitsForEveryActiveFrame(t *testing.T) {
	tracker := newFrameLoadTracker()
	tracker.Start("frame-1")
	waitResult := startFrameLoadWait(t, tracker, t.Context())

	tracker.Start("frame-2")
	if !tracker.Finish("frame-1") {
		t.Fatal("first frame finish was not tracked")
	}
	if trackerFinished(tracker) {
		t.Fatal("tracker completed while another frame was active")
	}
	if !tracker.Finish("frame-2") {
		t.Fatal("second frame finish was not tracked")
	}
	if err := receiveWaitResult(t, waitResult); err != nil {
		t.Fatalf("Wait() error = %v, want nil", err)
	}
}

func TestFrameLoadTrackerIgnoresStartAfterCompletion(t *testing.T) {
	tracker := newFrameLoadTracker()
	tracker.Start("frame-1")
	waitResult := startFrameLoadWait(t, tracker, t.Context())

	if !tracker.Finish("frame-1") {
		t.Fatal("last frame finish was not tracked")
	}
	if tracker.Start("frame-2") {
		t.Fatal("late start was tracked")
	}
	if tracker.Finish("frame-2") {
		t.Fatal("late finish was tracked")
	}
	if err := receiveWaitResult(t, waitResult); err != nil {
		t.Fatalf("Wait() error = %v, want nil", err)
	}

	tracker.Cancel(errInitialRequestCached)
	if err := tracker.Wait(t.Context()); err != nil {
		t.Fatalf("terminal result changed after Cancel(): %v", err)
	}
}

func TestFrameLoadTrackerConcurrentFinalFinishAndStart(t *testing.T) {
	for range 100 {
		tracker := newFrameLoadTracker()
		tracker.Start("frame-1")
		waitResult := startFrameLoadWait(t, tracker, t.Context())

		start := make(chan struct{})
		var actions sync.WaitGroup
		actions.Add(2)
		go func() {
			defer actions.Done()
			<-start
			tracker.Finish("frame-1")
		}()
		go func() {
			defer actions.Done()
			<-start
			tracker.Start("frame-2")
		}()
		close(start)
		actions.Wait()

		// If the new start won the lock, it extended the wait. If the final
		// finish won, this is a harmless no-op against a terminal tracker.
		tracker.Finish("frame-2")
		if err := receiveWaitResult(t, waitResult); err != nil {
			t.Fatalf("Wait() error = %v, want nil", err)
		}
	}
}

func TestFrameLoadTrackerWaitWithNoFramesIsTerminal(t *testing.T) {
	tracker := newFrameLoadTracker()

	if err := tracker.Wait(t.Context()); err != nil {
		t.Fatalf("Wait() error = %v, want nil", err)
	}
	if tracker.Start("late-frame") {
		t.Fatal("start after empty completion was tracked")
	}
	if snapshot := tracker.Snapshot(); len(snapshot) != 0 {
		t.Fatalf("terminal snapshot = %v, want empty", snapshot)
	}
}

func TestFrameLoadTrackerContextErrorsAreTerminal(t *testing.T) {
	t.Run("deadline", func(t *testing.T) {
		tracker := newFrameLoadTracker()
		tracker.Start("frame-1")
		ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
		defer cancel()

		if err := tracker.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Wait() error = %v, want %v", err, context.DeadlineExceeded)
		}
		if snapshot := tracker.Snapshot(); !slices.Equal(snapshot, []string{"frame-1"}) {
			t.Fatalf("timeout snapshot = %v, want [frame-1]", snapshot)
		}
		if tracker.Finish("frame-1") {
			t.Fatal("finish after timeout was tracked")
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		tracker := newFrameLoadTracker()
		tracker.Start("frame-1")
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		if err := tracker.Wait(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("Wait() error = %v, want %v", err, context.Canceled)
		}
	})
}

func TestFrameLoadTrackerExplicitCacheAbort(t *testing.T) {
	tracker := newFrameLoadTracker()
	tracker.Start("frame-1")
	waitResult := startFrameLoadWait(t, tracker, t.Context())

	tracker.Cancel(errInitialRequestCached)
	tracker.Cancel(errors.New("later cancellation must not replace the result"))
	if err := receiveWaitResult(t, waitResult); !errors.Is(err, errInitialRequestCached) {
		t.Fatalf("Wait() error = %v, want %v", err, errInitialRequestCached)
	}
	if tracker.Start("frame-2") {
		t.Fatal("start after cancellation was tracked")
	}
	if err := tracker.Wait(t.Context()); !errors.Is(err, errInitialRequestCached) {
		t.Fatalf("second Wait() error = %v, want %v", err, errInitialRequestCached)
	}
}

func startFrameLoadWait(t *testing.T, tracker *frameLoadTracker, ctx context.Context) <-chan error {
	t.Helper()

	result := make(chan error, 1)
	go func() {
		result <- tracker.Wait(ctx)
	}()

	deadline := time.Now().Add(time.Second)
	for {
		tracker.mu.Lock()
		waiting := tracker.waiting
		tracker.mu.Unlock()
		if waiting {
			return result
		}
		if time.Now().After(deadline) {
			t.Fatal("tracker did not begin waiting")
		}
		runtime.Gosched()
	}
}

func receiveWaitResult(t *testing.T, result <-chan error) error {
	t.Helper()

	select {
	case err := <-result:
		return err
	case <-time.After(time.Second):
		t.Fatal("tracker did not finish waiting")
		return nil
	}
}

func trackerFinished(tracker *frameLoadTracker) bool {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	return tracker.finished
}
