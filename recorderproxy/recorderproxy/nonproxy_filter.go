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

	"github.com/getlantern/proxy/v3/filters"
)

var errNonProxyRequest = errors.New("this is a proxy server and does not respond to non-proxy requests")

// NonproxyFilter is a filter which returns an error if the proxy is accessed as if it where a web server and not a proxy.
type NonproxyFilter struct{}

func (f *NonproxyFilter) Apply(cs *filters.ConnectionState, req *http.Request, next filters.Next) (resp *http.Response, nextCS *filters.ConnectionState, err error) {
	if req.Method == http.MethodConnect {
		return next(cs, req)
	}

	if !req.URL.IsAbs() && !cs.IsMITMing() {
		resp, nextCS, _ := filters.Fail(
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
