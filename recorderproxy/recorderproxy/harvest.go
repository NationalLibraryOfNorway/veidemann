package recorderproxy

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	robotsV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator/v1"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	rperrors "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	url "github.com/nationallibraryofnorway/whatwg-url/url"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type attemptKey struct{}

type attemptConnection struct {
	attempt  *harvestAttempt
	verified atomic.Bool
}

// LogWriter is the persistence boundary of a recorder attempt.
type LogWriter interface {
	WriteCrawlLog(context.Context, *logV1.CrawlLog) error
	WritePageLog(context.Context, *logV1.PageLog) error
}

type transferOutcome struct {
	id           uint64
	log          *logV1.CrawlLog
	cached, main bool
	resourceType string
}

type harvestAttempt struct {
	mu        sync.Mutex
	spec      *recorderV1.BeginHarvestRequest
	started   time.Time
	lifecycle *lifecycleTracker
	timeout   time.Duration
	closing   bool
	done      chan struct{}
	results   []transferOutcome
	logs      LogWriter
	sealed    bool
	accepted  uint64
	discovery discoveryIndex
	failure   *commonsV1.Error
	reply     *recorderV1.FinishHarvestReply
}

// Control uses the same slot objects as the proxy listeners. No browser event
// participates in admission or completion.
type Control struct {
	recorderV1.UnimplementedRecorderControlServer
	proxies map[int32]*RecorderProxy
	logs    LogWriter
}

func NewControl(logs LogWriter, proxies ...*RecorderProxy) *Control {
	c := &Control{proxies: make(map[int32]*RecorderProxy), logs: logs}
	for _, p := range proxies {
		c.proxies[p.id] = p
	}
	return c
}

func (c *Control) slot(id int32) (*RecorderProxy, error) {
	p := c.proxies[id]
	if id <= 0 || p == nil {
		return nil, status.Error(codes.NotFound, "unknown harvest proxy slot")
	}
	return p, nil
}

// RecoverSlot is a startup barrier for the slot's exclusive controller. It does
// not discard live work or rewrite a published Finish reply. A timed-out caller
// can retry once late terminal work has drained.
func (c *Control) RecoverSlot(ctx context.Context, req *recorderV1.RecoverSlotRequest) (*recorderV1.RecoverSlotReply, error) {
	p, err := c.slot(req.GetProxyId())
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	if p.shuttingDown {
		p.mu.Unlock()
		return nil, status.Error(codes.Unavailable, "recorder is shutting down")
	}
	if p.nextToken == "" {
		p.nextToken = rand.Text()
	}
	reply := &recorderV1.RecoverSlotReply{NextToken: p.nextToken}
	a := p.attempt
	if a == nil {
		p.mu.Unlock()
		return reply, nil
	}
	a.startFinish(nil, nil)
	p.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, p.finalizationTimeout)
	defer cancel()
	select {
	case <-a.done:
	case <-ctx.Done():
		return nil, status.FromContextError(ctx.Err()).Err()
	}
	if err := a.lifecycle.wait(ctx); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	return reply, nil
}

func (c *Control) BeginHarvest(_ context.Context, req *recorderV1.BeginHarvestRequest) (*recorderV1.BeginHarvestReply, error) {
	p, err := c.slot(req.GetProxyId())
	if err != nil {
		return nil, err
	}
	if len(req.GetAttemptId()) < 16 || len(req.GetAttemptId()) > 128 || req.GetCrawlExecutionId() == "" || req.GetCollectionRef().GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "attempt identity and crawl metadata are required")
	}
	p.mu.Lock()
	var started *recorderV1.BeginHarvestRequest
	defer func() {
		p.mu.Unlock()
		if started != nil {
			harvestLog(started).WithField("phase", "recorder_begin").Info("Recording started")
		}
	}()
	if p.shuttingDown {
		return nil, status.Error(codes.Unavailable, "recorder is shutting down")
	}
	if a := p.attempt; a != nil {
		if a.spec.AttemptId == req.AttemptId {
			if !proto.Equal(a.spec, req) {
				return nil, status.Error(codes.AlreadyExists, "attempt metadata differs")
			}
			return &recorderV1.BeginHarvestReply{AttemptId: req.AttemptId, NextToken: p.nextToken}, nil
		}
		if !a.reusable() {
			return nil, status.Error(codes.FailedPrecondition, "previous attempt has not drained")
		}
	}
	if req.AdmissionToken == "" || req.AdmissionToken != p.nextToken {
		return nil, status.Error(codes.FailedPrecondition, "invalid slot admission token")
	}
	p.nextToken = rand.Text()
	timeout := p.finalizationTimeout
	if requested := time.Duration(req.FinalizationTimeoutMs) * time.Millisecond; requested > 0 && requested < timeout {
		timeout = requested
	}
	p.attempt = &harvestAttempt{
		started:   time.Now(),
		spec:      proto.Clone(req).(*recorderV1.BeginHarvestRequest),
		lifecycle: newLifecycleTracker(),
		timeout:   timeout,
		logs:      c.logs,
		done:      make(chan struct{}),
	}
	started = p.attempt.spec
	return &recorderV1.BeginHarvestReply{AttemptId: req.AttemptId, NextToken: p.nextToken}, nil
}

func (c *Control) FinishHarvest(stream recorderV1.RecorderControl_FinishHarvestServer) error {
	msg, err := stream.Recv()
	if err != nil && err != io.EOF {
		return err
	}
	header := msg.GetHeader()
	if header == nil || header.AttemptId == "" || (header.PageUri != nil && *header.PageUri == "") {
		return status.Error(codes.InvalidArgument, "expected finish header with attempt identity and optional page URI")
	}
	p, err := c.slot(header.ProxyId)
	if err != nil {
		return err
	}
	p.mu.Lock()
	attempt := p.attempt
	p.mu.Unlock()
	if attempt == nil || attempt.spec.AttemptId != header.AttemptId {
		return status.Error(codes.FailedPrecondition, "stale or unknown attempt")
	}
	var links []string
	for {
		msg, err = stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		link, ok := msg.Value.(*recorderV1.FinishHarvestRequest_Outlink)
		if !ok || header.PageUri == nil || link.Outlink == "" {
			return status.Error(codes.InvalidArgument, "expected outlink for captured page")
		}
		links = append(links, link.Outlink)
	}
	reply, err := p.finishPage(stream.Context(), header, links, attempt)
	if err != nil {
		return err
	}
	return stream.SendAndClose(reply)
}

// FinishPage finalizes an in-process caller's completed capture.
func (c *Control) FinishPage(ctx context.Context, header *recorderV1.FinishHarvestHeader, links []string) (*recorderV1.FinishHarvestReply, error) {
	p, err := c.slot(header.GetProxyId())
	if err != nil {
		return nil, err
	}
	if header.PageUri != nil && *header.PageUri == "" || header.PageUri == nil && len(links) > 0 || slices.Contains(links, "") {
		return nil, status.Error(codes.InvalidArgument, "invalid page capture")
	}
	// Local callers retain ownership of their inputs even if they cancel while
	// recorder-owned finalization is still running.
	return p.finishPage(ctx, proto.CloneOf(header), slices.Clone(links), nil)
}

func (p *RecorderProxy) finishPage(ctx context.Context, header *recorderV1.FinishHarvestHeader, links []string, uploading *harvestAttempt) (*recorderV1.FinishHarvestReply, error) {
	p.mu.Lock()
	a := p.attempt
	if a == nil || a.spec.AttemptId != header.AttemptId || (uploading != nil && uploading != a) {
		p.mu.Unlock()
		return nil, status.Error(codes.FailedPrecondition, "stale or unknown attempt")
	}
	if err := ctx.Err(); err != nil {
		p.mu.Unlock()
		return nil, status.FromContextError(err).Err()
	}
	a.startFinish(header.PageUri, links)
	p.mu.Unlock()
	select {
	case <-a.done:
		return a.reply, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// reusable describes current capacity; the published reply describes Finish's
// outcome at the time it completed and must never change after late drainage.
func (a *harvestAttempt) reusable() bool {
	select {
	case <-a.done:
		return a.lifecycle.drained()
	default:
		return false
	}
}

func (a *harvestAttempt) startFinish(page *string, links []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closing {
		return
	}
	a.closing = true
	go a.finish(page, links)
}

func (a *harvestAttempt) finish(page *string, links []string) {
	start := time.Now()
	harvestLog(a.spec).WithField("phase", "recorder_finalization").Info("Recording finalization started")
	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()
	for _, conn := range a.lifecycle.closeAndSnapshotConnections() {
		_ = conn.Close()
	}
	drainErr := a.lifecycle.wait(ctx)
	a.mu.Lock()
	a.sealed = true
	a.discovery = discoveryIndex{}
	accepted := a.accepted
	results := a.results
	a.results = nil // Only the small unary reply is retained for replay.
	reply := &recorderV1.FinishHarvestReply{Reusable: drainErr == nil, Failure: a.failure}
	if drainErr != nil {
		reply.Failure = &commonsV1.Error{Code: -5, Msg: "Recorder finalization timed out", Detail: drainErr.Error()}
	} else if uint64(len(results)) != a.accepted {
		reply.Failure = &commonsV1.Error{Code: -5, Msg: "Missing recorder outcome"}
	}
	a.mu.Unlock()

	resources := make([]*logV1.PageLog_Resource, 0, len(results))
	var mainID uint64
	var captureURI string
	if page != nil {
		captureURI = networkURI(*page)
	}
	for _, result := range results {
		cl := result.log
		if cl.WarcId != "" {
			reply.UriCount++
			reply.BytesDownloaded += cl.Size
		}
		if page != nil && result.main && result.id > mainID && networkURI(cl.RequestedUri) == captureURI {
			mainID, reply.Main, reply.MainCached = result.id, cl, result.cached
		}
		resources = append(resources, &logV1.PageLog_Resource{
			Uri:           cl.RequestedUri,
			Method:        cl.Method,
			ResourceType:  result.resourceType,
			Referrer:      cl.Referrer,
			FromCache:     result.cached,
			ContentType:   cl.ContentType,
			StatusCode:    cl.StatusCode,
			DiscoveryPath: cl.DiscoveryPath,
			WarcId:        cl.WarcId,
			Error:         cl.Error,
		})
	}
	pageLogStatus := "not_requested"
	if page != nil {
		pageLogStatus = "skipped"
	}
	main := reply.Main
	if page != nil && !reply.MainCached && main.GetWarcId() != "" && main.GetError() == nil && main.GetStatusCode() >= 200 {
		pageLog := &logV1.PageLog{
			WarcId:              main.WarcId,
			Uri:                 a.spec.RequestedUri,
			ExecutionId:         a.spec.CrawlExecutionId,
			JobExecutionId:      a.spec.JobExecutionId,
			CollectionFinalName: main.CollectionFinalName,
			Method:              main.Method,
			Referrer:            a.spec.Referrer,
			Resource:            resources,
			Outlink:             links,
		}
		if err := a.logs.WritePageLog(ctx, pageLog); err != nil {
			pageLogStatus = "failed"
			reply.Failure = persistenceFailure("page log", err)
		} else {
			pageLogStatus = "stored"
		}
	}
	a.logFinalization(reply, accepted, results, pageLogStatus, start)
	a.mu.Lock()
	a.reply = reply
	close(a.done)
	a.mu.Unlock()
}

func networkURI(raw string) string {
	raw, _, _ = strings.Cut(raw, "#")
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return parsed.String()
}

func persistenceFailure(kind string, err error) *commonsV1.Error {
	return &commonsV1.Error{Code: -5, Msg: "Failed to persist " + kind, Detail: err.Error()}
}

// admit runs before policy or writer setup, serialized with FinishHarvest.
func (a *harvestAttempt) admit(rc *rpcontext.RecordContext, initialize func()) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closing {
		return false
	}
	a.accepted++
	id := a.accepted
	rc.Harvest = a.spec
	a.wireDiscovery(rc, id)
	rc.Complete = func(cl *logV1.CrawlLog, cached bool, attemptFailure *commonsV1.Error) {
		// Persistence belongs to terminal work and is awaited by the lifecycle.
		// Never hold the attempt mutex across a service call.
		ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
		err := a.logs.WriteCrawlLog(ctx, cl)
		cancel()
		// The callback is owned by the idempotent terminal path, after ContentWriter
		// confirmation and crawl-log persistence. Log outside the attempt mutex.
		logRecordingOutcome(a.spec, rc.Session(), cl, cached, err)
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.sealed {
			return
		}
		a.results = append(a.results, transferOutcome{id: id, log: cl, cached: cached, main: rc.MainDocument, resourceType: rc.ResourceType})
		code := rperrors.ErrorCode(attemptFailure.GetCode())
		policyExclusion := code == rperrors.PrecludedByRobots && !rc.MainDocument
		if err != nil {
			a.failure = persistenceFailure("crawl log", err)
		} else if attemptFailure != nil && code != rperrors.CanceledByBrowser && !policyExclusion && a.failure == nil {
			a.failure = attemptFailure
		}
	}
	initialize()
	return true
}

func (a *harvestAttempt) allowed(ctx context.Context, client robotsV1.RobotsEvaluatorClient, uri string) bool {
	policy := a.spec.GetPoliteness()
	var resolved configV1.PolitenessConfig_RobotsPolicy
	switch policy.GetPolitenessConfig().GetRobotsPolicy() {
	case configV1.PolitenessConfig_OBEY_ROBOTS_CLASSIC:
		resolved = configV1.PolitenessConfig_OBEY_ROBOTS
	case configV1.PolitenessConfig_CUSTOM_ROBOTS_CLASSIC:
		resolved = configV1.PolitenessConfig_CUSTOM_ROBOTS
	case configV1.PolitenessConfig_CUSTOM_IF_MISSING_CLASSIC:
		resolved = configV1.PolitenessConfig_CUSTOM_IF_MISSING
	default:
		return true
	}
	if client == nil {
		return true
	}
	policy = proto.Clone(policy).(*configV1.ConfigObject)
	policy.GetPolitenessConfig().RobotsPolicy = resolved
	reply, err := client.IsAllowed(ctx, &robotsV1.IsAllowedRequest{
		JobExecutionId: a.spec.JobExecutionId,
		ExecutionId:    a.spec.CrawlExecutionId,
		Uri:            uri,
		UserAgent:      a.spec.UserAgent,
		Politeness:     policy,
		CollectionRef:  a.spec.CollectionRef,
	})
	if err != nil {
		logger.Log.WithError(err).Warn("Robots evaluator failed; preserving allow-on-error policy")
		return true
	}
	return reply.GetIsAllowed()
}

func (a *harvestAttempt) connectionFailure(err error) {
	if a == nil || err == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closing || a.failure != nil {
		return
	}
	failure := classifyFailure(err, FailureScopeConnection)
	if failure.Code == rperrors.CanceledByBrowser {
		return
	}
	a.failure = &commonsV1.Error{Code: int32(failure.Code), Msg: failure.Message, Detail: fmt.Sprint(err)}
}
