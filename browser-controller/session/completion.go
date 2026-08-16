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
	"time"
)

var errCompletionIdleTimeout = errors.New("completion activity idle timeout")

// signalCompletionActivity records that completion state may have changed.
// The signal is level-triggered: one pending wake-up is enough because the
// waiter always rechecks authoritative network and request state.
func signalCompletionActivity(activity chan<- struct{}) {
	if activity == nil {
		return
	}

	select {
	case activity <- struct{}{}:
	default:
	}
}

func waitForCompletionActivity(ctx context.Context, activity <-chan struct{}, maxIdleTime time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	idleTimer := time.NewTimer(maxIdleTime)
	defer idleTimer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-activity:
		return nil
	case <-idleTimer.C:
		return errCompletionIdleTimeout
	}
}

func waitForSettled(
	ctx context.Context,
	maxIdleTime time.Duration,
	activity <-chan struct{},
	waitForNetworkIdle func(context.Context) error,
	requestsComplete func() bool,
) error {
	// ctx is the sole overall deadline. maxIdleTime only bounds how long an
	// incomplete request set may remain unchanged between state checks.
	for {
		if err := waitForNetworkIdle(ctx); err != nil {
			return err
		}
		if requestsComplete() {
			return nil
		}
		if err := waitForCompletionActivity(ctx, activity, maxIdleTime); err != nil {
			return err
		}
	}
}
