package recorderproxy

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	robotsV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator/v1"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func beginSpec(id string) *recorderV1.BeginHarvestRequest {
	return &recorderV1.BeginHarvestRequest{AdmissionToken: "fixture-token", ProxyId: 1, AttemptId: id, CrawlExecutionId: "crawl", CollectionRef: &configV1.ConfigRef{Id: "collection"}, RequestedUri: "https://example.com/"}
}
func testControl() (*Control, *RecorderProxy) {
	p := &RecorderProxy{nextToken: "fixture-token", id: 1, finalizationTimeout: time.Second}
	return NewControl(&testutil.LogWriterMock{}, p), p
}

func TestAttemptIdempotenceAndStaleRejection(t *testing.T) {
	c, p := testControl()
	req := beginSpec("attempt-000000000001")
	if _, err := c.BeginHarvest(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	if _, err := c.BeginHarvest(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	changed := proto.Clone(req).(*recorderV1.BeginHarvestRequest)
	changed.CrawlExecutionId = "other"
	if _, err := c.BeginHarvest(t.Context(), changed); err == nil {
		t.Fatal("accepted changed metadata")
	}
	next := beginSpec("attempt-000000000002")
	next.AdmissionToken = p.nextToken
	if _, err := c.BeginHarvest(t.Context(), next); err == nil {
		t.Fatal("reused open slot")
	}
	finish := &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: req.AttemptId}
	first, err := c.FinishPage(t.Context(), finish, nil)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := c.FinishPage(t.Context(), finish, nil)
	if err != nil || !proto.Equal(first, replay) {
		t.Fatal("replay changed", err)
	}
	if !p.attempt.reusable() {
		t.Fatal("not drained")
	}
	if _, err := c.BeginHarvest(t.Context(), next); err != nil {
		t.Fatal(err)
	}
	if _, err := c.BeginHarvest(t.Context(), req); err == nil {
		t.Fatal("stale begin accepted")
	}
	if _, err := c.FinishPage(t.Context(), finish, nil); err == nil {
		t.Fatal("stale finish accepted")
	}
}

func TestAttemptFinishSerializesAdmissionAndSurvivesCanceledReader(t *testing.T) {
	c, p := testControl()
	req := beginSpec("attempt-000000000001")
	_, _ = c.BeginHarvest(t.Context(), req)
	a := p.attempt
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Go(func() {
			rc := &rpcontext.RecordContext{}
			if a.admit(rc, func() { a.lifecycle.addRecord(rc) }) {
				rc.Complete(&logV1.CrawlLog{RequestedUri: "https://example.com/repeated", StatusCode: 200, WarcId: "confirmed"}, false, nil)
				rc.CloseFunc()
			}
		})
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	_, _ = c.FinishPage(canceled, &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: req.AttemptId}, nil)
	wg.Wait()
	reply, err := c.FinishPage(t.Context(), &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: req.AttemptId}, nil)
	if err != nil || reply.Failure != nil || reply.UriCount != int32(a.accepted) {
		t.Fatalf("reply=%v error=%v", reply, err)
	}
	if len(c.logs.(*testutil.LogWriterMock).CrawlLogs()) != int(a.accepted) {
		t.Fatal("lost persisted outcome")
	}
	if a.admit(&rpcontext.RecordContext{}, func() { t.Error("initialized after closure") }) {
		t.Fatal("late admission")
	}
}

func TestAttemptFinalizationTimeoutPreventsReuseButLateLogsArePersisted(t *testing.T) {
	c, p := testControl()
	p.finalizationTimeout = 10 * time.Millisecond
	req := beginSpec("attempt-000000000001")
	_, _ = c.BeginHarvest(t.Context(), req)
	a := p.attempt
	rc := &rpcontext.RecordContext{}
	a.admit(rc, func() { a.lifecycle.addRecord(rc) })
	a.failure = &commonsV1.Error{Code: -9998, Msg: "PRECLUDED_BY_ROBOTS"}
	reply, err := c.FinishPage(t.Context(), &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: req.AttemptId}, nil)
	if err != nil || reply.GetFailure().GetCode() != -5 || reply.Reusable {
		t.Fatalf("reply=%v error=%v", reply, err)
	}
	next := beginSpec("attempt-000000000002")
	next.AdmissionToken = p.nextToken
	if _, err := c.BeginHarvest(t.Context(), next); err == nil {
		t.Fatal("reused undrained slot")
	}
	rc.Complete(&logV1.CrawlLog{WarcId: "late"}, false, nil)
	rc.CloseFunc()
	if len(a.results) != 0 || len(c.logs.(*testutil.LogWriterMock).CrawlLogs()) != 1 {
		t.Fatal("late completion changed reply or lost log")
	}
	if _, err := c.BeginHarvest(t.Context(), next); err != nil {
		t.Fatalf("fully drained slot remained unavailable: %v", err)
	}
}

func TestRecorderPersistsFailuresWithoutWARC(t *testing.T) {
	for _, code := range []int32{-5011, -5, -1} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			c, p := testControl()
			_, _ = c.BeginHarvest(t.Context(), beginSpec("attempt-000000000001"))
			a := p.attempt
			rc := &rpcontext.RecordContext{}
			a.admit(rc, func() {})
			rc.Complete(&logV1.CrawlLog{StatusCode: code, Error: &commonsV1.Error{Code: code}}, false, &commonsV1.Error{Code: code})
			logs := c.logs.(*testutil.LogWriterMock).CrawlLogs()
			if len(logs) != 1 || logs[0].WarcId != "" || logs[0].GetError().GetCode() != code {
				t.Fatal(logs)
			}
			if (a.failure != nil) != (code != -5011) {
				t.Fatal(a.failure)
			}
		})
	}
}

func TestMainSelectionAndPersistence(t *testing.T) {
	for _, scenario := range []string{"fragment", "cached-then-recorded", "recorded-then-cached", "recorded-then-failed", "crawl-log-failure", "page-log-failure", "no-capture"} {
		t.Run(scenario, func(t *testing.T) {
			c, p := testControl()
			logs := c.logs.(*testutil.LogWriterMock)
			if scenario == "crawl-log-failure" {
				logs.CrawlError = errors.New("offline")
			}
			if scenario == "page-log-failure" {
				logs.PageError = errors.New("offline")
			}
			req := beginSpec("attempt-000000000001")
			req.Referrer = "https://example.org/"
			_, _ = c.BeginHarvest(t.Context(), req)
			a := p.attempt
			first, second := &rpcontext.RecordContext{MainDocument: true}, &rpcontext.RecordContext{MainDocument: true}
			a.admit(first, func() {})
			a.admit(second, func() {})
			latest := &logV1.CrawlLog{WarcId: "latest", RequestedUri: req.RequestedUri, StatusCode: 200, Size: 12}
			if scenario == "recorded-then-failed" {
				latest.WarcId = ""
				latest.Error = &commonsV1.Error{Code: -2}
			}
			cached := scenario == "recorded-then-cached"
			if cached {
				latest.WarcId = ""
			}
			// Complete out of order; admission order selects the document.
			second.Complete(latest, cached, latest.Error)
			first.Complete(&logV1.CrawlLog{WarcId: "first", RequestedUri: req.RequestedUri, StatusCode: 200, Size: 5}, scenario == "cached-then-recorded", nil)
			finish := &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: req.AttemptId, PageUri: new(req.RequestedUri + "#section")}
			links := []string{"https://example.com/next"}
			if scenario == "no-capture" {
				finish.PageUri = nil
				links = nil
			}
			reply, err := c.FinishPage(t.Context(), finish, links)
			if err != nil {
				t.Fatal(err)
			}
			if scenario != "no-capture" && (!proto.Equal(reply.Main, latest) || reply.MainCached != cached) {
				t.Fatal(reply)
			}
			wantFailure := scenario == "recorded-then-failed" || scenario == "crawl-log-failure" || scenario == "page-log-failure"
			if (reply.Failure != nil) != wantFailure {
				t.Fatal(reply)
			}
			wantPage := !cached && scenario != "recorded-then-failed" && scenario != "page-log-failure" && scenario != "no-capture"
			pages := logs.PageLogs()
			if (len(pages) == 1) != wantPage {
				t.Fatal(pages)
			}
			if wantPage && (pages[0].WarcId != "latest" || pages[0].Referrer != req.Referrer || len(pages[0].Resource) != 2 || len(pages[0].Outlink) != 1) {
				t.Fatal(pages)
			}
			before := len(logs.CrawlLogs())
			replay, err := c.FinishPage(t.Context(), finish, links)
			if err != nil || !proto.Equal(reply, replay) || len(logs.CrawlLogs()) != before || len(logs.PageLogs()) != len(pages) {
				t.Fatal("Finish repeated persistence")
			}
		})
	}
}

type waitingRobots struct {
	robotsV1.RobotsEvaluatorClient
	t        *testing.T
	deadline time.Time
	entered  chan *robotsV1.IsAllowedRequest
}

func (r waitingRobots) IsAllowed(ctx context.Context, req *robotsV1.IsAllowedRequest, _ ...grpc.CallOption) (*robotsV1.IsAllowedReply, error) {
	if deadline, ok := ctx.Deadline(); !ok || !deadline.Equal(r.deadline) {
		r.t.Error("robots evaluation must use the caller's deadline")
	}
	r.entered <- req
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestFinishDuringRobotsEvaluationAccountsForAdmittedTransfer(t *testing.T) {
	c, p := testControl()
	spec := beginSpec("attempt-000000000001")
	spec.Politeness = &configV1.ConfigObject{Spec: &configV1.ConfigObject_PolitenessConfig{PolitenessConfig: &configV1.PolitenessConfig{RobotsPolicy: configV1.PolitenessConfig_CUSTOM_ROBOTS_CLASSIC}}}
	spec.UserAgent = "crawler"
	if _, err := c.BeginHarvest(t.Context(), spec); err != nil {
		t.Fatal(err)
	}
	a := p.attempt
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	client, peer := net.Pipe()
	defer peer.Close()
	conn := WrapConn(client, "test", false)
	conn.cancelFunc = cancel
	if !a.lifecycle.addConnection(conn) {
		t.Fatal("connection admission failed")
	}
	rc := &rpcontext.RecordContext{}
	if !a.admit(rc, func() { a.lifecycle.addRecord(rc) }) {
		t.Fatal("transfer admission failed")
	}
	deadline, _ := ctx.Deadline()
	robots := waitingRobots{t: t, deadline: deadline, entered: make(chan *robotsV1.IsAllowedRequest, 1)}
	go func() {
		a.allowed(ctx, robots, "https://example.com/resource")
		rc.Complete(&logV1.CrawlLog{Error: &commonsV1.Error{Code: -5011}}, false, &commonsV1.Error{Code: -5011})
		rc.CloseFunc()
		a.lifecycle.removeConnection(conn)
	}()
	select {
	case req := <-robots.entered:
		if req.GetPoliteness().GetPolitenessConfig().GetRobotsPolicy() != configV1.PolitenessConfig_CUSTOM_ROBOTS || req.ExecutionId != spec.CrawlExecutionId || req.UserAgent != spec.UserAgent {
			t.Fatal(req)
		}
	case <-time.After(time.Second):
		t.Fatal("policy evaluation never entered")
	}
	reply, err := c.FinishPage(t.Context(), &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: spec.AttemptId}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Failure != nil || a.accepted != 1 || len(c.logs.(*testutil.LogWriterMock).CrawlLogs()) != 1 {
		t.Fatal(reply)
	}
	if spec.Politeness.GetPolitenessConfig().RobotsPolicy != configV1.PolitenessConfig_CUSTOM_ROBOTS_CLASSIC {
		t.Fatal("mutated attempt policy")
	}
}

func TestRobotsSubresourceExclusionPreservesPolicyOutcome(t *testing.T) {
	for _, main := range []bool{true, false} {
		a := &harvestAttempt{spec: beginSpec("attempt-000000000001"), logs: &testutil.LogWriterMock{}, timeout: time.Second}
		rc := &rpcontext.RecordContext{MainDocument: main}
		a.admit(rc, func() {})
		rc.Complete(&logV1.CrawlLog{RequestedUri: a.spec.RequestedUri, Error: &commonsV1.Error{Code: -9998}}, false, &commonsV1.Error{Code: -9998})
		if (a.failure != nil) != main {
			t.Fatalf("main=%v failure=%v", main, a.failure)
		}
	}
}

type finishUpload struct {
	recorderV1.RecorderControl_FinishHarvestServer
	ctx    context.Context
	frames []*recorderV1.FinishHarvestRequest
	end    func() error
}

func (s *finishUpload) Context() context.Context { return s.ctx }
func (s *finishUpload) Recv() (*recorderV1.FinishHarvestRequest, error) {
	if len(s.frames) == 0 {
		return nil, s.end()
	}
	msg := s.frames[0]
	s.frames = s.frames[1:]
	return msg, nil
}

func TestInvalidAndInterruptedUploadsLeaveAttemptOpen(t *testing.T) {
	c, p := testControl()
	spec := beginSpec("upload-attempt-0001")
	if _, err := c.BeginHarvest(t.Context(), spec); err != nil {
		t.Fatal(err)
	}
	header := func(page *string) *recorderV1.FinishHarvestRequest {
		return &recorderV1.FinishHarvestRequest{Value: &recorderV1.FinishHarvestRequest_Header{Header: &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: spec.AttemptId, PageUri: page}}}
	}
	link := &recorderV1.FinishHarvestRequest{Value: &recorderV1.FinishHarvestRequest_Outlink{Outlink: "https://example.com/link"}}
	for _, frames := range [][]*recorderV1.FinishHarvestRequest{nil, {link}, {header(nil), link}, {header(&spec.RequestedUri), header(nil)}, {header(&spec.RequestedUri), {}}, {header(new(""))}} {
		stream := &finishUpload{ctx: t.Context(), frames: frames, end: func() error { return io.EOF }}
		if err := c.FinishHarvest(stream); status.Code(err) != codes.InvalidArgument || p.attempt.closing {
			t.Fatalf("invalid upload finalized: %v", err)
		}
	}
	stream := &finishUpload{ctx: t.Context(), frames: []*recorderV1.FinishHarvestRequest{header(&spec.RequestedUri), link}, end: func() error { return context.Canceled }}
	if err := c.FinishHarvest(stream); !errors.Is(err, context.Canceled) || p.attempt.closing {
		t.Fatalf("interrupted upload finalized: %v", err)
	}
	stream.frames = []*recorderV1.FinishHarvestRequest{header(&spec.RequestedUri), link}
	stream.end = func() error {
		recovery, err := c.RecoverSlot(t.Context(), &recorderV1.RecoverSlotRequest{ProxyId: 1})
		if err != nil {
			t.Fatal(err)
		}
		next := beginSpec("replacement-attempt-0002")
		next.AdmissionToken = recovery.NextToken
		if _, err = c.BeginHarvest(t.Context(), next); err != nil {
			t.Fatal(err)
		}
		return io.EOF
	}
	if err := c.FinishHarvest(stream); status.Code(err) != codes.FailedPrecondition || p.attempt.closing {
		t.Fatalf("replacement finalized by old upload: %v", err)
	}
	if len(c.logs.(*testutil.LogWriterMock).PageLogs()) != 0 {
		t.Fatal("partial page persisted")
	}
}

func TestAdmissionTokensAcrossRestartsAndRepeatedHarvests(t *testing.T) {
	ctx := t.Context()
	proxy := func(id int32) *RecorderProxy { return &RecorderProxy{id: id, finalizationTimeout: time.Second} }
	newControl := func() *Control { return NewControl(&testutil.LogWriterMock{}, proxy(1), proxy(2)) }
	c := newControl()
	recoverToken := func(slot int32) string {
		r, err := c.RecoverSlot(ctx, &recorderV1.RecoverSlotRequest{ProxyId: slot})
		if err != nil || r.GetNextToken() == "" {
			t.Fatalf("recover: %v %v", r, err)
		}
		return r.NextToken
	}
	token := recoverToken(1)
	other := recoverToken(2)
	if token == other {
		t.Fatal("slots share admission token")
	}
	spec := beginSpec("token-attempt-0001")
	spec.AdmissionToken = token
	c = newControl()
	if _, err := c.BeginHarvest(ctx, spec); status.Code(err) != codes.FailedPrecondition {
		t.Fatal("recorder restart accepted old token", err)
	}
	token = recoverToken(1)
	spec.AdmissionToken = other
	for i := range 128 {
		spec.AttemptId = fmt.Sprintf("repeated-attempt-%d", i)
		if _, err := c.BeginHarvest(ctx, spec); status.Code(err) != codes.FailedPrecondition {
			t.Fatal("stale token accepted", err)
		}
		spec.AdmissionToken = token
		original, err := c.BeginHarvest(ctx, spec)
		if err != nil {
			t.Fatal(err)
		}
		replay, err := c.BeginHarvest(ctx, spec)
		if err != nil || !proto.Equal(original, replay) || original.NextToken == token {
			t.Fatal("lost Begin reply not recovered", err)
		}
		token = original.NextToken
		if _, err := c.FinishPage(ctx, &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: spec.AttemptId}, nil); err != nil {
			t.Fatal(err)
		}
	}
}
