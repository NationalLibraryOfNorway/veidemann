/*
 * Copyright 2026 National Library of Norway.
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

package server

import (
	"context"
	"errors"
	"io"
	"testing"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	"google.golang.org/grpc/metadata"
)

type contextRecordingConfigAdapter struct {
	ctx context.Context
}

func (a *contextRecordingConfigAdapter) GetConfigObject(ctx context.Context, _ *configV1.ConfigRef) (*configV1.ConfigObject, error) {
	a.ctx = ctx
	return nil, errors.New("stop after recording context")
}

type contextTestWriteServer struct {
	ctx      context.Context
	requests []*contentwriterV1.WriteRequest
}

func (s *contextTestWriteServer) Recv() (*contentwriterV1.WriteRequest, error) {
	if len(s.requests) == 0 {
		return nil, io.EOF
	}
	request := s.requests[0]
	s.requests = s.requests[1:]
	return request, nil
}

func (*contextTestWriteServer) SendAndClose(*contentwriterV1.WriteReply) error { return nil }
func (*contextTestWriteServer) SetHeader(metadata.MD) error                    { return nil }
func (*contextTestWriteServer) SendHeader(metadata.MD) error                   { return nil }
func (*contextTestWriteServer) SetTrailer(metadata.MD)                         {}
func (s *contextTestWriteServer) Context() context.Context                     { return s.ctx }
func (*contextTestWriteServer) SendMsg(any) error                              { return nil }
func (*contextTestWriteServer) RecvMsg(any) error                              { return nil }

func TestWriteUsesStreamContextForConfigLookup(t *testing.T) {
	adapter := &contextRecordingConfigAdapter{}
	service := &ContentWriterService{configAdapter: adapter}
	wantCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := &contextTestWriteServer{
		ctx: wantCtx,
		requests: []*contentwriterV1.WriteRequest{
			{
				Value: &contentwriterV1.WriteRequest_Meta{Meta: &contentwriterV1.WriteRequestMeta{
					TargetUri:     "https://example.com/",
					IpAddress:     "192.0.2.1",
					CollectionRef: &configV1.ConfigRef{Id: "collection-id"},
				}},
			},
		},
	}

	err := service.Write(stream)

	if err == nil {
		t.Fatal("Write() error = nil, want an error")
	}
	if adapter.ctx != wantCtx {
		t.Errorf("config lookup context = %v, want stream context %v", adapter.ctx, wantCtx)
	}
}
