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
	"fmt"
	"net/http"
	"strconv"

	dnsresolverV1 "github.com/NationalLibraryOfNorway/veidemann/api/dnsresolver/v1"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	proxy "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/internal/proxy"
	"google.golang.org/grpc/status"
)

type dnsLookupFilter struct {
	dnsResolverClient dnsresolverV1.DnsResolverClient
}

func (f *dnsLookupFilter) Apply(cs *proxy.State, req *http.Request, next proxy.Next) (resp *http.Response, nextCS *proxy.State, err error) {
	ctx := filterContext(cs, req)
	l := rpcontext.LogWithContextAndRequest(ctx, req, "FLT:dns")

	ip := rpcontext.GetIp(ctx)
	host := rpcontext.GetHost(ctx)
	port := rpcontext.GetPort(ctx)
	if ip == "" && host != "" {
		if e := f.resolve(ctx, host, port); e != nil {
			return handleRequestError(cs, req, e)
		}
		l.Debugf("resolved '%v' to '%v'", host, rpcontext.GetIp(ctx))
	}
	resp, nextCS, err = next(cs, req)
	return
}

func (f *dnsLookupFilter) resolve(ctx context.Context, host, port string) (err error) {
	var p = 0
	if port != "" {
		p, err = strconv.Atoi(port)
		if err != nil {
			err = errors.Wrap(err, errors.DomainLookupFailed, "illegal port", port)
			return
		}
	}
	dnsReq := &dnsresolverV1.ResolveRequest{
		ExecutionId:   rpcontext.GetCrawlExecutionId(ctx),
		CollectionRef: rpcontext.GetCollectionRef(ctx),
		Host:          host,
		Port:          int32(p),
	}

	dnsResp, err := f.dnsResolverClient.Resolve(ctx, dnsReq)
	s := status.Convert(err)
	if err != nil {
		err = errors.Wrap(err, errors.DomainLookupFailed, fmt.Sprintf("Got 'no such host' from DNS for host: %s, port: %s", host, port), s.Message())
		return
	}

	rpcontext.SetIp(ctx, dnsResp.TextualIp)
	return
}
