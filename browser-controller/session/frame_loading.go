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
	"slices"
	"sync"
)

// frameLoadTracker tracks the one-shot initial page-load phase.
//
// Starts are accepted until Wait observes that no frames are loading, the
// context expires, or Cancel is called. The first of those outcomes is
// terminal; later frame events are ignored.
type frameLoadTracker struct {
	mu sync.Mutex

	loading  map[string]struct{}
	waiting  bool
	finished bool
	result   error
	done     chan struct{}
}

func newFrameLoadTracker() *frameLoadTracker {
	return &frameLoadTracker{
		loading: make(map[string]struct{}),
		done:    make(chan struct{}),
	}
}

func (t *frameLoadTracker) Start(frameID string) bool {
	if frameID == "" {
		return false
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.finished {
		return false
	}
	if _, ok := t.loading[frameID]; ok {
		return false
	}

	t.loading[frameID] = struct{}{}
	return true
}

func (t *frameLoadTracker) Finish(frameID string) bool {
	if frameID == "" {
		return false
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.finished {
		return false
	}
	if _, ok := t.loading[frameID]; !ok {
		return false
	}

	delete(t.loading, frameID)
	if t.waiting && len(t.loading) == 0 {
		_ = t.finishLocked(nil)
	}
	return true
}

func (t *frameLoadTracker) Wait(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	t.mu.Lock()
	if t.finished {
		result := t.result
		t.mu.Unlock()
		return result
	}
	if err := ctx.Err(); err != nil {
		result := t.finishLocked(err)
		t.mu.Unlock()
		return result
	}

	t.waiting = true
	if len(t.loading) == 0 {
		result := t.finishLocked(nil)
		t.mu.Unlock()
		return result
	}
	done := t.done
	t.mu.Unlock()

	select {
	case <-done:
		return t.resultValue()
	case <-ctx.Done():
		return t.finish(ctx.Err())
	}
}

func (t *frameLoadTracker) Cancel(err error) {
	if err == nil {
		err = context.Canceled
	}
	_ = t.finish(err)
}

func (t *frameLoadTracker) Snapshot() []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.loading) == 0 {
		return nil
	}

	snapshot := make([]string, 0, len(t.loading))
	for frameID := range t.loading {
		snapshot = append(snapshot, frameID)
	}
	slices.Sort(snapshot)
	return snapshot
}

func (t *frameLoadTracker) finish(result error) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.finishLocked(result)
}

func (t *frameLoadTracker) finishLocked(result error) error {
	if t.finished {
		return t.result
	}

	t.finished = true
	t.result = result
	close(t.done)
	return result
}

func (t *frameLoadTracker) resultValue() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.result
}
