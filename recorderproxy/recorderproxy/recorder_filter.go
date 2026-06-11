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
	"bytes"
	"context"
	"net/http"
	"strings"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	dnsresolverV1 "github.com/NationalLibraryOfNorway/veidemann/api/dnsresolver/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/getlantern/proxy/v3/filters"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RecorderFilter is a filter which returns an error if the proxy is accessed as if it where a web server and not a proxy.
type RecorderFilter struct {
	proxyId           int32
	DnsResolverClient dnsresolverV1.DnsResolverClient
	hasNextProxy      bool
}

func (f *RecorderFilter) Apply(cs *filters.ConnectionState, req *http.Request, next filters.Next) (resp *http.Response, nextCS *filters.ConnectionState, err error) {
	ctx := filterContext(cs, req)
	l := rpcontext.LogWithContextAndRequest(ctx, req, "FLT:rec")
	connectErr := rpcontext.GetConnectError(ctx)
	if connectErr != nil {
		resp, nextCS, err = next(cs, req)
		return
	}

	if req.Method == http.MethodConnect {
		// Handle HTTPS CONNECT
		resp, nextCS, err = next(cs, req)
		if err != nil {
			l.WithError(err).Infof("Could not CONNECT to upstream server: %v", req.Host)
		}
		shortCircuitCS := nextCS
		if shortCircuitCS == nil {
			shortCircuitCS = cs
		}
		resp, nextCS, err = filters.ShortCircuit(shortCircuitCS, req, &http.Response{
			StatusCode: http.StatusOK,
		})
	} else {
		rc := rpcontext.GetRecordContext(ctx)
		span := opentracing.SpanFromContext(ctx)

		span.LogFields(log.String("event", "rec upstream request"))

		req, err = f.filterRequest(ctx, span, req, rc)
		if err != nil {
			return handleRequestError(cs, req, err)
		}

		roundTripSpan, roundtripCtx := opentracing.StartSpanFromContext(ctx, "Roundtrip upstream")
		req = req.WithContext(roundtripCtx)
		resp, nextCS, err = next(cs, req)
		roundTripSpan.Finish()
		if err != nil {
			return
		}

		resp, err = f.filterResponse(ctx, span, resp, rc)
		if err != nil {
			return handleRequestError(cs, req, err)
		}

		span.LogFields(log.String("event", "rec upstream response"))
	}
	return
}

func (f *RecorderFilter) filterRequest(ctx context.Context, span opentracing.Span, req *http.Request, rc *rpcontext.RecordContext) (*http.Request, error) {
	span.LogKV("event", "StartFilterRequest")

	var prolog bytes.Buffer
	err := writeRequestProlog(req, &prolog)
	if err != nil {
		e := errors.WrapInternalError(err, errors.RuntimeException, "Unable to write request headers", err.Error())
		return req, e
	}

	fetchTimeStamp := timestamppb.New(rc.FetchTimesTamp)
	uri := rc.Uri
	rc.IpAddress = rpcontext.GetIp(ctx)

	req.Header.Set(constants.HeaderAcceptEncoding, "identity")
	req.Header.Set(constants.HeaderCrawlExecutionId, rc.CrawlExecutionId)
	req.Header.Set(constants.HeaderJobExecutionId, rc.JobExecutionId)

	rc.Meta = &contentwriterV1.WriteRequest_Meta{
		Meta: &contentwriterV1.WriteRequestMeta{
			RecordMeta:     map[int32]*contentwriterV1.WriteRequestMeta_RecordMeta{},
			TargetUri:      uri.String(),
			ExecutionId:    rc.CrawlExecutionId,
			IpAddress:      rc.IpAddress,
			CollectionRef:  rc.CollectionRef,
			FetchTimeStamp: fetchTimeStamp,
		},
	}

	rc.CrawlLog.RequestedUri = uri.String()

	contentType := req.Header.Get("Content-Type")
	bodyWrapper, err := WrapRequestBody(ctx, rc, req.Body, contentType, prolog.Bytes())
	if err != nil {
		e := errors.WrapInternalError(err, errors.RuntimeException, "Veidemann proxy lost connection to GRPC services", err.Error())
		return req, e
	}
	req.Body = bodyWrapper

	return req, nil
}

func (f *RecorderFilter) filterResponse(ctx context.Context, span opentracing.Span, respOrig *http.Response, rc *rpcontext.RecordContext) (*http.Response, error) {
	span.LogKV("event", "StartFilterResponse")

	resp := respOrig
	if resp == nil {
		panic(http.ErrAbortHandler)
	}

	if rc.Error != nil && strings.HasPrefix(rc.Error.Error(), "unknown error from browser controller") {
		return resp, nil
	}

	if isFromCache(resp) {
		span.LogKV("event", "Loaded from cache")
		rpcontext.LogWithRecordContext(rc, "FLT:rec").Info("Loaded from cache")
		rc.FoundInCache = true
	}

	var prolog bytes.Buffer
	err := writeResponseProlog(resp, &prolog)
	if err != nil {
		e := errors.WrapInternalError(err, errors.RuntimeException, "Unable to write response headers", err.Error())
		return resp, e
	}

	contentType := resp.Header.Get("Content-Type")
	statusCode := int32(resp.StatusCode)
	bodyWrapper, err := WrapResponseBody(ctx, rc, resp.Body, statusCode, contentType, contentwriterV1.RecordType_RESPONSE, prolog.Bytes())
	if err != nil {
		e := errors.WrapInternalError(err, errors.RuntimeException, "Veidemann proxy lost connection to GRPC services", err.Error())
		return nil, e
	}

	if rc.ReplacementScript != nil {
		rpcontext.LogWithRecordContext(rc, "FLT:rec").Info("Replacement script")
		resp.ContentLength = int64(len(rc.ReplacementScript.Script))
	}
	resp.Body = bodyWrapper

	return resp, nil
}

func isFromCache(resp *http.Response) bool {
	cacheHeaders := resp.Header["X-Cache"]
	if cacheHeaders == nil {
		return false
	}

	for _, v := range cacheHeaders {
		if strings.Contains(v, "HIT from veidemann_cache") {
			return true
		}
	}

	return false
}
