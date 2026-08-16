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

	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	proxy "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/internal/proxy"
)

// errorHandlerFilter converts downstream failures into canonical recorder responses.
type errorHandlerFilter struct{}

func (f *errorHandlerFilter) Apply(cs *proxy.State, req *http.Request, next proxy.Next) (resp *http.Response, nextCS *proxy.State, err error) {
	ctx := filterContext(cs, req)
	l := rpcontext.LogWithContextAndRequest(ctx, req, "FLT:err")

	resp, nextCS, err = next(cs, req)

	if err != nil {
		l.WithError(err).Debug("Handle roundtrip error")

		failure := classifyFailure(err, FailureScopeResource)
		l.WithField("phase", failure.Phase).Debug("Classified request failure")
		return handleRequestError(cs, req, failure.asError())
	}

	if resp != nil {
		squidErr := resp.Header.Get("X-Squid-Error")
		if squidErr != "" {
			squidFailure := handleSquidErrorString(squidErr)
			if squidFailure != nil {
				failure := classifyFailure(squidFailure, FailureScopeResource)
				return handleRequestError(cs, req, failure.asError())
			}
		}
	}

	return
}
