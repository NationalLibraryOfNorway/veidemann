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
	"google.golang.org/protobuf/proto"
)

type ResponseSnapshot struct {
	InitialRequest  *Request
	RootRequest     *Request
	Requests        []*Request
	BlockingCount   int
	ResolvedCount   int
	UnresolvedCount int
	IgnoredCount    int
}

type Registry struct {
	log *slog.Logger

	mu       sync.Mutex
	requests []*Request
	byID     map[string]*Request

	lastMatchLog string
}

// TODO(follow-up): GetOrAddRequest, GetByUrl, InitialRequest, GotNew,
// GotComplete, and CompleteRequest return live *Request values after releasing
// mu. CDP listeners and resource RPC goroutines can then read or mutate the same
// fields, including CrawlLog, without the registry lock. Move mutations and
// predicates behind operation-specific methods and return immutable snapshots.

func (r *Registry) InitialRequest() *Request {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.requests) == 0 {
		return nil
	}

	return r.requests[0]
}

func (r *Registry) RootRequestSnapshot() *Request {
	r.mu.Lock()
	defer r.mu.Unlock()

	return cloneRequest(resolveRootRequest(r.requests, false))
}

func NewRegistry(logger *slog.Logger) *Registry {
	if logger == nil {
		logger = slog.Default()
	}
	return &Registry{
		log:  logger,
		byID: make(map[string]*Request),
	}
}

func (r *Registry) AddRequest(req *Request) {
	if req == nil || req.ID == "" {
		panic("request must have canonical ID")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, req)
	r.byID[req.ID] = req
}

func (r *Registry) GetOrAddRequest(req *Request) (*Request, bool) {
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

func (r *Registry) RemoveRequest(req *Request) bool {
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

func (r *Registry) GetByUrl(url string, onlyNew bool) *Request {
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

func (r *Registry) GotNew(id string) *Request {
	return r.mark(id, func(req *Request) {
		req.GotNew = true
	})
}

func (r *Registry) GotComplete(id string) *Request {
	return r.mark(id, func(req *Request) {
		req.GotComplete = true
	})
}

func (r *Registry) CompleteRequest(id string, crawlLog *logV1.CrawlLog, cached bool) *Request {
	return r.mark(id, func(req *Request) {
		req.CrawlLog = crawlLog
		req.GotComplete = true

		if cached {
			req.FromCache = true
		}
	})
}

func (r *Registry) mark(id string, fn func(*Request)) *Request {
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

func (r *Registry) MatchCrawlLogs() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := buildCrawlLogMatchSnapshot(r.requests)
	if signature := snapshot.signature(); signature != r.lastMatchLog {
		eventLog := r.log.With(
			"blockingRequests", snapshot.blockingCount,
			"resolvedRequests", snapshot.resolvedCount,
			"missingRequests", snapshot.unresolvedCount,
			"ignoredRequests", snapshot.ignoredCount,
		)
		if len(snapshot.missingRequests) > 0 {
			eventLog = eventLog.With("missingRequests", snapshot.missingRequests)
		}
		eventLog.Debug("Match crawl")
		r.lastMatchLog = signature
	}
	return snapshot.unresolvedCount == 0
}

func (r *Registry) FinalizeResponses(requestedUrl *frontierV1.QueuedUri) *ResponseSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := r.snapshotLocked()
	byURL := requestsByURL(snapshot.Requests)

	for idx, rr := range snapshot.Requests {
		resourceLog := r.log.With(resourceLogAttrs(rr)...)
		if rr.CrawlLog == nil {
			if rr.BlocksPageCompletion() {
				resourceLog.Warn("Missing crawlLog", "index", idx)
			} else {
				resourceLog.Debug("Resource finalized")
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
		if rr.CrawlLog.GetWarcId() == "" {
			resourceLog.Warn("Crawl log in registry without WARC ID")
		} else {
			resourceLog.Debug("Resource finalized")
		}
	}

	return snapshot
}

func (r *Registry) snapshotLocked() *ResponseSnapshot {
	snapshot := &ResponseSnapshot{}
	if len(r.requests) == 0 {
		return snapshot
	}

	snapshot.Requests = make([]*Request, len(r.requests))
	for idx, req := range r.requests {
		snapshot.Requests[idx] = cloneRequest(req)
	}
	snapshot.InitialRequest = snapshot.Requests[0]
	snapshot.RootRequest = resolveRootRequest(snapshot.Requests, true)
	match := buildCrawlLogMatchSnapshot(snapshot.Requests)
	snapshot.BlockingCount = match.blockingCount
	snapshot.ResolvedCount = match.resolvedCount
	snapshot.UnresolvedCount = match.unresolvedCount
	snapshot.IgnoredCount = match.ignoredCount
	return snapshot
}

func resolveRootRequest(requests []*Request, linkParents bool) *Request {
	if len(requests) == 0 {
		return nil
	}

	rootRequest := requests[0]
	byURL := requestsByURL(requests)
	for _, req := range requests {
		if !req.Redirected || req.RedirectFromURL == "" {
			continue
		}
		parent := byURL[req.RedirectFromURL]
		if parent == nil || parent == req {
			continue
		}
		if linkParents {
			req.RedirectParent = parent
		}
		if parent == rootRequest {
			rootRequest = req
		}
	}
	return rootRequest
}

func cloneRequest(req *Request) *Request {
	if req == nil {
		return nil
	}

	clone := *req
	clone.RedirectParent = nil
	if req.CrawlLog != nil {
		clone.CrawlLog = proto.Clone(req.CrawlLog).(*logV1.CrawlLog)
	}
	return &clone
}

func requestsByURL(requests []*Request) map[string]*Request {
	byURL := make(map[string]*Request, len(requests))
	for _, req := range requests {
		if req.URL != "" {
			byURL[req.URL] = req
		}
	}
	return byURL
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

// BoundedURLForLog truncates a URL for logging and returns its original byte length.
func BoundedURLForLog(rawURL string) (string, int) {
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

func resourceLogAttrs(req *Request) []any {
	loggedURL, urlLength := BoundedURLForLog(req.URL)
	redirectFromURL, redirectFromURLLength := BoundedURLForLog(req.RedirectFromURL)
	attrs := []any{
		"requestId", req.ID,
		"networkId", req.NetworkID,
		"fetchRequestId", req.FetchRequestID,
		"url", loggedURL,
		"urlLength", urlLength,
		"method", req.Method,
		"resourceType", req.ResourceType,
		"initiator", req.Initiator,
		"fromCache", req.FromCache,
		"redirected", req.Redirected,
		"redirectFromUrl", redirectFromURL,
		"redirectFromUrlLength", redirectFromURLLength,
		"gotNew", req.GotNew,
		"gotComplete", req.GotComplete,
		"hasCrawlLog", req.CrawlLog != nil,
	}
	if req.CrawlLog != nil {
		attrs = append(attrs,
			"statusCode", req.CrawlLog.GetStatusCode(),
			"crawlLogError", req.CrawlLog.GetError(),
			"warcId", req.CrawlLog.GetWarcId(),
		)
	}
	return attrs
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
				loggedURL, urlLength := BoundedURLForLog(req.URL)
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
