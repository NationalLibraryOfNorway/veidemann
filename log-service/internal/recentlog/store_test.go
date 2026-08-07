package recentlog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNewValidatesConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config Config
	}{
		{name: "empty path", config: Config{CrawlMaxEntries: 1, PageMaxEntries: 1}},
		{name: "invalid crawl limit", config: Config{Path: "logs.db", PageMaxEntries: 1}},
		{name: "invalid page limit", config: Config{Path: "logs.db", CrawlMaxEntries: 1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store, err := New(context.Background(), tc.config)
			if err == nil {
				_ = store.Close()
				t.Fatal("expected invalid config to fail")
			}
		})
	}
}

func TestStoreRoundTripAndNewestFirstPagination(t *testing.T) {
	store := newTestStore(t, 10, 10, nil)
	ctx := context.Background()

	crawlLogs := []*logV1.CrawlLog{
		crawlLog("crawl-1", "exec-crawl", "https://example.com/1"),
		crawlLog("crawl-2", "exec-crawl", "https://example.com/2"),
		crawlLog("crawl-3", "exec-crawl", "https://example.com/3"),
	}
	for _, crawlLog := range crawlLogs {
		if err := store.WriteCrawlLog(ctx, crawlLog); err != nil {
			t.Fatal(err)
		}
	}

	var page []*logV1.CrawlLog
	if err := store.ListCrawlLogsByExecutionID(ctx, "exec-crawl", 1, 1, func(crawlLog *logV1.CrawlLog) error {
		page = append(page, crawlLog)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(page) != 1 || page[0].GetWarcId() != "crawl-2" {
		t.Fatalf("expected second-newest crawl log, got %+v", page)
	}

	var byWarcID []*logV1.CrawlLog
	if err := store.ListCrawlLogsByWarcID(ctx, []string{"crawl-1", "missing", "crawl-3"}, func(crawlLog *logV1.CrawlLog) error {
		byWarcID = append(byWarcID, crawlLog)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(byWarcID) != 2 || byWarcID[0].GetWarcId() != "crawl-1" || byWarcID[1].GetWarcId() != "crawl-3" {
		t.Fatalf("expected requested WARC ID order, got %+v", byWarcID)
	}
	if !proto.Equal(crawlLogs[0], byWarcID[0]) {
		t.Fatalf("crawl log did not round trip:\nwant: %v\ngot:  %v", crawlLogs[0], byWarcID[0])
	}

	pageLog := &logV1.PageLog{
		WarcId:              "page-1",
		ExecutionId:         "exec-page",
		JobExecutionId:      "job-page",
		CollectionFinalName: "collection-page",
		Uri:                 "https://example.com/page",
		Outlink:             []string{"https://example.com/a", "https://example.com/b"},
		Resource: []*logV1.PageLog_Resource{{
			WarcId:      "resource-1",
			Uri:         "https://example.com/image.png",
			ContentType: "image/png",
			StatusCode:  500,
			FromCache:   true,
			Error:       &commonsV1.Error{Code: 42, Msg: "resource error", Detail: "detail"},
		}},
	}
	if err := store.WritePageLog(ctx, pageLog); err != nil {
		t.Fatal(err)
	}
	var actualPageLog *logV1.PageLog
	if err := store.ListPageLogsByWarcID(ctx, []string{"page-1"}, func(result *logV1.PageLog) error {
		actualPageLog = result
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(pageLog, actualPageLog) {
		t.Fatalf("page log did not round trip:\nwant: %v\ngot:  %v", pageLog, actualPageLog)
	}
}

func TestListRecentLogs(t *testing.T) {
	store := newTestStore(t, 10, 10, nil)
	ctx := context.Background()

	var emptyCrawlLogs []*logV1.CrawlLog
	if err := store.ListRecentCrawlLogs(ctx, 0, 0, func(crawlLog *logV1.CrawlLog) error {
		emptyCrawlLogs = append(emptyCrawlLogs, crawlLog)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var emptyPageLogs []*logV1.PageLog
	if err := store.ListRecentPageLogs(ctx, 0, 0, func(pageLog *logV1.PageLog) error {
		emptyPageLogs = append(emptyPageLogs, pageLog)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(emptyCrawlLogs) != 0 || len(emptyPageLogs) != 0 {
		t.Fatalf("expected empty recent queries, got crawl=%d page=%d", len(emptyCrawlLogs), len(emptyPageLogs))
	}

	for _, id := range []string{"crawl-1", "crawl-2", "crawl-3"} {
		if err := store.WriteCrawlLog(ctx, crawlLog(id, "exec-"+id, id)); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"page-1", "page-2"} {
		if err := store.WritePageLog(ctx, &logV1.PageLog{WarcId: id, ExecutionId: "exec-" + id}); err != nil {
			t.Fatal(err)
		}
	}

	var latestCrawlIDs []string
	if err := store.ListRecentCrawlLogs(ctx, 0, 0, func(crawlLog *logV1.CrawlLog) error {
		latestCrawlIDs = append(latestCrawlIDs, crawlLog.GetWarcId())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(latestCrawlIDs) != fmt.Sprint([]string{"crawl-3"}) {
		t.Fatalf("expected the latest crawl log by default, got %v", latestCrawlIDs)
	}

	var latestPageIDs []string
	if err := store.ListRecentPageLogs(ctx, 0, 0, func(pageLog *logV1.PageLog) error {
		latestPageIDs = append(latestPageIDs, pageLog.GetWarcId())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(latestPageIDs) != fmt.Sprint([]string{"page-2"}) {
		t.Fatalf("expected the latest page log by default, got %v", latestPageIDs)
	}

	var recentCrawlIDs []string
	if err := store.ListRecentCrawlLogs(ctx, 0, 2, func(crawlLog *logV1.CrawlLog) error {
		recentCrawlIDs = append(recentCrawlIDs, crawlLog.GetWarcId())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(recentCrawlIDs) != fmt.Sprint([]string{"crawl-3", "crawl-2"}) {
		t.Fatalf("expected two newest crawl logs, got %v", recentCrawlIDs)
	}

	var offsetCrawlIDs []string
	if err := store.ListRecentCrawlLogs(ctx, 1, 1, func(crawlLog *logV1.CrawlLog) error {
		offsetCrawlIDs = append(offsetCrawlIDs, crawlLog.GetWarcId())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(offsetCrawlIDs) != fmt.Sprint([]string{"crawl-2"}) {
		t.Fatalf("expected the second-newest crawl log, got %v", offsetCrawlIDs)
	}

	if err := store.WriteCrawlLog(ctx, crawlLog("crawl-2", "replacement-exec", "replacement")); err != nil {
		t.Fatal(err)
	}
	var replacement *logV1.CrawlLog
	if err := store.ListRecentCrawlLogs(ctx, 0, 0, func(crawlLog *logV1.CrawlLog) error {
		replacement = crawlLog
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if replacement.GetWarcId() != "crawl-2" || replacement.GetRequestedUri() != "replacement" {
		t.Fatalf("expected duplicate replacement to be newest, got %v", replacement)
	}
}

func TestIndependentRetentionAndDuplicatePromotion(t *testing.T) {
	store := newTestStore(t, 2, 1, nil)
	ctx := context.Background()

	for _, id := range []string{"crawl-1", "crawl-2", "crawl-3"} {
		if err := store.WriteCrawlLog(ctx, crawlLog(id, "exec", id)); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.WritePageLog(ctx, &logV1.PageLog{WarcId: "page-1", ExecutionId: "exec"}); err != nil {
		t.Fatal(err)
	}

	assertCrawlIDs(t, store, "exec", []string{"crawl-3", "crawl-2"})
	var pageIDs []string
	if err := store.ListPageLogsByExecutionID(ctx, "exec", 0, 0, func(pageLog *logV1.PageLog) error {
		pageIDs = append(pageIDs, pageLog.GetWarcId())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(pageIDs) != fmt.Sprint([]string{"page-1"}) {
		t.Fatalf("crawl retention evicted page logs: %v", pageIDs)
	}

	replacement := crawlLog("crawl-2", "exec", "replacement")
	if err := store.WriteCrawlLog(ctx, replacement); err != nil {
		t.Fatal(err)
	}
	assertCrawlIDs(t, store, "exec", []string{"crawl-2", "crawl-3"})
	var replaced *logV1.CrawlLog
	if err := store.ListCrawlLogsByWarcID(ctx, []string{"crawl-2"}, func(crawlLog *logV1.CrawlLog) error {
		replaced = crawlLog
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if replaced.GetRequestedUri() != "replacement" {
		t.Fatalf("expected replacement payload, got %q", replaced.GetRequestedUri())
	}

	if err := store.WritePageLog(ctx, &logV1.PageLog{WarcId: "page-2", ExecutionId: "exec"}); err != nil {
		t.Fatal(err)
	}
	var pages []string
	if err := store.ListPageLogsByExecutionID(ctx, "exec", 0, 0, func(pageLog *logV1.PageLog) error {
		pages = append(pages, pageLog.GetWarcId())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(pages) != fmt.Sprint([]string{"page-2"}) {
		t.Fatalf("expected independent page retention, got %v", pages)
	}
}

func TestPersistenceAndStartupPruning(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "logs.db")
	store, err := New(ctx, Config{Path: databasePath, CrawlMaxEntries: 3, PageMaxEntries: 3})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"crawl-1", "crawl-2", "crawl-3"} {
		if err := store.WriteCrawlLog(ctx, crawlLog(id, "exec", id)); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"page-1", "page-2", "page-3"} {
		if err := store.WritePageLog(ctx, &logV1.PageLog{WarcId: id, ExecutionId: "exec"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	store, err = New(ctx, Config{Path: databasePath, CrawlMaxEntries: 2, PageMaxEntries: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	assertCrawlIDs(t, store, "exec", []string{"crawl-3", "crawl-2"})
	var pages []string
	if err := store.ListPageLogsByExecutionID(ctx, "exec", 0, 0, func(pageLog *logV1.PageLog) error {
		pages = append(pages, pageLog.GetWarcId())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(pages) != fmt.Sprint([]string{"page-3"}) {
		t.Fatalf("expected page startup pruning to retain the newest row, got %v", pages)
	}
}

func TestSQLiteConfigurationAndMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	databasePath := filepath.Join(t.TempDir(), "logs.db")
	metrics := NewMetrics(registry, databasePath)
	store, err := New(context.Background(), Config{
		Path: databasePath, CrawlMaxEntries: 1, PageMaxEntries: 1, Metrics: metrics,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	pragmas := []struct {
		name string
		want int
	}{
		{name: "busy_timeout", want: busyTimeoutMilliseconds},
		{name: "cache_size", want: -cacheSizeKiB},
		{name: "mmap_size", want: 0},
		{name: "auto_vacuum", want: 2},
		{name: "synchronous", want: 1},
		{name: "wal_autocheckpoint", want: 1000},
	}
	for _, pragma := range pragmas {
		var actual int
		if err := store.db.QueryRow("PRAGMA " + pragma.name).Scan(&actual); err != nil {
			t.Fatal(err)
		}
		if actual != pragma.want {
			t.Fatalf("expected %s=%d, got %d", pragma.name, pragma.want, actual)
		}
	}
	var journalMode string
	if err := store.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if strings.ToLower(journalMode) != "wal" {
		t.Fatalf("expected WAL journal mode, got %q", journalMode)
	}
	if store.db.Stats().MaxOpenConnections != maxOpenConnections {
		t.Fatalf("expected %d pooled connections, got %d", maxOpenConnections, store.db.Stats().MaxOpenConnections)
	}

	if err := store.WriteCrawlLog(context.Background(), crawlLog("crawl-1", "exec", "one")); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteCrawlLog(context.Background(), crawlLog("crawl-2", "exec", "two")); err != nil {
		t.Fatal(err)
	}
	if got := testutil.ToFloat64(metrics.entries.WithLabelValues(CrawlLogType)); got != 1 {
		t.Fatalf("expected one retained crawl log metric, got %f", got)
	}
	if got := testutil.ToFloat64(metrics.evicted.WithLabelValues(CrawlLogType)); got != 1 {
		t.Fatalf("expected one evicted crawl log metric, got %f", got)
	}

	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	foundFileMetrics := 0
	payloadSamples := uint64(0)
	for _, family := range metricFamilies {
		switch family.GetName() {
		case "veidemann_recent_logs_database_file_bytes":
			foundFileMetrics = len(family.GetMetric())
			if foundFileMetrics != 3 {
				t.Fatalf("expected main, wal, and shm file metrics, got %d", foundFileMetrics)
			}
		case "veidemann_recent_logs_payload_bytes":
			for _, metric := range family.GetMetric() {
				for _, label := range metric.GetLabel() {
					if label.GetName() == "type" && label.GetValue() == CrawlLogType {
						payloadSamples = metric.GetHistogram().GetSampleCount()
					}
				}
			}
		}
	}
	if foundFileMetrics == 0 {
		t.Fatal("database file metrics were not collected")
	}
	if payloadSamples != 2 {
		t.Fatalf("expected two crawl payload observations, got %d", payloadSamples)
	}

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(databasePath + "-wal"); err == nil && info.Size() != 0 {
		t.Fatalf("expected close to truncate or remove WAL, got %d bytes", info.Size())
	} else if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := store.WriteCrawlLog(context.Background(), crawlLog("crawl-3", "exec", "three")); err == nil {
		t.Fatal("expected write to closed database to fail")
	}
	if got := testutil.ToFloat64(metrics.writeFailure.WithLabelValues(CrawlLogType)); got != 1 {
		t.Fatalf("expected one failed write metric, got %f", got)
	}
}

func TestListPropagatesEmitterError(t *testing.T) {
	store := newTestStore(t, 2, 2, nil)
	if err := store.WriteCrawlLog(context.Background(), crawlLog("crawl-1", "exec", "one")); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("send failed")
	err := store.ListCrawlLogsByExecutionID(context.Background(), "exec", 0, 0, func(*logV1.CrawlLog) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected emitter error, got %v", err)
	}
}

func newTestStore(t *testing.T, crawlLimit, pageLimit int64, metrics *Metrics) *Store {
	t.Helper()
	store, err := New(context.Background(), Config{
		Path:            filepath.Join(t.TempDir(), "logs.db"),
		CrawlMaxEntries: crawlLimit,
		PageMaxEntries:  pageLimit,
		Metrics:         metrics,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func crawlLog(warcID, executionID, requestedURI string) *logV1.CrawlLog {
	return &logV1.CrawlLog{
		WarcId:              warcID,
		ExecutionId:         executionID,
		JobExecutionId:      "job-1",
		CollectionFinalName: "collection-1",
		RequestedUri:        requestedURI,
		ResponseUri:         requestedURI,
		TimeStamp:           timestamppb.New(timestamppb.Now().AsTime().Truncate(time.Millisecond)),
		FetchTimeStamp:      timestamppb.New(timestamppb.Now().AsTime().Truncate(time.Millisecond)),
		Error:               &commonsV1.Error{Code: 7, Msg: "error", Detail: "detail"},
	}
}

func assertCrawlIDs(t *testing.T, store *Store, executionID string, expected []string) {
	t.Helper()
	var actual []string
	if err := store.ListCrawlLogsByExecutionID(context.Background(), executionID, 0, 0, func(crawlLog *logV1.CrawlLog) error {
		actual = append(actual, crawlLog.GetWarcId())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(actual) != fmt.Sprint(expected) {
		t.Fatalf("expected crawl log order %v, got %v", expected, actual)
	}
}

func BenchmarkStorePayloadSizes(b *testing.B) {
	for _, payloadSize := range []int{512, 1024, 4096, 16384} {
		b.Run(fmt.Sprintf("payload-%d", payloadSize), func(b *testing.B) {
			store, err := New(context.Background(), Config{
				Path:            filepath.Join(b.TempDir(), "logs.db"),
				CrawlMaxEntries: 1000000,
				PageMaxEntries:  1,
			})
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = store.Close() }()

			payload := strings.Repeat("x", payloadSize)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := store.WriteCrawlLog(context.Background(), &logV1.CrawlLog{
					WarcId:       fmt.Sprintf("crawl-%d", i),
					ExecutionId:  "exec",
					RequestedUri: payload,
				}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
