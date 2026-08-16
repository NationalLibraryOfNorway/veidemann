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
	"errors"
	"net/http"

	proxy "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/internal/proxy"
)

var errNonProxyRequest = errors.New("this is a proxy server and does not respond to non-proxy requests")

// nonproxyFilter rejects requests that address the proxy as an origin server.
type nonproxyFilter struct{}

func (f *nonproxyFilter) Apply(cs *proxy.State, req *http.Request, next proxy.Next) (resp *http.Response, nextCS *proxy.State, err error) {
	if req.Method == http.MethodConnect {
		return next(cs, req)
	}

	if !req.URL.IsAbs() && !cs.IsMITMing() {
		resp, nextCS, _ := proxy.Fail(
			cs,
			req,
			http.StatusBadRequest,
			errNonProxyRequest,
		)
		if resp != nil {
			resp.Close = true
		}
		return resp, nextCS, nil
	}

	return next(cs, req)
}
