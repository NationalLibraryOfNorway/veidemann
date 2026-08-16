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
	"net/http"
	"net/url"

	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	proxy "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/internal/proxy"
)

// chainedProxyFilter rewrites requests to support chained proxies.
type chainedProxyFilter struct {
	proxy *RecorderProxy
}

func (f *chainedProxyFilter) Apply(cs *proxy.State, req *http.Request, next proxy.Next) (resp *http.Response, nextCS *proxy.State, err error) {
	ctx := filterContext(cs, req)
	l := rpcontext.LogWithContextAndRequest(ctx, req, "FLT:chain")

	if req.Method == http.MethodConnect {
		resp, nextCS, err = next(cs, req)
	} else {
		if rpcontext.GetHost(ctx) == "" || (f.proxy.nextProxy != "" && !cs.IsMITMing()) {
			rc := rpcontext.GetRecordContext(ctx)
			uri, err := url.Parse("http:" + rc.Uri.String())
			if err != nil {
				l.WithError(err).Warnf("Error parsing uri for chained proxy: %v", "http:"+rc.Uri.String())
			}
			req.URL = uri
		}
		resp, nextCS, err = next(cs, req)
	}
	return
}
