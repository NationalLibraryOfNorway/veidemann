/*
 * Copyright 2019 National Library of Norway.
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

package recorderproxy

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	robotsV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	rperrors "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	proxy "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/internal/proxy"
)

// contextInitFilter initializes request state with the ContentWriter client.
type contextInitFilter struct {
	robots              robotsV1.RobotsEvaluatorClient
	contentWriterClient contentwriterV1.ContentWriterClient
	proxyId             int32
	finalizationTimeout time.Duration
	lifecycle           *lifecycleTracker
}

func requestAuthority(req *http.Request) (string, string) {
	if req == nil {
		return "", ""
	}

	if host := req.URL.Hostname(); host != "" {
		return host, req.URL.Port()
	}

	if req.Host == "" {
		return "", ""
	}

	host, port, err := net.SplitHostPort(req.Host)
	if err == nil {
		return host, port
	}

	return req.Host, ""
}

func requestBaseURI(ctx context.Context, cs *proxy.State, req *http.Request) *url.URL {
	if req == nil {
		return nil
	}

	if req.URL.IsAbs() {
		return &url.URL{Scheme: req.URL.Scheme, Host: req.URL.Host}
	}

	authority := req.Host
	if authority != "" {
		scheme := "http"
		if cs.IsMITMing() {
			scheme = "https"
		}
		return &url.URL{Scheme: scheme, Host: authority}
	}

	if uri := rpcontext.GetUri(ctx); uri != nil && uri.Host != "" {
		return uri
	}

	if authority == "" {
		host := rpcontext.GetHost(ctx)
		if host == "" {
			return nil
		}
		if port := rpcontext.GetPort(ctx); port != "" {
			authority = net.JoinHostPort(host, port)
		} else {
			authority = host
		}
	}

	scheme := "http"
	if cs.IsMITMing() {
		scheme = "https"
	}

	return &url.URL{Scheme: scheme, Host: authority}
}

func (f *contextInitFilter) Apply(cs *proxy.State, req *http.Request, next proxy.Next) (resp *http.Response, nextCS *proxy.State, err error) {
	if req.Method == http.MethodConnect {
		return f.applyConnect(cs, req, next)
	} else {
		return f.apply(cs, req, next)
	}
}

func (f *contextInitFilter) applyConnect(cs *proxy.State, req *http.Request, next proxy.Next) (resp *http.Response, nextCS *proxy.State, err error) {
	ctx := filterContext(cs, req)

	host, port, hostPort := connectAuthority(req)
	uri := &url.URL{
		Scheme: "https",
		Host:   hostPort,
	}

	rpcontext.ResetRequestState(ctx, false)
	rpcontext.SetHost(ctx, host)
	rpcontext.SetPort(ctx, port)
	rpcontext.SetUri(ctx, uri)

	l := rpcontext.LogWithContextAndRequest(ctx, req, "connection_setup")

	if a := requestAttempt(cs); a != nil {
		a.mu.Lock()
		closed := a.closing
		a.mu.Unlock()
		if closed {
			return Deny(cs, req, http.StatusForbidden, "Harvest closed")
		}
		rpcontext.SetCrawlExecutionId(ctx, a.spec.CrawlExecutionId)
		rpcontext.SetJobExecutionId(ctx, a.spec.JobExecutionId)
		rpcontext.SetCollectionRef(ctx, a.spec.CollectionRef)
	} else {
		return Deny(cs, req, http.StatusForbidden, "Active harvest required")
	}

	req = req.WithContext(ctx)

	l.Trace("Context initialized")

	return next(cs, req)
}

func (f *contextInitFilter) apply(cs *proxy.State, req *http.Request, next proxy.Next) (resp *http.Response, nextCS *proxy.State, err error) {
	ctx := filterContext(cs, req)

	connectionCtx := ctx

	preserveSessionMetadata := cs.IsMITMing()
	requestCtx := rpcontext.WrapIfNecessary(rpcontext.NewRequestContext(ctx, preserveSessionMetadata))

	rpcontext.ResetRequestState(requestCtx, preserveSessionMetadata)
	if host, port := requestAuthority(req); host != "" {
		rpcontext.SetHost(requestCtx, host)
		rpcontext.SetPort(requestCtx, port)
	} else if rpcontext.GetHost(requestCtx) == "" {
		rpcontext.SetHost(requestCtx, req.URL.Hostname())
		rpcontext.SetPort(requestCtx, req.URL.Port())
	}

	baseURI := requestBaseURI(requestCtx, cs, req)
	var uri *url.URL
	if baseURI != nil {
		uri = baseURI.ResolveReference(req.URL)
	} else {
		uri = req.URL
	}
	if uri != nil && uri.Host != "" {
		newUri := &url.URL{Scheme: uri.Scheme, Host: uri.Host}
		rpcontext.SetUri(requestCtx, newUri)
	}

	req = req.WithContext(requestCtx)

	binding := requestAttemptConnection(cs)
	if binding == nil {
		return Deny(cs, req, http.StatusForbidden, "Active harvest required")
	}
	a := binding.attempt
	// Connections pin an attempt at accept, but a surviving old browser may
	// reconnect after recovery. Reject it before admission, policy, or storage.
	attemptID := req.Header.Get(constants.HeaderAttemptId)
	req.Header.Del(constants.HeaderAttemptId)
	if attemptID != a.spec.AttemptId {
		return Deny(cs, req, http.StatusForbidden, "Harvest identity mismatch")
	}
	// Pre-request CONNECT/TLS errors from an unverified browser cannot poison
	// the current attempt. Once identified, connection failures belong to it.
	binding.verified.Store(true)
	rc := rpcontext.NewRecordContext(f.finalizationTimeout)
	initialize := func() {
		f.lifecycle.addRecord(rc)
		a.lifecycle.addRecord(rc)
		rpcontext.SetRecordContext(requestCtx, rc)
		rc.Init(f.proxyId, f.contentWriterClient, req, uri)
	}
	if !a.admit(rc, initialize) {
		return Deny(cs, req, http.StatusForbidden, "Harvest closed")
	}
	if !a.allowed(requestCtx, f.robots, uri.String()) {
		rc.PrecludedByRobots = true
		return handleRequestError(cs, req, rperrors.Error(rperrors.PrecludedByRobots, "PRECLUDED_BY_ROBOTS", "Robots.txt rules precluded fetch"))
	}

	req = req.WithContext(requestCtx)
	resp, nextCS, err = next(cs, req)
	if err == nil && resp != nil && rc.ObserveResponse != nil {
		rc.ObserveResponse(resp)
	}

	if preserveSessionMetadata {
		rpcontext.CopySessionMetadata(connectionCtx, requestCtx)
	}
	return
}

func connectAuthority(req *http.Request) (host string, port string, hostPort string) {
	hostPort = req.Host

	if hostPort == "" && req.URL != nil {
		hostPort = req.URL.Host
	}

	if hostPort == "" && req.URL != nil {
		hostPort = req.URL.Opaque
	}

	hostPort = strings.ToLower(strings.TrimSpace(hostPort))

	h, p, err := net.SplitHostPort(hostPort)
	if err == nil {
		return h, p, net.JoinHostPort(h, p)
	}

	return hostPort, "443", net.JoinHostPort(hostPort, "443")
}

func requestAttempt(cs *proxy.State) *harvestAttempt {
	if binding := requestAttemptConnection(cs); binding != nil {
		return binding.attempt
	}
	return nil
}

func requestAttemptConnection(cs *proxy.State) *attemptConnection {
	if cs == nil {
		return nil
	}
	ctx := proxyContextFromConn(cs.Downstream())
	if ctx == nil {
		return nil
	}
	binding, _ := ctx.Value(attemptKey{}).(*attemptConnection)
	return binding
}
