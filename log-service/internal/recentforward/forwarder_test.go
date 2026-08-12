package recentforward

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	server "github.com/NationalLibraryOfNorway/veidemann/log-service/internal/logservice"
	"github.com/NationalLibraryOfNorway/veidemann/log-service/internal/recentlog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type writerStub struct {
	started chan string
	release <-chan struct{}
	err     error

	mu              sync.Mutex
	crawlLogs       []*logV1.CrawlLog
	pageLogs        []*logV1.PageLog
	forwardedMarker []string
}

func (w *writerStub) WriteCrawlLog(ctx context.Context, crawlLog *logV1.CrawlLog) error {
	w.mu.Lock()
	w.crawlLogs = append(w.crawlLogs, crawlLog)
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		w.forwardedMarker = append(w.forwardedMarker, md.Get(ForwardedMetadataKey)...)
	}
	w.mu.Unlock()
	return w.wait(ctx, CrawlLogType)
}

func (w *writerStub) WritePageLog(ctx context.Context, pageLog *logV1.PageLog) error {
	w.mu.Lock()
	w.pageLogs = append(w.pageLogs, pageLog)
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		w.forwardedMarker = append(w.forwardedMarker, md.Get(ForwardedMetadataKey)...)
	}
	w.mu.Unlock()
	return w.wait(ctx, PageLogType)
}

func (w *writerStub) wait(ctx context.Context, logType string) error {
	if w.started != nil {
		w.started <- logType
	}
	if w.release != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.release:
		}
	}
	return w.err
}

func TestForwarderForwardsClonedLogsWithMarker(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)
	writer := &writerStub{started: make(chan string, 2)}
	forwarder, err := New(writer, Config{QueueSize: 2, Workers: 1, Timeout: time.Second, Metrics: metrics})
	if err != nil {
		t.Fatal(err)
	}

	crawlLog := &logV1.CrawlLog{WarcId: "crawl-1"}
	pageLog := &logV1.PageLog{WarcId: "page-1"}
	forwarder.EnqueueCrawlLog(crawlLog)
	forwarder.EnqueuePageLog(pageLog)
	crawlLog.WarcId = "mutated"
	pageLog.WarcId = "mutated"

	for range 2 {
		select {
		case <-writer.started:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for forwarded log")
		}
	}
	if err := forwarder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	writer.mu.Lock()
	defer writer.mu.Unlock()
	if got := writer.crawlLogs[0].GetWarcId(); got != "crawl-1" {
		t.Fatalf("expected cloned crawl log, got %q", got)
	}
	if got := writer.pageLogs[0].GetWarcId(); got != "page-1" {
		t.Fatalf("expected cloned page log, got %q", got)
	}
	if len(writer.forwardedMarker) != 2 || writer.forwardedMarker[0] != "true" || writer.forwardedMarker[1] != "true" {
		t.Fatalf("expected forwarded metadata marker, got %v", writer.forwardedMarker)
	}
	if got := testutil.ToFloat64(metrics.total.WithLabelValues(CrawlLogType, OutcomeSuccess)); got != 1 {
		t.Fatalf("expected one successful crawl forward, got %v", got)
	}
	if got := testutil.ToFloat64(metrics.total.WithLabelValues(PageLogType, OutcomeSuccess)); got != 1 {
		t.Fatalf("expected one successful page forward, got %v", got)
	}
	if got := testutil.ToFloat64(metrics.queueDepth); got != 0 {
		t.Fatalf("expected empty queue, got %v", got)
	}
}

func TestForwarderQueueFullDropsWithoutBlocking(t *testing.T) {
	release := make(chan struct{})
	writer := &writerStub{started: make(chan string, 1), release: release}
	metrics := NewMetrics(prometheus.NewRegistry())
	forwarder, err := New(writer, Config{QueueSize: 1, Workers: 1, Timeout: time.Second, Metrics: metrics})
	if err != nil {
		t.Fatal(err)
	}

	forwarder.EnqueueCrawlLog(&logV1.CrawlLog{WarcId: "active"})
	select {
	case <-writer.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for active forward")
	}
	forwarder.EnqueueCrawlLog(&logV1.CrawlLog{WarcId: "queued"})

	started := time.Now()
	forwarder.EnqueuePageLog(&logV1.PageLog{WarcId: "dropped"})
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("queue-full enqueue blocked for %s", elapsed)
	}
	if got := testutil.ToFloat64(metrics.total.WithLabelValues(PageLogType, OutcomeQueueFull)); got != 1 {
		t.Fatalf("expected one queue-full drop, got %v", got)
	}

	close(release)
	if err := forwarder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestForwarderTimeoutIsRecordedAsFailure(t *testing.T) {
	release := make(chan struct{})
	writer := &writerStub{started: make(chan string, 1), release: release}
	metrics := NewMetrics(prometheus.NewRegistry())
	forwarder, err := New(writer, Config{QueueSize: 1, Workers: 1, Timeout: 20 * time.Millisecond, Metrics: metrics})
	if err != nil {
		t.Fatal(err)
	}
	forwarder.EnqueueCrawlLog(&logV1.CrawlLog{WarcId: "timeout"})
	select {
	case <-writer.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for forward attempt")
	}
	if err := forwarder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := testutil.ToFloat64(metrics.total.WithLabelValues(CrawlLogType, OutcomeFailure)); got != 1 {
		t.Fatalf("expected one failed forward, got %v", got)
	}
}

func TestForwarderShutdownDeadlineCancelsAndDrops(t *testing.T) {
	release := make(chan struct{})
	writer := &writerStub{started: make(chan string, 1), release: release}
	metrics := NewMetrics(prometheus.NewRegistry())
	forwarder, err := New(writer, Config{QueueSize: 2, Workers: 1, Timeout: time.Minute, Metrics: metrics})
	if err != nil {
		t.Fatal(err)
	}
	forwarder.EnqueueCrawlLog(&logV1.CrawlLog{WarcId: "active"})
	select {
	case <-writer.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for active forward")
	}
	forwarder.EnqueuePageLog(&logV1.PageLog{WarcId: "queued"})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err = forwarder.Close(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected shutdown deadline, got %v", err)
	}
	if got := testutil.ToFloat64(metrics.total.WithLabelValues(CrawlLogType, OutcomeShutdownDrop)); got != 1 {
		t.Fatalf("expected active crawl shutdown drop, got %v", got)
	}
	if got := testutil.ToFloat64(metrics.total.WithLabelValues(PageLogType, OutcomeShutdownDrop)); got != 1 {
		t.Fatalf("expected queued page shutdown drop, got %v", got)
	}
	if got := testutil.ToFloat64(metrics.queueDepth); got != 0 {
		t.Fatalf("expected empty queue after shutdown, got %v", got)
	}
}

func TestForwarderRecordsWriterFailure(t *testing.T) {
	metrics := NewMetrics(prometheus.NewRegistry())
	forwarder, err := New(&writerStub{err: errors.New("unavailable")}, Config{
		QueueSize: 1,
		Workers:   1,
		Timeout:   time.Second,
		Metrics:   metrics,
	})
	if err != nil {
		t.Fatal(err)
	}
	forwarder.EnqueuePageLog(&logV1.PageLog{WarcId: "page"})
	if err := forwarder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := testutil.ToFloat64(metrics.total.WithLabelValues(PageLogType, OutcomeFailure)); got != 1 {
		t.Fatalf("expected one failed page forward, got %v", got)
	}
}

func TestForwarderWritesQueryableLogsThroughGRPC(t *testing.T) {
	ctx := context.Background()
	store, err := recentlog.New(ctx, recentlog.Config{
		Path:            filepath.Join(t.TempDir(), "logs.db"),
		CrawlMaxEntries: 10,
		PageMaxEntries:  10,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	logV1.RegisterLogServer(grpcServer, server.NewRecent(store))
	go func() { _ = grpcServer.Serve(listener) }()
	t.Cleanup(grpcServer.Stop)

	conn, err := grpc.NewClient("passthrough:///recent-log-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	forwarder, err := New(NewLogWriter(logV1.NewLogClient(conn)), Config{
		QueueSize: 2,
		Workers:   1,
		Timeout:   time.Second,
		Metrics:   NewMetrics(prometheus.NewRegistry()),
	})
	if err != nil {
		t.Fatal(err)
	}
	wantTimestamp := timestamppb.New(time.Date(2026, 3, 4, 5, 6, 7, 8000000, time.UTC))
	forwarder.EnqueueCrawlLog(&logV1.CrawlLog{
		WarcId:              "crawl-grpc",
		ExecutionId:         "execution-grpc",
		CollectionFinalName: "collection-grpc",
		TimeStamp:           wantTimestamp,
	})
	forwarder.EnqueuePageLog(&logV1.PageLog{
		WarcId:              "page-grpc",
		ExecutionId:         "execution-grpc",
		CollectionFinalName: "collection-grpc",
		Resource:            []*logV1.PageLog_Resource{{WarcId: "resource-grpc"}},
		Outlink:             []string{"https://example.test/outlink"},
	})
	if err := forwarder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	var crawlLogs []*logV1.CrawlLog
	if err := store.ListCrawlLogsByExecutionID(ctx, "execution-grpc", 0, 10, func(log *logV1.CrawlLog) error {
		crawlLogs = append(crawlLogs, log)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(crawlLogs) != 1 || crawlLogs[0].GetWarcId() != "crawl-grpc" {
		t.Fatalf("expected forwarded crawl log, got %v", crawlLogs)
	}
	if !crawlLogs[0].GetTimeStamp().AsTime().Equal(wantTimestamp.AsTime()) {
		t.Fatalf("expected writer timestamp %s, got %s", wantTimestamp.AsTime(), crawlLogs[0].GetTimeStamp().AsTime())
	}

	var pageLogs []*logV1.PageLog
	if err := store.ListPageLogsByExecutionID(ctx, "execution-grpc", 0, 10, func(log *logV1.PageLog) error {
		pageLogs = append(pageLogs, log)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(pageLogs) != 1 || pageLogs[0].GetWarcId() != "page-grpc" {
		t.Fatalf("expected forwarded page log, got %v", pageLogs)
	}
	if len(pageLogs[0].GetResource()) != 1 || pageLogs[0].GetResource()[0].GetWarcId() != "resource-grpc" {
		t.Fatalf("expected forwarded page resource, got %v", pageLogs[0].GetResource())
	}
	if len(pageLogs[0].GetOutlink()) != 1 || pageLogs[0].GetOutlink()[0] != "https://example.test/outlink" {
		t.Fatalf("expected forwarded page outlink, got %v", pageLogs[0].GetOutlink())
	}
}
