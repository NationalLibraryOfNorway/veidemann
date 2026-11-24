// Copyright 2026 National Library of Norway.
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/frontier"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/recorderproxy"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

type reportedHarvests struct {
	frontierV1.UnimplementedFrontierServer
	reports chan []*frontierV1.QueuedUri
}

func (f *reportedHarvests) PageCompleted(stream frontierV1.Frontier_PageCompletedServer) error {
	var links []*frontierV1.QueuedUri
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if msg.GetError() != nil {
			return fmt.Errorf("failed harvest: %v", msg.GetError())
		}
		if link := msg.GetOutlink(); link != nil {
			links = append(links, link)
		}
	}
	f.reports <- links
	return stream.SendAndClose(&emptypb.Empty{})
}

type lostFinishReply struct{ grpc.ServerStream }

func (lostFinishReply) SendMsg(any) error {
	return status.Error(codes.Unavailable, "lost finish reply")
}

func TestLargeCaptureReplaysFinishAndReportsOnce(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	services := testutil.NewGrpcServiceMock()
	defer services.Close()
	proxy := recorderproxy.NewRecorderProxy(1, services.ContentWriterClient, services.DnsResolverClient, "")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		if err := proxy.Serve(listener); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("serve recorder proxy: %v", err)
		}
	}()
	defer func() { _ = proxy.Shutdown(ctx) }()
	proxyURL, _ := url.Parse("http://" + listener.Addr().String())
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	defer transport.CloseIdleConnections()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "page")
	}))
	defer origin.Close()
	var finishes atomic.Int32
	server := grpc.NewServer(grpc.StreamInterceptor(func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if info.FullMethod == recorderV1.RecorderControl_FinishHarvest_FullMethodName && finishes.Add(1) == 1 {
			ss = lostFinishReply{ss}
		}
		return handler(srv, ss)
	}))
	reports := &reportedHarvests{reports: make(chan []*frontierV1.QueuedUri, 3)}
	recorderV1.RegisterRecorderControlServer(server, recorderproxy.NewControl(services, proxy))
	frontierV1.RegisterFrontierServer(server, reports)
	wire := bufconn.Listen(1 << 20)
	go func() {
		if err := server.Serve(wire); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("serve gRPC: %v", err)
		}
	}()
	defer server.Stop()
	conn, err := grpc.NewClient("passthrough:///harvest", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return wire.DialContext(ctx) }))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	client := recorderV1.NewRecorderControlClient(conn)
	reporter := frontier.New(frontierV1.NewFrontierClient(conn))
	recovery, err := client.RecoverSlot(ctx, &recorderV1.RecoverSlotRequest{ProxyId: 1})
	if err != nil {
		t.Fatal(err)
	}
	token := recovery.NextToken
	for harvest := range 2 {
		uri := fmt.Sprintf("%s/page%d", origin.URL, harvest)
		request := &frontierV1.QueuedUri{Uri: uri, ExecutionId: "crawl", JobExecutionId: "job", DiscoveryPath: "L"}
		spec := &recorderV1.BeginHarvestRequest{ProxyId: 1, AttemptId: fmt.Sprintf("large-page-attempt-%d", harvest), AdmissionToken: token, CollectionRef: &configV1.ConfigRef{Id: "collection"}, RequestedUri: uri, CrawlExecutionId: request.ExecutionId, JobExecutionId: request.JobExecutionId, DiscoveryPath: request.DiscoveryPath}
		begin, err := client.BeginHarvest(ctx, spec)
		if err != nil {
			t.Fatal(err)
		}
		token = begin.NextToken
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
		req.Header.Set(constants.HeaderAttemptId, spec.AttemptId)
		req.Header.Set(constants.HeaderMainDocument, spec.AttemptId)
		resp, err := (&http.Client{Transport: transport}).Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if err != nil || resp.StatusCode != 200 {
			t.Fatalf("fetch: %d %v", resp.StatusCode, err)
		}
		capture := artifactBundle{uri: uri + "#current", committedURL: uri, cookies: []*frontierV1.Cookie{{Name: "session", Value: "fixture"}}}
		if harvest == 0 {
			// Each consecutive pair is identical. The URLs alone exceed 4 MiB.
			for i := range 4096 {
				capture.outlinks = append(capture.outlinks, fmt.Sprintf("%s/%d/%s", uri, i/2, strings.Repeat("a", 1024)))
			}
		}
		sess := New(1, WithRecorder(client))
		sess.attemptID = spec.AttemptId
		reply, err := sess.finishHarvest(ctx, &capture.committedURL, capture.outlinks)
		if err != nil || reply.GetMain().GetWarcId() == "" {
			t.Fatalf("finish: %v %v", reply, err)
		}
		result := capture.renderResult(request, reply, time.Second)
		err = reporter.PageCompleted(ctx, &frontierV1.PageHarvestSpec{QueuedUri: request, SessionToken: "frontier"}, result, nil)
		if err != nil {
			t.Fatal(err)
		}
		received := <-reports.reports
		pages := services.PageLogs()
		if len(pages) != harvest+1 || !slices.Equal(pages[harvest].Outlink, capture.outlinks) || len(received) != len(capture.outlinks) {
			t.Fatal("page log or Frontier candidate list differs")
		}
		for i, link := range received {
			if link.Uri != capture.outlinks[i] || link.ExecutionId != "crawl" || link.JobExecutionId != "job" || link.Referrer != capture.uri || link.DiscoveryPath != "LL" || link.DiscoveredTimeStamp == nil || len(link.Cookies) != 1 || link.Cookies[0].Value != "fixture" {
				t.Fatalf("Frontier candidate %d differs", i)
			}
		}
	}
	if finishes.Load() != 3 || len(reports.reports) != 0 {
		t.Fatal("unexpected Finish or Frontier report count")
	}
}
