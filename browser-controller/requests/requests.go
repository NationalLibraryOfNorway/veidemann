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

	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/syncx"
)

type RequestRegistry interface {
	NotifyLoadStart()
	NotifyLoadFinished()
	AddRequest(req *Request)
	GetOrAddRequest(req *Request) (*Request, bool)
	RemoveRequest(req *Request) bool
	GetByNetworkId(id string) *Request
	GetByRequestId(id string) *Request
	GetByUrl(url string, onlyNew bool) *Request
	MatchCrawlLogs() bool
	Walk(w func(*Request))
	InitialRequest() *Request
	RootRequest() *Request
	FinalizeResponses(requestedUrl *frontierV1.QueuedUri)
}

type requestRegistry struct {
	done         *syncx.WaitGroup
	mu           sync.Mutex
	requests     []*Request
	rootRequest  *Request
	lastMatchLog string
}

func (r *requestRegistry) InitialRequest() *Request {
	return r.requests[0]
}

func (r *requestRegistry) RootRequest() *Request {
	return r.rootRequest
}

func NewRegistry(done *syncx.WaitGroup) RequestRegistry {
	r := &requestRegistry{
		done: done,
	}

	return r
}

func (r *requestRegistry) NotifyLoadStart() {
	r.done.Add(1)
}

func (r *requestRegistry) NotifyLoadFinished() {
	r.done.Done()
}

func (r *requestRegistry) AddRequest(req *Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, req)
}

func (r *requestRegistry) GetOrAddRequest(req *Request) (*Request, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.requests {
		if req.RequestId != "" && existing.RequestId == req.RequestId {
			return existing, false
		}
		if req.NetworkId != "" && existing.NetworkId == req.NetworkId {
			return existing, false
		}
	}

	r.requests = append(r.requests, req)
	return req, true
}

func (r *requestRegistry) RemoveRequest(req *Request) bool {
	if req == nil {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for i, existing := range r.requests {
		if existing != req {
			continue
		}

		r.requests = append(r.requests[:i], r.requests[i+1:]...)
		if r.rootRequest == req {
			r.rootRequest = nil
		}
		return true
	}

	return false
}

func (r *requestRegistry) GetByNetworkId(id string) *Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, req := range r.requests {
		if req.NetworkId == id {
			return req
		}
	}
	return nil
}

func (r *requestRegistry) GetByRequestId(id string) *Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, req := range r.requests {
		if req.RequestId == id {
			return req
		}
	}
	return nil
}

func (r *requestRegistry) GetByUrl(url string, onlyNew bool) *Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, req := range r.requests {
		if req.Url == url {
			if !onlyNew || !req.GotNew {
				return req
			}
		}
	}
	return nil
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
		if len(snapshot.missingRequestIDs) > 0 {
			eventLog = eventLog.With("missingRequestIds", snapshot.missingRequestIDs)
		}
		eventLog.Debug("CrawlLog match status")
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
	urls := make(map[string]*Request)
	ids := make(map[string]*Request)
	r.rootRequest = r.requests[0]
	for idx, rr := range r.requests {
		urls[rr.Url] = rr
		if p, ok := ids[rr.NetworkId]; ok {
			rr.RedirectParent = p
			if p == r.rootRequest {
				r.rootRequest = rr
			}
		}
		ids[rr.NetworkId] = rr

		if rr.CrawlLog != nil {
			rr.CrawlLog.Referrer = rr.Referrer
			referrerRequest := urls[rr.Referrer]

			var discoveryType string
			if idx == 0 {
				discoveryType = requestedUrl.DiscoveryPath
			} else if rr.Initiator == "script" {
				// Resource is loaded by a script
				discoveryType = "X"
			} else if rr.RedirectParent != nil {
				discoveryType = "R"
			} else {
				discoveryType = "E"
			}

			if rr.RedirectParent != nil && rr.RedirectParent.CrawlLog != nil {
				rr.CrawlLog.DiscoveryPath = rr.RedirectParent.CrawlLog.DiscoveryPath + discoveryType
			} else if referrerRequest != nil && referrerRequest.CrawlLog != nil {
				rr.CrawlLog.DiscoveryPath = referrerRequest.CrawlLog.DiscoveryPath + discoveryType
			} else {
				rr.CrawlLog.DiscoveryPath = discoveryType
			}
		} else {
			if !rr.BlocksPageCompletion() {
				slog.Info("Skipping missing crawlLog for non-blocking request",
					"url", rr.Url,
					"requestId", rr.RequestId,
					"resourceType", rr.ResourceType)
				continue
			}
			slog.Warn("Missing crawlLog",
				"url", rr.Url,
				"index", idx,
				"requestId", rr.RequestId,
				"networkId", rr.NetworkId,
				"new", rr.GotNew,
				"complete", rr.GotComplete)
		}
	}
}

const maxLoggedMissingRequestIDs = 8

type crawlLogMatchSnapshot struct {
	blockingCount     int
	resolvedCount     int
	unresolvedCount   int
	ignoredCount      int
	missingRequestIDs []string
}

func (s crawlLogMatchSnapshot) signature() string {
	return fmt.Sprintf("%d|%d|%d|%d|%v", s.blockingCount, s.resolvedCount, s.unresolvedCount, s.ignoredCount, s.missingRequestIDs)
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
			if len(snapshot.missingRequestIDs) < maxLoggedMissingRequestIDs {
				snapshot.missingRequestIDs = append(snapshot.missingRequestIDs, req.RequestId)
			}
			continue
		}
		snapshot.resolvedCount++
	}
	return snapshot
}
