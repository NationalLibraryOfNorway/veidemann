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
	"context"
	"sync"
)

type WaitGroup struct {
	ctx context.Context

	wg       sync.WaitGroup
	doneOnce sync.Once
	done     chan struct{}

	cancelOnce sync.Once
	cancel     chan struct{}
}

func NewWaitGroup(ctx context.Context) *WaitGroup {
	if ctx == nil {
		ctx = context.Background()
	}

	return &WaitGroup{
		ctx:    ctx,
		done:   make(chan struct{}),
		cancel: make(chan struct{}),
	}
}

func (wg *WaitGroup) Add(delta int) {
	wg.wg.Add(delta)
}

func (wg *WaitGroup) Done() {
	wg.wg.Done()
}

func (wg *WaitGroup) Wait() error {
	wg.doneOnce.Do(func() {
		go func() {
			wg.wg.Wait()
			close(wg.done)
		}()
	})

	select {
	case <-wg.done:
		return nil
	case <-wg.ctx.Done():
		return ErrExceededMaxTime
	case <-wg.cancel:
		return ErrCancelled
	}
}

func (wg *WaitGroup) Cancel() {
	wg.cancelOnce.Do(func() {
		close(wg.cancel)
	})
}
