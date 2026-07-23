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

	browsercontrollerV2 "github.com/NationalLibraryOfNorway/veidemann/api/browsercontroller/v2"
	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	dnsresolverV1 "github.com/NationalLibraryOfNorway/veidemann/api/dnsresolver/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/serviceconnections"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/tracing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

/**
 * Server mocks
 */
type GrpcServiceMock struct {
	dnsresolverV1.UnimplementedDnsResolverServer
	contentwriterV1.UnimplementedContentWriterServer
	browsercontrollerV2.UnimplementedBrowserControllerServer

	dnsOpts               *serviceconnections.ConnectionOptions
	contentWriterOpts     *serviceconnections.ConnectionOptions
	browserControllerOpts *serviceconnections.ConnectionOptions
	lis                   *bufconn.Listener
	mu                    sync.Mutex
	Requests              Requests
	DoneBC                chan bool
	DoneCW                chan bool
	contextDialer         grpc.DialOption
	Server                *grpc.Server
	ClientConn            *serviceconnections.Connections
}

// ConnectionOption configures how we parse a URL.
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

func WithExternalBrowserController(option *serviceconnections.ConnectionOptions) MockOption {
	return newFuncMockOption(func(c *GrpcServiceMock) {
		c.browserControllerOpts = option
	})
}

func WithExternalContentWriter(option *serviceconnections.ConnectionOptions) MockOption {
	return newFuncMockOption(func(c *GrpcServiceMock) {
		c.contentWriterOpts = option
	})
}

func WithExternalDns(option *serviceconnections.ConnectionOptions) MockOption {
	return newFuncMockOption(func(c *GrpcServiceMock) {
		c.dnsOpts = option
	})
}

type Requests struct {
	BrowserControllerRequests []*BrowserControllerRequest
	DnsResolverRequests       []*dnsresolverV1.ResolveRequest
	ContentWriterRequests     []*contentwriterV1.WriteRequest
}

type BrowserControllerRequest struct {
	RegisterResource *browsercontrollerV2.RegisterResourceRequest
	CompleteResource *browsercontrollerV2.CompleteResourceRequest
}

func NewGrpcServiceMock(opts ...MockOption) *GrpcServiceMock {
	m := &GrpcServiceMock{
		lis: bufconn.Listen(bufSize),
	}

	for _, opt := range opts {
		opt.apply(m)
	}

	m.contextDialer = grpc.WithContextDialer(m.bufDialer)

	m.Server = grpc.NewServer()

	if m.dnsOpts == nil {
		dnsresolverV1.RegisterDnsResolverServer(m.Server, m)
	}
	if m.contentWriterOpts == nil {
		contentwriterV1.RegisterContentWriterServer(m.Server, m)
	}
	if m.browserControllerOpts == nil {
		browsercontrollerV2.RegisterBrowserControllerServer(m.Server, m)
	}
	go func() {
		if err := m.Server.Serve(m.lis); err != nil {
			logger.LogWithComponent("MOCK:grpc").Fatalf("Server exited with error: %v", err)
		}
	}()

	dialOption := grpc.WithContextDialer(m.bufDialer)

	if m.contentWriterOpts == nil {
		m.contentWriterOpts = serviceconnections.NewConnectionOptions(
			"ContentWriter",
			serviceconnections.WithHost("passthrough://bufnet"),
			serviceconnections.WithDialOptions(dialOption, tracing.NewStatsHandler("ContentWriter", logger.DebugLevel)),
		)
	}
	if m.dnsOpts == nil {
		m.dnsOpts = serviceconnections.NewConnectionOptions(
			"DnsService",
			serviceconnections.WithHost("passthrough://bufnet"),
			serviceconnections.WithDialOptions(dialOption, tracing.NewStatsHandler("DnsService", logger.DebugLevel)),
		)
	}
	if m.browserControllerOpts == nil {
		m.browserControllerOpts = serviceconnections.NewConnectionOptions(
			"BrowserController",
			serviceconnections.WithHost("passthrough://bufnet"),
			serviceconnections.WithDialOptions(dialOption, tracing.NewStatsHandler("BrowserController", logger.DebugLevel)),
		)
	}

	m.ClientConn = serviceconnections.NewConnections(m.contentWriterOpts, m.dnsOpts, m.browserControllerOpts)

	err := m.ClientConn.Connect()
	if err != nil {
		logger.LogWithComponent("MOCK:grpc").Panicf("Could not connect to services: %v", err)
	}

	return m
}

func (s *GrpcServiceMock) Close() {
	s.ClientConn.Close()
	s.Server.GracefulStop()
	s.lis.Close()
}

func (s *GrpcServiceMock) Clear() {
	s.Requests = Requests{}
}

func (s *GrpcServiceMock) bufDialer(context.Context, string) (net.Conn, error) {
	return s.lis.Dial()
}

func (s *GrpcServiceMock) addBcRequest(r *BrowserControllerRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch {
	case r.RegisterResource != nil:
		logger.LogWithComponent("MOCK:BrowserController").Print("Register resource: ", r.RegisterResource)
	case r.CompleteResource != nil:
		logger.LogWithComponent("MOCK:BrowserController").Print("Complete resource: ", r.CompleteResource)
	default:
		panic("BUG: Invalid request")
	}

	s.Requests.BrowserControllerRequests = append(s.Requests.BrowserControllerRequests, r)
}

func (s *GrpcServiceMock) addBcRegisterRequest(r *browsercontrollerV2.RegisterResourceRequest) {
	s.addBcRequest(&BrowserControllerRequest{RegisterResource: r})
}

func (s *GrpcServiceMock) addBcCompleteRequest(r *browsercontrollerV2.CompleteResourceRequest) {
	s.addBcRequest(&BrowserControllerRequest{CompleteResource: r})
}

func (s *GrpcServiceMock) addDnsRequest(r *dnsresolverV1.ResolveRequest) {
	s.mu.Lock()

	logger.LogWithComponent("MOCK:DNSResolver").Print("Resolve ", r)

	s.Requests.DnsResolverRequests = append(s.Requests.DnsResolverRequests, r)
	s.mu.Unlock()
}

func (s *GrpcServiceMock) addCwRequest(r *contentwriterV1.WriteRequest) {
	s.mu.Lock()

	switch v := r.Value.(type) {
	case *contentwriterV1.WriteRequest_Payload:
		logger.LogWithComponent("MOCK:ContentWriter").
			Printf("Write: payload:<record_num:%d data:\"%s... (%d bytes)\" >\n",
				v.Payload.RecordNum, v.Payload.Data[0:5], len(v.Payload.Data))

	default:
		logger.LogWithComponent("MOCK:ContentWriter").Print("Write: ", r)
	}

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
	records := map[int32]*contentwriterV1.WriteResponseMeta_RecordMeta{}
	data := make(map[int32][]byte)
	size := make(map[int32]int64)
	gotMeta := false
	gotCancel := false
	blockDigest := make(map[int32]hash.Hash)

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

		if s.DoneCW == nil {
			s.DoneCW = make(chan bool, 200)
			go func() {
				<-server.Context().Done()
				s.DoneCW <- true
				s.DoneCW = nil
			}()
		}

		s.addCwRequest(request)

		switch v := request.Value.(type) {
		case *contentwriterV1.WriteRequest_ProtocolHeader:
			size[v.ProtocolHeader.RecordNum] = int64(len(v.ProtocolHeader.Data))
			blockDigest[v.ProtocolHeader.RecordNum] = sha1.New()
			blockDigest[v.ProtocolHeader.RecordNum].Write(v.ProtocolHeader.Data)
			data[v.ProtocolHeader.RecordNum] = v.ProtocolHeader.Data
		case *contentwriterV1.WriteRequest_Payload:
			size[v.Payload.RecordNum] += int64(len(v.Payload.Data))
			blockDigest[v.Payload.RecordNum].Write(v.Payload.Data)
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
					WarcId:              "warcid_" + idxString,
					PayloadDigest:       v2.PayloadDigest,
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

func browserControllerRegistered(request *browsercontrollerV2.RegisterResourceRequest) *browsercontrollerV2.ResourceRegistered {
	registered := &browsercontrollerV2.ResourceRegistered{
		CrawlExecutionId: request.CrawlExecutionId,
		JobExecutionId:   request.JobExecutionId,
		CollectionRef:    request.CollectionRef,
	}
	if registered.CrawlExecutionId == "" {
		registered.CrawlExecutionId = "eid"
	}
	if registered.JobExecutionId == "" {
		registered.JobExecutionId = "jid"
	}
	if registered.CollectionRef == nil {
		registered.CollectionRef = &configV1.ConfigRef{
			Kind: configV1.Kind_collection,
			Id:   "col1",
		}
	}
	return registered
}

// RegisterResource implements part of the browser controller server API
func (s *GrpcServiceMock) RegisterResource(ctx context.Context, in *browsercontrollerV2.RegisterResourceRequest) (*browsercontrollerV2.RegisterResourceReply, error) {
	s.addBcRegisterRequest(in)

	trimmedURI := strings.TrimSuffix(in.Uri, "/")
	switch {
	case strings.HasSuffix(trimmedURI, "blocked"):
		return &browsercontrollerV2.RegisterResourceReply{
			Result: &browsercontrollerV2.RegisterResourceReply_Cancel{Cancel: "Blocked by robots.txt"},
		}, nil
	case strings.HasSuffix(trimmedURI, "cancel"):
		return &browsercontrollerV2.RegisterResourceReply{
			Result: &browsercontrollerV2.RegisterResourceReply_Cancel{Cancel: "Cancelled by browser controller"},
		}, nil
	default:
		return &browsercontrollerV2.RegisterResourceReply{
			Result: &browsercontrollerV2.RegisterResourceReply_Registered{
				Registered: browserControllerRegistered(in),
			},
		}, nil
	}
}

// CompleteResource implements part of the browser controller server API
func (s *GrpcServiceMock) CompleteResource(ctx context.Context, in *browsercontrollerV2.CompleteResourceRequest) (*browsercontrollerV2.CompleteResourceReply, error) {
	s.addBcCompleteRequest(in)

	requestedURI := ""
	if in.GetCrawlLog() != nil {
		requestedURI = strings.TrimSuffix(in.GetCrawlLog().GetRequestedUri(), "/")
	}
	if strings.HasSuffix(requestedURI, "bccerr") {
		return nil, fmt.Errorf("browser controller error")
	}

	return &browsercontrollerV2.CompleteResourceReply{}, nil
}
