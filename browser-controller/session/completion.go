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

// readinessSnapshot contains authoritative timestamps. Coalesced notifications
// cannot extend either the quiet window or the media allowance on their own.
type readinessSnapshot struct {
	requiredLast, mediaLast, mediaDeadline time.Time
	requiredPending                        int
}

func waitForReadiness(
	ctx context.Context, quiet, additionalGrace time.Duration,
	networkUpdates <-chan struct{}, waitPending bool, snapshot func() readinessSnapshot,
) (string, error) {
	start := time.Now()
	for {
		if err := ctx.Err(); err != nil {
			if errors.Is(err, context.Canceled) {
				return "canceled", err
			}
			return "deadline", err
		}
		state := snapshot()
		pending := waitPending && state.requiredPending != 0
		now := time.Now()
		requiredSince := state.requiredLast
		if requiredSince.Before(start) {
			requiredSince = start
		}
		quietSince := requiredSince
		mediaRelevant := !state.mediaDeadline.IsZero() && now.Before(state.mediaDeadline)
		if mediaRelevant && state.mediaLast.After(quietSince) {
			quietSince = state.mediaLast
		}
		quietDeadline := quietSince.Add(quiet)
		if !pending && !now.Before(quietDeadline) {
			reason := "quiet"
			if !state.mediaDeadline.IsZero() && !mediaRelevant && state.mediaLast.After(requiredSince) {
				reason = "media_allowance_elapsed"
			}
			return reason, nil
		}
		next := quietDeadline
		if pending {
			stallDeadline := requiredSince.Add(quiet + additionalGrace)
			if !now.Before(stallDeadline) {
				return "required_request_stalled", errCompletionIdleTimeout
			}
			next = stallDeadline
		}
		if mediaRelevant && state.mediaDeadline.Before(next) {
			next = state.mediaDeadline
		}
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-networkUpdates:
			timer.Stop()
		case <-timer.C:
		}
	}
}
