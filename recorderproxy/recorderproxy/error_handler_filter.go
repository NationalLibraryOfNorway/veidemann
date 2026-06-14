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
	"crypto/tls"
	"net"
	"net/http"

	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	rperrors "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	glerrors "github.com/getlantern/errors"
	"github.com/getlantern/proxy/v3/filters"
)

// ErrorHandlerFilter is a filter which initializes the context with sessions to external services.
type ErrorHandlerFilter struct {
	hasNextProxy bool
}

func (f *ErrorHandlerFilter) Apply(cs *filters.ConnectionState, req *http.Request, next filters.Next) (resp *http.Response, nextCS *filters.ConnectionState, err error) {
	ctx := filterContext(cs, req)
	l := rpcontext.LogWithContextAndRequest(ctx, req, "FLT:err")

	connectErr := rpcontext.GetConnectError(ctx)
	if connectErr != nil {
		l.WithError(connectErr).WithField("method", req.Method).Debug("Handle connect error")
		e := f.normalizeError(connectErr, l)
		return handleRequestError(cs, req, e)
	}

	resp, nextCS, err = next(cs, req)

	if err != nil {
		l.WithError(err).Debug("Handle roundtrip error")

		e := f.normalizeError(err, l)
		return handleRequestError(cs, req, e)
	}

	squidErr := resp.Header.Get("X-Squid-Error")
	if squidErr != "" {
		e := handleSquidErrorString(squidErr)
		if e != nil {
			return handleRequestError(cs, req, e)
		}
	}

	return
}

func (f *ErrorHandlerFilter) normalizeError(err error, l *logger.Logger) error {
	l = l.WithError(err)
	l.Tracef("Normalize error (type: %T): %v", err, err)
	switch e := err.(type) {
	case *rperrors.ProxyError:
		return e
	case *net.OpError:
		return f.normalizeNetOpError(e, l)
	case tls.RecordHeaderError:
		return rperrors.Wrap(&e, rperrors.ConnectFailed, "CONNECT_FAILED", "tls: handshake failure")
	case glerrors.Error:
		return f.normalizeGetlanternProxyError(e, l)
	default:
		switch s := e.Error(); {
		case s == "EOF":
			return rperrors.Wrap(e, rperrors.EmptyResponse, "EMPTY_RESPONSE", "Empty reply from server")
		default:
			l.Debugf("Unknown error (type: %T): %v. Returning -5 UNKNOWN_ERROR", err, err)
			return rperrors.Wrap(e, rperrors.RuntimeException, "UNKNOWN_ERROR", s)
		}
	}
}

func (f *ErrorHandlerFilter) normalizeNetOpError(err *net.OpError, l *logger.Logger) error {
	l.Tracef("Normalize error (type: %T) (op: %s): %v", err, err.Op, err)
	var e error
	switch err.Op {
	case "dial":
		e = rperrors.Wrap(err.Err, rperrors.ConnectFailed, "CONNECT_FAILED", err.Err.Error())
	case "remote error":
		e = rperrors.Wrap(err.Err, rperrors.ConnectFailed, "CONNECT_FAILED", err.Err.Error())
	default:
		l.Debugf("Unknown error operation (type: %T): %v. Returning -2 CONNECT_FAILED", err, err)
		e = rperrors.Wrap(err, rperrors.ConnectFailed, "CONNECT_FAILED", err.Error())
	}
	return e
}

func (f *ErrorHandlerFilter) normalizeGetlanternProxyError(err glerrors.Error, l *logger.Logger) error {
	l.Tracef("Normalize getlantern error (type: %T) (root cause type: %T): %v", err, err.RootCause(), err.ErrorClean())
	switch e := err.RootCause().(type) {
	case *net.OpError:
		return f.normalizeNetOpError(e, l)
	default:
		switch s := e.Error(); {
		case s == "EOF":
			return rperrors.Wrap(e, rperrors.EmptyResponse, "EMPTY_RESPONSE", "Empty reply from server")
		default:
			l.Debugf("Unknown root cause (type: %T) for proxy err '%s': %v. Returning -5 UNKNOWN_ERROR", err.RootCause(), err.ErrorClean(), err.RootCause())
			return rperrors.Wrap(e, rperrors.RuntimeException, "UNKNOWN_ERROR", s)
		}
	}
}
