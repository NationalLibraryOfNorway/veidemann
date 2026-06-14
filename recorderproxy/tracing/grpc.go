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

package tracing

import (
	"context"
	"fmt"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

type sh struct {
	service string
}

// NewStatsHandler creates a stats.Handler for gRPC which logs all traffic if loglevel is equal or finer than the submitted loglevel
func NewStatsHandler(serviceName string, loglevel logger.Level) grpc.DialOption {
	if logger.IsLevelEnabled(loglevel) {
		return grpc.WithStatsHandler(&sh{"gRPC:" + serviceName})
	} else {
		return grpc.EmptyDialOption{}
	}
}

func (h *sh) TagRPC(c context.Context, i *stats.RPCTagInfo) context.Context {
	rpcontext.LogWithContext(c, h.service).Debugf("TagRPC: %s %v", i.FullMethodName, i.FailFast)
	return c
}

func (h *sh) HandleRPC(c context.Context, s stats.RPCStats) {
	span := opentracing.SpanFromContext(c)
	switch v := s.(type) {
	case *stats.Begin:
		rpcontext.LogWithContext(c, h.service).Debugf("Begin HandleRPC: %v", v.BeginTime)
		span.LogKV("event", fmt.Sprintf("%s Begin", h.service))
	case *stats.End:
		rpcontext.LogWithContext(c, h.service).Debugf("End HandleRPC: %v, %v, %v", v.BeginTime, v.EndTime, v.Error)
		span.LogKV("event", fmt.Sprintf("%s End %v", h.service, v.Trailer))
	case *stats.InHeader:
		rpcontext.LogWithContext(c, h.service).Debugf("InHeader HandleRPC: %v", v)
	case *stats.InPayload:
		rpcontext.LogWithContext(c, h.service).Debugf("InPayload HandleRPC: %T", v.Payload)
		span.LogKV(
			"xx", fmt.Sprintf("%T", v.Payload),
			"data", fmt.Sprintf("%v", v.Payload),
			"component", h.service,
			"direction", "in",
		)
	case *stats.InTrailer:
		rpcontext.LogWithContext(c, h.service).Debugf("InTrailer HandleRPC: %v", v)
	case *stats.OutHeader:
		rpcontext.LogWithContext(c, h.service).Debugf("OutHeader HandleRPC: %v", v)
	case *stats.OutPayload:
		switch p := v.Payload.(type) {
		//case *dnsresolver.ResolveRequest:
		case *contentwriterV1.WriteRequest:
			switch w := p.GetValue().(type) {
			case *contentwriterV1.WriteRequest_Meta:
				rpcontext.LogWithContext(c, h.service).Debug(w.Meta)
			case *contentwriterV1.WriteRequest_Payload:
				if logger.IsLevelEnabled(logger.TraceLevel) {
					rpcontext.LogWithContext(c, h.service).Tracef("payload[%v]: %v", w.Payload.RecordNum, string(w.Payload.Data))
				} else {
					rpcontext.LogWithContext(c, h.service).Debugf("payload[%v]: %v bytes", w.Payload.RecordNum, len(w.Payload.Data))
				}
			case *contentwriterV1.WriteRequest_Cancel:
				rpcontext.LogWithContext(c, h.service).Debug(w.Cancel)
			case *contentwriterV1.WriteRequest_ProtocolHeader:
				rpcontext.LogWithContext(c, h.service).Debugf("header[%v]: %v", w.ProtocolHeader.RecordNum, string(w.ProtocolHeader.Data))
			}
		default:
			rpcontext.LogWithContext(c, h.service).Debugf("OutPayload HandleRPC: %T", v.Payload)
		}
		span.LogKV(
			"xx", fmt.Sprintf("%T", v.Payload),
			"data", fmt.Sprintf("%v", v.Payload),
			"component", h.service,
			"direction", "out",
		)
	case *stats.OutTrailer:
		rpcontext.LogWithContext(c, h.service).Debugf("OutTrailer HandleRPC: %v", v)
	default:
		rpcontext.LogWithContext(c, h.service).Debugf("HandleRPC: isclient %v %T", s.IsClient(), s)
	}
}

func (h *sh) TagConn(c context.Context, i *stats.ConnTagInfo) context.Context {
	rpcontext.LogWithContext(c, h.service).Debugf("TagConn: %s --> %s\n", i.LocalAddr, i.RemoteAddr)
	return c
}

func (h *sh) HandleConn(c context.Context, s stats.ConnStats) {
	switch v := s.(type) {
	case *stats.ConnBegin:
		rpcontext.LogWithContext(c, h.service).Debugf("Begin HandleConn: isclient %v", v.IsClient())
	case *stats.ConnEnd:
		rpcontext.LogWithContext(c, h.service).Debugf("End HandleConn: isclient %v", v.IsClient())
	default:
		rpcontext.LogWithContext(c, h.service).Debugf("HandleConn: isclient %v %T", s.IsClient(), s)
	}
}
