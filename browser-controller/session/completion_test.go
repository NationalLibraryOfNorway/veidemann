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
	"testing"
	"time"

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

	if err := sess.SignalCompletionActivity(); !errors.Is(err, context.Canceled) {
		t.Fatalf("SignalCompletionActivity() error = %v, want %v", err, context.Canceled)
	}
	select {
	case <-sess.completionActivity:
		t.Fatal("canceled session signaled completion activity")
	default:
	}
}

func TestSessionSignalRequestActivityOnlySignalsBlockingRequests(t *testing.T) {
	sess := &Session{completionActivity: make(chan struct{}, 1)}

	if err := sess.SignalRequestActivity(&requests.Request{GotNew: true, ResourceType: "XHR"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-sess.completionActivity:
		t.Fatal("non-blocking request signaled completion activity")
	default:
	}

	if err := sess.SignalRequestActivity(&requests.Request{GotNew: true, ResourceType: "Document"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-sess.completionActivity:
	default:
		t.Fatal("blocking request did not signal completion activity")
	}
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
