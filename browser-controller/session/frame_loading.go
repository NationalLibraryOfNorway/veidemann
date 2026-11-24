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
	"sync"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
)

// pageLoadSignal remembers an early load event until Navigate returns its loader.
// Only the root document's initial load gates ON_LOAD scripts. Committed identity
// is kept separately from the Navigate reply used to establish that load gate.
type pageLoadSignal struct {
	mu              sync.Mutex
	frame           cdp.FrameID
	committedURL    string
	committedFrame  cdp.FrameID
	committedLoader cdp.LoaderID
	loader, loaded  cdp.LoaderID
	done            chan struct{}
	signaled        bool
}

func newPageLoadSignal() *pageLoadSignal { return &pageLoadSignal{done: make(chan struct{})} }

func (s *pageLoadSignal) start(frame cdp.FrameID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.frame = frame
}

func (s *pageLoadSignal) selectNavigation(frame cdp.FrameID, loader cdp.LoaderID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.frame = frame
	// A committed navigation may already have redirected again before the
	// Navigate reply arrives. The ordered frame event owns the current loader.
	if s.loader == "" {
		s.loader = loader
	}
	s.signal()
}

func (s *pageLoadSignal) navigated(frame *cdp.Frame) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if frame.ParentID == "" && frame.ID == s.frame {
		s.committedURL = frame.URL
		s.committedFrame = frame.ID
		s.committedLoader = frame.LoaderID
		s.loader = frame.LoaderID
		s.signal()
	}
}

// documentURL only associates capture with an observed root document commit.
// Same-document navigation events must never update this network identity.
func (s *pageLoadSignal) documentURL(frame *cdp.Frame) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if frame.ID == "" || frame.ID != s.committedFrame || frame.LoaderID == "" || frame.LoaderID != s.committedLoader || s.committedURL == "" {
		return "", false
	}
	return s.committedURL, true
}

func (s *pageLoadSignal) observe(event *page.EventLifecycleEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if event.FrameID != s.frame || event.Name != "load" {
		return
	}
	s.loaded = event.LoaderID
	s.signal()
}

func (s *pageLoadSignal) signal() {
	if !s.signaled && s.loader != "" && s.loaded == s.loader {
		s.signaled = true
		close(s.done)
	}
}

func (sess *Session) waitForPageLoad(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-sess.pageLoad.done:
		return nil
	}
}

func (s *pageLoadSignal) isRoot(frame cdp.FrameID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return frame != "" && frame == s.frame
}
