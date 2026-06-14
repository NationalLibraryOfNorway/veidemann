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

	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/serviceconnections"
	"github.com/getlantern/proxy/v3/filters"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

// ContextInitFilter is a filter which initializes the context with sessions to external services.
type ContextInitFilter struct {
	conn    *serviceconnections.Connections
	proxyId int32
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

func requestBaseURI(ctx context.Context, cs *filters.ConnectionState, req *http.Request) *url.URL {
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

func (f *ContextInitFilter) Apply(cs *filters.ConnectionState, req *http.Request, next filters.Next) (resp *http.Response, nextCS *filters.ConnectionState, err error) {
	ctx := filterContext(cs, req)
	if req.Method == http.MethodConnect {
		l := rpcontext.LogWithContextAndRequest(ctx, req, "FLT:ctx")
		rpcontext.ResetRequestState(ctx, false)

		// Handle HTTPS CONNECT
		rpcontext.SetHost(ctx, req.URL.Hostname())
		rpcontext.SetPort(ctx, req.URL.Port())

		// Copy URI by value and add scheme
		uv := *req.URL
		uri := &uv
		uri.Scheme = "https"

		req = req.WithContext(ctx)

		rpcontext.SetUri(ctx, uri)
		rpcontext.RegisterConnectRequest(ctx, f.conn, f.proxyId, req, uri)

		l.Debugf("Converted CONNECT request uri form %v to %v", req.URL, uri)
		resp, nextCS, err = next(cs, req)
	} else {
		connectionCtx := ctx
		preserveSessionMetadata := cs.IsMITMing()
		requestCtx := rpcontext.WrapIfNecessary(rpcontext.NewRequestContext(ctx, preserveSessionMetadata))
		rpcontext.ResetRequestState(requestCtx, preserveSessionMetadata)
		l := rpcontext.LogWithContextAndRequest(requestCtx, req, "FLT:ctx")

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
			rpcontext.SetUri(requestCtx, &url.URL{Scheme: uri.Scheme, Host: uri.Host})
		}

		l.Debugf("Converted GET request uri form %v to %v", req.URL, uri)

		req = req.WithContext(requestCtx)
		rc := rpcontext.NewRecordContext()
		rpcontext.SetRecordContext(requestCtx, rc)
		span := opentracing.SpanFromContext(requestCtx)
		span.LogFields(log.String("event", "Start init record context"))
		rc.Init(f.proxyId, f.conn, req, uri)

		if e := rc.RegisterNewRequest(requestCtx); e != nil {
			span.LogFields(log.String("event", "Failed init record context"), log.Error(e))
			return handleRequestError(cs, req, e)
		}
		span.LogFields(log.String("event", "Finished init record context"))

		req = req.WithContext(requestCtx)
		resp, nextCS, err = next(cs, req)
		if preserveSessionMetadata {
			rpcontext.CopySessionMetadata(connectionCtx, requestCtx)
		}
	}
	return
}
