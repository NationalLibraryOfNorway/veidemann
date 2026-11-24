//go:build integration

// Copyright 2026 National Library of Norway.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/controller"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/frontier"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type fixtureFrontier struct {
	pending  []*frontierV1.PageHarvestSpec
	reports  []*frontier.RenderResult
	failures []error
}

func (f *fixtureFrontier) GetNextPage(context.Context) (*frontierV1.PageHarvestSpec, error) {
	if len(f.pending) == 0 {
		return nil, io.EOF
	}
	next := f.pending[0]
	f.pending = f.pending[1:]
	return next, nil
}
func (f *fixtureFrontier) PageCompleted(_ context.Context, _ *frontierV1.PageHarvestSpec, r *frontier.RenderResult, err error) error {
	f.reports = append(f.reports, r)
	f.failures = append(f.failures, err)
	return nil
}

type loseFirstFinish struct {
	recorderV1.RecorderControlClient
	finishes int
}
type discardReply struct {
	recorderV1.RecorderControl_FinishHarvestClient
}

func (s discardReply) CloseAndRecv() (*recorderV1.FinishHarvestReply, error) {
	if _, err := s.RecorderControl_FinishHarvestClient.CloseAndRecv(); err != nil {
		return nil, err
	}
	return nil, status.Error(codes.Unavailable, "injected lost reply after commit")
}
func (c *loseFirstFinish) FinishHarvest(ctx context.Context, opts ...grpc.CallOption) (recorderV1.RecorderControl_FinishHarvestClient, error) {
	s, err := c.RecorderControlClient.FinishHarvest(ctx, opts...)
	c.finishes++
	if c.finishes == 1 && err == nil {
		return discardReply{s}, nil
	}
	return s, err
}

func testLargeCaptureWorker(t *testing.T, conf *configV1.ConfigObject) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	conn, err := grpc.NewClient("localhost:7780", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	recorder := &loseFirstFinish{RecorderControlClient: recorderV1.NewRecorderControlClient(conn)}
	tokens, err := controller.RecoverSlots(ctx, recorder, 1, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	queue := &fixtureFrontier{}
	for _, path := range []string{"/large-links.html", "/linked.html?after-large"} {
		queue.pending = append(queue.pending, &frontierV1.PageHarvestSpec{QueuedUri: &frontierV1.QueuedUri{Uri: fixtureSiteHTTPBaseURL + path, DiscoveryPath: "L", JobExecutionId: "job", ExecutionId: "crawl"}, CrawlConfig: conf})
	}
	options := append([]session.Option{}, sessionOptions...)
	worker := controller.New(tokens, queue, time.Minute, time.Minute, append(options, session.WithRecorder(recorder))...)
	if err := worker.Run(ctx); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	if recorder.finishes != 3 || len(queue.reports) != 2 || queue.failures[0] != nil || queue.failures[1] != nil {
		t.Fatalf("finishes=%d reports=%d errors=%v", recorder.finishes, len(queue.reports), queue.failures)
	}
	uri := fixtureSiteHTTPBaseURL + "/large-links.html"
	recordedPages.mu.Lock()
	defer recordedPages.mu.Unlock()
	count := 0
	for _, page := range recordedPages.pages {
		if page.Uri != uri {
			continue
		}
		count++
		links := queue.reports[0].Outlinks
		if len(page.Outlink) != 4096 || len(links) != 4096 {
			t.Fatalf("recorder=%d frontier=%d", len(page.Outlink), len(links))
		}
		for i, link := range links {
			want := fmt.Sprintf("%s/candidate/%d/%s", fixtureSiteHTTPBaseURL, i, strings.Repeat("a", 1024))
			if page.Outlink[i] != want || link.Uri != want || link.ExecutionId != "crawl" || link.JobExecutionId != "job" || link.DiscoveryPath != "LL" || link.Referrer != uri || link.DiscoveredTimeStamp == nil || len(link.Cookies) != 1 || link.Cookies[0].Value != "large" {
				t.Fatalf("incorrect candidate %d", i)
			}
		}
	}
	if count != 1 {
		t.Fatalf("large harvest wrote %d page logs", count)
	}
}
