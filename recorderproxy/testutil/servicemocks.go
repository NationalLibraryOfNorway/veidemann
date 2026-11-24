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

package testutil

import (
	"context"
	"crypto/sha1"
	"fmt"
	"hash"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	dnsresolverV1 "github.com/NationalLibraryOfNorway/veidemann/api/dnsresolver/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/tracing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

/**
 * Server mocks
 */
type GrpcServiceMock struct {
	LogWriterMock
	dnsresolverV1.UnimplementedDnsResolverServer
	contentwriterV1.UnimplementedContentWriterServer

	dnsResolverConn        *grpc.ClientConn
	contentWriterConn      *grpc.ClientConn
	lis                    *bufconn.Listener
	mu                     sync.Mutex
	Requests               Requests
	Server                 *grpc.Server
	ContentWriterClient    contentwriterV1.ContentWriterClient
	DnsResolverClient      dnsresolverV1.DnsResolverClient
	ContentWriterWriteFunc func(contentwriterV1.ContentWriter_WriteServer) error
}

// MockOption configures service clients or mock behavior.
type MockOption interface {
	apply(*GrpcServiceMock)
}

// funcMockOption wraps a function that modifies GrpcServiceMock into an
// implementation of the MockOption interface.
type funcMockOption struct {
	f func(*GrpcServiceMock)
}

func (fmo *funcMockOption) apply(mock *GrpcServiceMock) {
	fmo.f(mock)
}

func newFuncMockOption(f func(*GrpcServiceMock)) *funcMockOption {
	return &funcMockOption{
		f: f,
	}
}

// WithExternalContentWriter injects a client whose connection the caller owns.
func WithExternalContentWriter(client contentwriterV1.ContentWriterClient) MockOption {
	return newFuncMockOption(func(c *GrpcServiceMock) {
		c.ContentWriterClient = client
	})
}

// WithExternalDns injects a client whose connection the caller owns.
func WithExternalDns(client dnsresolverV1.DnsResolverClient) MockOption {
	return newFuncMockOption(func(c *GrpcServiceMock) {
		c.DnsResolverClient = client
	})
}

func WithContentWriterWriteFunc(writeFunc func(contentwriterV1.ContentWriter_WriteServer) error) MockOption {
	return newFuncMockOption(func(c *GrpcServiceMock) {
		c.ContentWriterWriteFunc = writeFunc
	})
}

type Requests struct {
	DnsResolverRequests   []*dnsresolverV1.ResolveRequest
	ContentWriterRequests []*contentwriterV1.WriteRequest
}

func NewGrpcServiceMock(opts ...MockOption) *GrpcServiceMock {
	m := &GrpcServiceMock{
		lis: bufconn.Listen(bufSize),
	}

	for _, opt := range opts {
		opt.apply(m)
	}

	m.Server = grpc.NewServer()

	if m.DnsResolverClient == nil {
		dnsresolverV1.RegisterDnsResolverServer(m.Server, m)
	}
	if m.ContentWriterClient == nil {
		contentwriterV1.RegisterContentWriterServer(m.Server, m)
	}

	go func() {
		if err := m.Server.Serve(m.lis); err != nil {
			logger.LogWithPhase("MOCK:grpc").Fatalf("Server exited with error: %v", err)
		}
	}()

	dialOption := grpc.WithContextDialer(m.bufDialer)

	var err error
	if m.ContentWriterClient == nil {
		m.contentWriterConn, err = grpc.NewClient(
			"passthrough:///bufnet",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			dialOption,
			tracing.NewStatsHandler("ContentWriter", logger.DebugLevel),
		)
		if err != nil {
			m.Close()
			logger.LogWithPhase("MOCK:grpc").Panicf("Could not create ContentWriter client: %v", err)
		}
		m.ContentWriterClient = contentwriterV1.NewContentWriterClient(m.contentWriterConn)
	}
	if m.DnsResolverClient == nil {
		m.dnsResolverConn, err = grpc.NewClient(
			"passthrough:///bufnet",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			dialOption,
			tracing.NewStatsHandler("DnsService", logger.DebugLevel),
		)
		if err != nil {
			m.Close()
			logger.LogWithPhase("MOCK:grpc").Panicf("Could not create DNS resolver client: %v", err)
		}
		m.DnsResolverClient = dnsresolverV1.NewDnsResolverClient(m.dnsResolverConn)
	}

	return m
}

func (s *GrpcServiceMock) Close() {
	if s.contentWriterConn != nil {
		_ = s.contentWriterConn.Close()
	}
	if s.dnsResolverConn != nil {
		_ = s.dnsResolverConn.Close()
	}
	s.Server.GracefulStop()
	s.lis.Close()
}

func (s *GrpcServiceMock) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Requests = Requests{}
}

func (s *GrpcServiceMock) WaitForRequestCounts(dns, contentWriter int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		s.mu.Lock()
		ready := len(s.Requests.DnsResolverRequests) >= dns &&
			len(s.Requests.ContentWriterRequests) >= contentWriter
		s.mu.Unlock()
		if ready {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(time.Millisecond)
	}
}

func (s *GrpcServiceMock) bufDialer(context.Context, string) (net.Conn, error) {
	return s.lis.Dial()
}

func (s *GrpcServiceMock) addDnsRequest(r *dnsresolverV1.ResolveRequest) {
	s.mu.Lock()

	s.Requests.DnsResolverRequests = append(s.Requests.DnsResolverRequests, r)
	s.mu.Unlock()
}

func (s *GrpcServiceMock) addCwRequest(r *contentwriterV1.WriteRequest) {
	s.mu.Lock()

	s.Requests.ContentWriterRequests = append(s.Requests.ContentWriterRequests, r)
	s.mu.Unlock()
}

// Implements DNS service
func (s *GrpcServiceMock) Resolve(ctx context.Context, in *dnsresolverV1.ResolveRequest) (*dnsresolverV1.ResolveReply, error) {
	s.addDnsRequest(in)

	ips, err := net.LookupIP(in.Host)
	if err == nil {
		for _, ip := range ips {
			if ip.To4() != nil {
				out := &dnsresolverV1.ResolveReply{
					Host:      in.Host,
					Port:      in.Port,
					TextualIp: ip.To4().String(),
					RawIp:     ip,
				}
				return out, nil
			}
		}
	}
	return nil, err
}

// Implements ContentWriterService
func (s *GrpcServiceMock) Write(server contentwriterV1.ContentWriter_WriteServer) error {
	if s.ContentWriterWriteFunc != nil {
		return s.ContentWriterWriteFunc(server)
	}
	records := map[int32]*contentwriterV1.WriteResponseMeta_RecordMeta{}
	data := make(map[int32][]byte)
	size := make(map[int32]int64)
	gotMeta := false
	gotCancel := false
	blockDigest := make(map[int32]hash.Hash)
	payloadDigest := make(map[int32]hash.Hash)

	for {
		request, err := server.Recv()
		if err == io.EOF {
			if !gotMeta && !gotCancel {
				return fmt.Errorf("missing metadata")
			}
			return server.SendAndClose(&contentwriterV1.WriteReply{
				Meta: &contentwriterV1.WriteResponseMeta{
					RecordMeta: records,
				},
			})
		}
		if err != nil {
			return err
		}

		s.addCwRequest(request)

		switch v := request.Value.(type) {
		case *contentwriterV1.WriteRequest_ProtocolHeader:
			size[v.ProtocolHeader.RecordNum] = int64(len(v.ProtocolHeader.Data))
			blockDigest[v.ProtocolHeader.RecordNum] = sha1.New()
			payloadDigest[v.ProtocolHeader.RecordNum] = sha1.New()
			blockDigest[v.ProtocolHeader.RecordNum].Write(v.ProtocolHeader.Data)
			data[v.ProtocolHeader.RecordNum] = v.ProtocolHeader.Data
		case *contentwriterV1.WriteRequest_Payload:
			size[v.Payload.RecordNum] += int64(len(v.Payload.Data))
			blockDigest[v.Payload.RecordNum].Write(v.Payload.Data)
			payloadDigest[v.Payload.RecordNum].Write(v.Payload.Data)
			data[v.Payload.RecordNum] = append(data[v.Payload.RecordNum], v.Payload.Data...)
		case *contentwriterV1.WriteRequest_Meta:
			gotMeta = true
			for i, v2 := range v.Meta.RecordMeta {
				if size[v2.RecordNum] != v2.Size {
					return status.Error(codes.InvalidArgument, "Size mismatch")
				}

				pld := ""
				if blockDigest[v2.RecordNum] != nil {
					pld = fmt.Sprintf("sha1:%x", blockDigest[v2.RecordNum].Sum(nil))
				}
				if pld != v2.BlockDigest {
					return status.Error(codes.InvalidArgument, "Block digest mismatch")
				}

				// Fake error
				if strings.Contains(v.Meta.TargetUri, "cwerr") {
					return status.Error(codes.InvalidArgument, "Fake error")
				}

				idxString := strconv.Itoa(int(i))
				records[i] = &contentwriterV1.WriteResponseMeta_RecordMeta{
					RecordNum:           i,
					CollectionFinalName: "collection_0",
					StorageRef:          "storageRef_" + idxString,
					WarcId:              "warcid_" + idxString + "_" + fmt.Sprintf("%x", payloadDigest[v2.RecordNum].Sum(nil)),
					PayloadDigest:       fmt.Sprintf("sha1:%x", payloadDigest[v2.RecordNum].Sum(nil)),
					BlockDigest:         v2.BlockDigest,
				}

				if v2.Type == contentwriterV1.RecordType_RESPONSE {
					records[i].RevisitReferenceId = "revisit_0"
					records[i].Type = contentwriterV1.RecordType_REVISIT
				} else {
					records[i].Type = contentwriterV1.RecordType_REQUEST
				}
			}
		case *contentwriterV1.WriteRequest_Cancel:
			gotCancel = true
		default:
			panic(fmt.Sprintf("UNKNOWN REQ type %T\n", v))
		}
	}
}
