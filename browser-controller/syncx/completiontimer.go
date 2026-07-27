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
package syncx

import (
	"errors"
	"sync"
	"time"
)

var ErrIdleTimeout = errors.New("idle timeout")
var ErrExceededMaxTime = errors.New("exceeded max time")
var ErrCancelled = errors.New("cancelled")

type CompletionTimer struct {
	maxIdleTime  time.Duration
	maxTotalTime time.Duration
	check        func() bool

	mu          sync.Mutex
	waiting     bool
	cancelled   bool
	notifyCount int32

	notifyCh chan struct{}
	cancelCh chan struct{}
}

func NewCompletionTimer(maxIdleTime, maxTotalTime time.Duration, check func() bool) *CompletionTimer {
	if check == nil {
		check = func() bool { return false }
	}

	return &CompletionTimer{
		maxIdleTime:  maxIdleTime,
		maxTotalTime: maxTotalTime,
		check:        check,
		notifyCh:     make(chan struct{}, 1),
		cancelCh:     make(chan struct{}),
	}
}

func (t *CompletionTimer) Notify() {
	t.mu.Lock()
	t.notifyCount++
	t.mu.Unlock()

	select {
	case t.notifyCh <- struct{}{}:
	default:
	}
}

func (t *CompletionTimer) WaitForCompletion() error {
	cancelCh, err := t.beginWait()
	if err != nil {
		return err
	}
	defer t.endWait()

	if t.check() {
		return nil
	}

	totalTimer := time.NewTimer(t.maxTotalTime)
	defer totalTimer.Stop()

	idleTimer := time.NewTimer(t.maxIdleTime)
	defer idleTimer.Stop()

	for {
		select {
		case <-t.notifyCh:
			if t.check() {
				return nil
			}

			idleTimer.Stop()
			idleTimer.Reset(t.maxIdleTime)

		case <-idleTimer.C:
			if t.check() {
				return nil
			}
			return ErrIdleTimeout

		case <-totalTimer.C:
			if t.check() {
				return nil
			}
			return ErrExceededMaxTime

		case <-cancelCh:
			return ErrCancelled
		}
	}
}

func (t *CompletionTimer) Cancel() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cancelled {
		return
	}

	t.cancelled = true
	close(t.cancelCh)
}

// Reset prepares the timer for another WaitForCompletion call.
// It returns the number of notifications received since the previous reset.
func (t *CompletionTimer) Reset() int32 {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.waiting {
		panic("CompletionTimer: Reset during WaitForCompletion")
	}

	count := t.notifyCount
	t.notifyCount = 0

	t.cancelled = false
	t.cancelCh = make(chan struct{})

	drain(t.notifyCh)

	return count
}

func (t *CompletionTimer) beginWait() (<-chan struct{}, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.waiting {
		panic("CompletionTimer: concurrent WaitForCompletion")
	}

	if t.cancelled {
		return nil, ErrCancelled
	}

	t.waiting = true
	return t.cancelCh, nil
}

func (t *CompletionTimer) endWait() {
	t.mu.Lock()
	t.waiting = false
	t.mu.Unlock()
}

func drain[T any](ch <-chan T) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
