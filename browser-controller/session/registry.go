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
	"fmt"
	"sync"

	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/metrics"
)

type Registry struct {
	sessions []*Session
	opts     []Option
	pool     chan int
	mu       sync.Mutex
	wg       sync.WaitGroup
}

func NewRegistry(maxSessions int, opts ...Option) (sr *Registry) {
	sr = &Registry{
		sessions: make([]*Session, maxSessions),
		pool:     make(chan int, maxSessions-1),
		opts:     opts,
	}
	for i := 1; i < maxSessions; i++ {
		sr.pool <- i
	}
	metrics.BrowserSessions.Set(float64(maxSessions))
	return
}

// GetNextAvailable returns next session from the pool.
func (sr *Registry) GetNextAvailable(ctx context.Context) (*Session, error) {
	var i int

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case i = <-sr.pool:
		sr.wg.Add(1)
	}

	sess := newSession(i, sr.opts...)

	sr.mu.Lock()
	sr.sessions[i] = sess
	sr.mu.Unlock()

	return sess, nil
}

func (sr *Registry) Get(sessId int) *Session {
	if sessId < 0 || sessId >= len(sr.sessions) {
		panic(fmt.Sprintf("BUG: session registry (Get): session id is out of slice range: %d", sessId))
	}
	sr.mu.Lock()
	defer sr.mu.Unlock()
	s := sr.sessions[sessId]
	return s
}

// GetActive returns the session for sessId when it is ready to accept resource
// RPCs. Invalid, unallocated, initializing, and draining sessions are reported
// as unavailable.
func (sr *Registry) GetActive(sessId int) *Session {
	if sessId < 0 || sessId >= len(sr.sessions) {
		return nil
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()

	sess := sr.sessions[sessId]
	if sess == nil || !sess.acceptingRequests() {
		return nil
	}
	return sess
}

func (sr *Registry) Release(sess *Session) {
	if sess.Id < 0 || sess.Id >= len(sr.sessions) {
		panic(fmt.Sprintf("BUG: session registry (Release): session id is out of slice range: %d", sess.Id))
	}
	sess.stopAcceptingRequests()
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.sessions[sess.Id] = nil
	sr.pool <- sess.Id
	sr.wg.Done()
}

func (sr *Registry) Close() {
	sr.wg.Wait()
}
