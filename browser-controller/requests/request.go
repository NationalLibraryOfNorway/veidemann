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
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
)

type Request struct {
	ID string // canonical internal ID: NetworkID when available, else FetchRequestID

	FetchRequestID string // only for Fetch.continue/fail/fulfill
	NetworkID      string // Network.RequestID / Fetch.NetworkID

	URL         string
	OriginalURL string
	Method      string

	ResourceType string
	Referrer     string
	Initiator    string

	GotNew      bool
	GotComplete bool
	CrawlLog    *logV1.CrawlLog

	Redirected      bool
	RedirectFromURL string

	RedirectParent *Request
	FromCache      bool
}

func (r *Request) BlocksPageCompletion() bool {
	if r == nil {
		return false
	}

	if !r.GotNew {
		return false
	}

	switch r.ResourceType {
	case "Ping",
		"EventSource",
		"WebSocket",
		"CSPViolationReport",
		"Preflight",
		"XHR",
		"Fetch",
		"Manifest",
		"SignedExchange",
		"FedCM",
		"Other",
		"Prefetch":
		return false
	default:
		return true
	}
}

func mergeRequest(dst, src *Request) {
	if dst.ID == "" {
		dst.ID = src.ID
	}
	if dst.FetchRequestID == "" {
		dst.FetchRequestID = src.FetchRequestID
	}
	if dst.NetworkID == "" {
		dst.NetworkID = src.NetworkID
	}

	if src.URL != "" {
		if dst.OriginalURL == "" {
			dst.OriginalURL = src.URL
		}
		dst.URL = src.URL
	}

	if dst.Method == "" {
		dst.Method = src.Method
	}
	if dst.ResourceType == "" {
		dst.ResourceType = src.ResourceType
	}
	if dst.Referrer == "" {
		dst.Referrer = src.Referrer
	}
	if dst.Initiator == "" {
		dst.Initiator = src.Initiator
	}

	if src.Redirected {
		dst.Redirected = true
		if dst.RedirectFromURL == "" {
			dst.RedirectFromURL = src.RedirectFromURL
		}
	}

	dst.GotNew = dst.GotNew || src.GotNew
	dst.GotComplete = dst.GotComplete || src.GotComplete
	dst.FromCache = dst.FromCache || src.FromCache

	if dst.CrawlLog == nil {
		dst.CrawlLog = src.CrawlLog
	}
	if dst.RedirectParent == nil {
		dst.RedirectParent = src.RedirectParent
	}
}
