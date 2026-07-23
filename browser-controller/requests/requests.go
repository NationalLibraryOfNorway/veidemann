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

package requests

import (
	"fmt"
	"log/slog"
	"sync"
	"unicode/utf8"

	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/syncx"
)

type RequestRegistry interface {
	NotifyLoadStart()
	NotifyLoadFinished()
	AddRequest(req *Request)
	GetOrAddRequest(req *Request) (*Request, bool)
	RemoveRequest(req *Request) bool
	GetByID(id string) *Request
	GetByUrl(url string, onlyNew bool) *Request
	MatchCrawlLogs() bool
	GotNew(id string) *Request
	GotComplete(id string) *Request
	CompleteRequest(id string, crawlLog *logV1.CrawlLog, cached bool) *Request
	Walk(w func(*Request))
	InitialRequest() *Request
	RootRequest() *Request
	FinalizeResponses(requestedUrl *frontierV1.QueuedUri)
}

type requestRegistry struct {
	done *syncx.WaitGroup

	mu       sync.Mutex
	requests []*Request
	byID     map[string]*Request

	rootRequest  *Request
	lastMatchLog string
}

func (r *requestRegistry) InitialRequest() *Request {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.requests) == 0 {
		return nil
	}

	return r.requests[0]
}

func (r *requestRegistry) RootRequest() *Request {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.rootRequest
}

func NewRegistry(done *syncx.WaitGroup) RequestRegistry {
	return &requestRegistry{
		done: done,
		byID: make(map[string]*Request),
	}
}

func (r *requestRegistry) NotifyLoadStart() {
	r.done.Add(1)
}

func (r *requestRegistry) NotifyLoadFinished() {
	r.done.Done()
}

func (r *requestRegistry) AddRequest(req *Request) {
	if req == nil || req.ID == "" {
		panic("request must have canonical ID")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, req)
	r.byID[req.ID] = req
}

func (r *requestRegistry) GetByID(id string) *Request {
	if id == "" {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return r.byID[id]
}

func (r *requestRegistry) GetOrAddRequest(req *Request) (*Request, bool) {
	if req == nil || req.ID == "" {
		panic("request must have canonical ID")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.byID[req.ID]; ok {
		mergeRequest(existing, req)
		return existing, false
	}

	r.requests = append(r.requests, req)
	r.byID[req.ID] = req
	return req, true
}

func (r *requestRegistry) RemoveRequest(req *Request) bool {
	if req == nil {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byID[req.ID]; !ok {
		return false
	}

	delete(r.byID, req.ID)
	n := -1
	for i, c := range r.requests {
		if c.ID == req.ID {
			n = i
			break
		}
	}
	if n > -1 {
		r.requests[n] = r.requests[len(r.requests)-1]
		r.requests = r.requests[:len(r.requests)-1]
	}
	return true
}

func (r *requestRegistry) GetByUrl(url string, onlyNew bool) *Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, req := range r.requests {
		if req.URL == url {
			if !onlyNew || !req.GotNew {
				return req
			}
		}
	}
	return nil
}

func (r *requestRegistry) GotNew(id string) *Request {
	return r.mark(id, func(req *Request) {
		req.GotNew = true
	})
}

func (r *requestRegistry) GotComplete(id string) *Request {
	return r.mark(id, func(req *Request) {
		req.GotComplete = true
	})
}

func (r *requestRegistry) CompleteRequest(id string, crawlLog *logV1.CrawlLog, cached bool) *Request {
	return r.mark(id, func(req *Request) {
		req.CrawlLog = crawlLog
		req.GotComplete = true

		if cached {
			req.FromCache = true
		}
	})
}

func (r *requestRegistry) mark(id string, fn func(*Request)) *Request {
	if id == "" {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	req := r.byID[id]
	if req == nil {
		return nil
	}

	fn(req)

	return req
}

func (r *requestRegistry) MatchCrawlLogs() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := buildCrawlLogMatchSnapshot(r.requests)
	if signature := snapshot.signature(); signature != r.lastMatchLog {
		eventLog := slog.Default().With(
			"blockingRequests", snapshot.blockingCount,
			"resolvedRequests", snapshot.resolvedCount,
			"missingRequests", snapshot.unresolvedCount,
			"ignoredRequests", snapshot.ignoredCount,
		)
		if len(snapshot.missingRequests) > 0 {
			eventLog = eventLog.With("missingRequests", snapshot.missingRequests)
		}
		eventLog.Info("Match crawl")
		r.lastMatchLog = signature
	}
	return snapshot.unresolvedCount == 0
}

func (r *requestRegistry) Walk(w func(*Request)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, req := range r.requests {
		w(req)
	}
}

func (r *requestRegistry) FinalizeResponses(requestedUrl *frontierV1.QueuedUri) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.requests) == 0 {
		return
	}

	byURL := make(map[string]*Request, len(r.requests))

	for _, req := range r.requests {
		if req.URL != "" {
			byURL[req.URL] = req
		}
	}

	r.rootRequest = r.requests[0]

	for idx, rr := range r.requests {
		// Find the root request
		if rr.Redirected && rr.RedirectFromURL != "" {
			if parent := byURL[rr.RedirectFromURL]; parent != nil && parent != rr {
				rr.RedirectParent = parent

				if parent == r.rootRequest {
					r.rootRequest = rr
				}
			}
		}

		if rr.CrawlLog == nil {
			loggedURL, urlLength := boundedURLForLog(rr.URL)
			if !rr.BlocksPageCompletion() {
				slog.Info("Skipping missing crawlLog for non-blocking request",
					"id", rr.ID,
					"url", loggedURL,
					"urlLength", urlLength,
					"fetchRequestId", rr.FetchRequestID,
					"networkId", rr.NetworkID,
					"resourceType", rr.ResourceType,
					"fromCache", rr.FromCache,
				)
			} else {
				slog.Warn("Missing crawlLog",
					"id", rr.ID,
					"url", loggedURL,
					"urlLength", urlLength,
					"index", idx,
					"fetchRequestId", rr.FetchRequestID,
					"networkId", rr.NetworkID,
					"new", rr.GotNew,
					"complete", rr.GotComplete,
					"fromCache", rr.FromCache,
				)
			}
			continue
		}

		rr.CrawlLog.Referrer = rr.Referrer

		referrerRequest := byURL[rr.Referrer]

		discoveryType := discoveryTypeForRequest(idx, rr, requestedUrl)

		switch {
		case rr.RedirectParent != nil && rr.RedirectParent.CrawlLog != nil:
			rr.CrawlLog.DiscoveryPath = rr.RedirectParent.CrawlLog.DiscoveryPath + discoveryType

		case referrerRequest != nil && referrerRequest.CrawlLog != nil:
			rr.CrawlLog.DiscoveryPath = referrerRequest.CrawlLog.DiscoveryPath + discoveryType

		default:
			rr.CrawlLog.DiscoveryPath = discoveryType
		}
	}
}

func discoveryTypeForRequest(idx int, req *Request, requestedUrl *frontierV1.QueuedUri) string {
	if idx == 0 {
		return requestedUrl.DiscoveryPath
	}

	if req.Initiator == "script" {
		return "X"
	}

	if req.RedirectParent != nil || req.Redirected {
		return "R"
	}

	return "E"
}

const maxLoggedMissingRequestIDs = 8
const maxLoggedURLBytes = 512

func boundedURLForLog(rawURL string) (string, int) {
	originalLength := len(rawURL)
	if originalLength <= maxLoggedURLBytes {
		return rawURL, originalLength
	}

	const ellipsis = "…"
	end := maxLoggedURLBytes - len(ellipsis)
	for end > 0 && !utf8.RuneStart(rawURL[end]) {
		end--
	}
	return rawURL[:end] + ellipsis, originalLength
}

type crawlLogMatchSnapshot struct {
	blockingCount   int
	resolvedCount   int
	unresolvedCount int
	ignoredCount    int
	missingRequests []missingRequestSummary
}

type missingRequestSummary struct {
	ID             string
	FetchRequestID string
	NetworkID      string
	URL            string
	URLLength      int
	ResourceType   string
	GotNew         bool
	GotComplete    bool
	FromCache      bool
}

func (s crawlLogMatchSnapshot) signature() string {
	return fmt.Sprintf(
		"%d|%d|%d|%d|%v",
		s.blockingCount,
		s.resolvedCount,
		s.unresolvedCount,
		s.ignoredCount,
		s.missingRequests,
	)
}

func buildCrawlLogMatchSnapshot(requests []*Request) crawlLogMatchSnapshot {
	snapshot := crawlLogMatchSnapshot{}

	for _, req := range requests {
		if !req.BlocksPageCompletion() {
			snapshot.ignoredCount++
			continue
		}

		snapshot.blockingCount++

		if req.CrawlLog == nil {
			snapshot.unresolvedCount++

			if len(snapshot.missingRequests) < maxLoggedMissingRequestIDs {
				loggedURL, urlLength := boundedURLForLog(req.URL)
				snapshot.missingRequests = append(snapshot.missingRequests, missingRequestSummary{
					ID:             req.ID,
					FetchRequestID: req.FetchRequestID,
					NetworkID:      req.NetworkID,
					URL:            loggedURL,
					URLLength:      urlLength,
					ResourceType:   req.ResourceType,
					GotNew:         req.GotNew,
					GotComplete:    req.GotComplete,
					FromCache:      req.FromCache,
				})
			}

			continue
		}

		snapshot.resolvedCount++
	}

	return snapshot
}
