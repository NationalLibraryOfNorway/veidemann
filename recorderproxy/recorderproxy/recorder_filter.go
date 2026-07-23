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
	"net/http"

	dnsresolverV1 "github.com/NationalLibraryOfNorway/veidemann/api/dnsresolver/v1"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	"github.com/getlantern/proxy/v3/filters"
)

// RecorderFilter is a filter which returns an error if the proxy is accessed as if it where a web server and not a proxy.
type RecorderFilter struct {
	proxyId           int32
	DnsResolverClient dnsresolverV1.DnsResolverClient
	hasNextProxy      bool
}

func (f *RecorderFilter) Apply(cs *filters.ConnectionState, req *http.Request, next filters.Next) (resp *http.Response, nextCS *filters.ConnectionState, err error) {
	ctx := filterContext(cs, req)

	if req.Method == http.MethodConnect {
		return next(cs, req)
	}

	rc := rpcontext.GetRecordContext(ctx)

	req, err = f.filterRequest(ctx, req, rc)
	if err != nil {
		return handleRequestError(cs, req, err)
	}

	resp, nextCS, err = next(cs, req)
	if err != nil {
		return
	}

	resp, err = f.filterResponse(ctx, resp, rc)
	if err != nil {
		return handleRequestError(cs, req, err)
	}

	return
}

func (f *RecorderFilter) filterRequest(
	ctx context.Context,
	req *http.Request,
	rc *rpcontext.RecordContext,
) (*http.Request, error) {
	return newRequestRecorder(ctx, rc).Wrap(req)
}

func (f *RecorderFilter) filterResponse(
	ctx context.Context,
	respOrig *http.Response,
	rc *rpcontext.RecordContext,
) (*http.Response, error) {
	return newResponseRecorder(ctx, rc).Wrap(respOrig)
}
