package logservice

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/log-service/internal/parquet"
	"github.com/NationalLibraryOfNorway/veidemann/log-service/internal/recentlog"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestParquetFailureSkipsRecentStore(t *testing.T) {
	wantErr := errors.New("parquet failed")
	archive := &archiveStub{crawlErr: wantErr, pageErr: wantErr}
	recent := &recentStoreStub{}

	err := writeCrawlLog(context.Background(), archive, recent, &logV1.CrawlLog{
		WarcId:         "crawl-1",
		FetchTimeStamp: timestamppb.Now(),
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected parquet error, got %v", err)
	}
	if recent.crawlWrites != 0 {
		t.Fatalf("expected recent write to be skipped, got %d calls", recent.crawlWrites)
	}

	err = writePageLog(context.Background(), archive, recent, &logV1.PageLog{WarcId: "page-1"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected parquet error, got %v", err)
	}
	if recent.pageWrites != 0 {
		t.Fatalf("expected recent write to be skipped, got %d calls", recent.pageWrites)
	}
}

func TestRecentStoreFailureDoesNotFailArchivedWrite(t *testing.T) {
	archive := &archiveStub{}
	recent := &recentStoreStub{crawlErr: errors.New("sqlite failed"), pageErr: errors.New("sqlite failed")}

	if err := writeCrawlLog(context.Background(), archive, recent, &logV1.CrawlLog{
		WarcId:         "crawl-1",
		ExecutionId:    "exec-1",
		FetchTimeStamp: timestamppb.Now(),
	}); err != nil {
		t.Fatalf("recent crawl write failure must be best effort: %v", err)
	}
	if err := writePageLog(context.Background(), archive, recent, &logV1.PageLog{
		WarcId:      "page-1",
		ExecutionId: "exec-1",
	}); err != nil {
		t.Fatalf("recent page write failure must be best effort: %v", err)
	}
	if archive.crawlWrites != 1 || archive.pageWrites != 1 {
		t.Fatalf("expected both archive writes, got crawl=%d page=%d", archive.crawlWrites, archive.pageWrites)
	}
	if recent.crawlWrites != 1 || recent.pageWrites != 1 {
		t.Fatalf("expected both recent writes, got crawl=%d page=%d", recent.crawlWrites, recent.pageWrites)
	}
}

func TestRecentLogsAreQueryableBeforeParquetRotation(t *testing.T) {
	ctx := context.Background()
	parquetDir := t.TempDir()
	archive, err := parquet.New(parquetDir, 100)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = archive.Close() })
	recent, err := recentlog.New(ctx, recentlog.Config{
		Path:            filepath.Join(t.TempDir(), "logs.db"),
		CrawlMaxEntries: 10,
		PageMaxEntries:  10,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = recent.Close() })

	crawlLog := &logV1.CrawlLog{
		WarcId:              "crawl-1",
		ExecutionId:         "exec-1",
		CollectionFinalName: "collection-1",
		FetchTimeStamp:      timestamppb.Now(),
	}
	if err := writeCrawlLog(ctx, archive, recent, crawlLog); err != nil {
		t.Fatal(err)
	}
	pageLog := &logV1.PageLog{
		WarcId:              "page-1",
		ExecutionId:         "exec-1",
		CollectionFinalName: "collection-1",
		Resource:            []*logV1.PageLog_Resource{{WarcId: "resource-1"}},
	}
	if err := writePageLog(ctx, archive, recent, pageLog); err != nil {
		t.Fatal(err)
	}

	finalizedFiles, err := filepath.Glob(filepath.Join(parquetDir, "*", "*", "*.parquet"))
	if err != nil {
		t.Fatal(err)
	}
	if len(finalizedFiles) != 0 {
		t.Fatalf("expected no finalized parquet files before rotation, got %v", finalizedFiles)
	}

	var crawlIDs []string
	if err := recent.ListCrawlLogsByExecutionID(ctx, "exec-1", 0, 10, func(crawlLog *logV1.CrawlLog) error {
		crawlIDs = append(crawlIDs, crawlLog.GetWarcId())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(crawlIDs) != 1 || crawlIDs[0] != "crawl-1" {
		t.Fatalf("expected immediate crawl log read, got %v", crawlIDs)
	}

	var pageIDs []string
	if err := recent.ListPageLogsByExecutionID(ctx, "exec-1", 0, 10, func(pageLog *logV1.PageLog) error {
		pageIDs = append(pageIDs, pageLog.GetWarcId())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(pageIDs) != 1 || pageIDs[0] != "page-1" {
		t.Fatalf("expected immediate page log read, got %v", pageIDs)
	}
}

func TestParquetOnlyLogsAreInvisibleToRecentReads(t *testing.T) {
	ctx := context.Background()
	archive, err := parquet.New(t.TempDir(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := archive.WriteCrawlLog(&logV1.CrawlLog{
		WarcId:              "archived-crawl",
		ExecutionId:         "archived-execution",
		CollectionFinalName: "collection-1",
	}); err != nil {
		t.Fatal(err)
	}
	if err := archive.WritePageLog(&logV1.PageLog{
		WarcId:              "archived-page",
		ExecutionId:         "archived-execution",
		CollectionFinalName: "collection-1",
	}); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}

	recent, err := recentlog.New(ctx, recentlog.Config{
		Path:            filepath.Join(t.TempDir(), "logs.db"),
		CrawlMaxEntries: 10,
		PageMaxEntries:  10,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = recent.Close() })

	crawlCount := 0
	if err := recent.ListCrawlLogsByExecutionID(ctx, "archived-execution", 0, 0, func(*logV1.CrawlLog) error {
		crawlCount++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	pageCount := 0
	if err := recent.ListPageLogsByExecutionID(ctx, "archived-execution", 0, 0, func(*logV1.PageLog) error {
		pageCount++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if crawlCount != 0 || pageCount != 0 {
		t.Fatalf("expected archived-only logs to be invisible, got crawl=%d page=%d", crawlCount, pageCount)
	}
}

func TestUnfilteredListRequestsUseRecentQueries(t *testing.T) {
	recent := &recentStoreStub{}
	server := New(&archiveStub{}, recent)

	crawlStream := &crawlListStream{ctx: context.Background()}
	if err := server.ListCrawlLogs(&logV1.CrawlLogListRequest{Offset: 2, PageSize: 3}, crawlStream); err != nil {
		t.Fatal(err)
	}
	if len(recent.crawlRecentReads) != 1 || recent.crawlRecentReads[0] != (recentRead{offset: 2, pageSize: 3}) {
		t.Fatalf("expected unfiltered crawl query, got %+v", recent.crawlRecentReads)
	}

	pageStream := &pageListStream{ctx: context.Background()}
	if err := server.ListPageLogs(&logV1.PageLogListRequest{}, pageStream); err != nil {
		t.Fatal(err)
	}
	if len(recent.pageRecentReads) != 1 || recent.pageRecentReads[0] != (recentRead{}) {
		t.Fatalf("expected unfiltered page query, got %+v", recent.pageRecentReads)
	}
}

func TestListFilterPrecedenceAndUnsupportedTemplates(t *testing.T) {
	recent := &recentStoreStub{}
	server := New(&archiveStub{}, recent)
	crawlStream := &crawlListStream{ctx: context.Background()}
	pageStream := &pageListStream{ctx: context.Background()}

	if err := server.ListCrawlLogs(&logV1.CrawlLogListRequest{
		WarcId:        []string{"crawl-1"},
		QueryTemplate: &logV1.CrawlLog{JobExecutionId: "ignored"},
	}, crawlStream); err != nil {
		t.Fatal(err)
	}
	if recent.crawlWarcReads != 1 || len(recent.crawlRecentReads) != 0 {
		t.Fatalf("expected WARC-ID query precedence, got warc=%d recent=%d", recent.crawlWarcReads, len(recent.crawlRecentReads))
	}

	if err := server.ListPageLogs(&logV1.PageLogListRequest{
		QueryTemplate: &logV1.PageLog{ExecutionId: "exec-1", JobExecutionId: "ignored"},
	}, pageStream); err != nil {
		t.Fatal(err)
	}
	if recent.pageExecutionReads != 1 || len(recent.pageRecentReads) != 0 {
		t.Fatalf("expected execution-ID query precedence, got execution=%d recent=%d", recent.pageExecutionReads, len(recent.pageRecentReads))
	}

	if err := server.ListCrawlLogs(&logV1.CrawlLogListRequest{
		QueryTemplate: &logV1.CrawlLog{JobExecutionId: "unsupported"},
	}, crawlStream); err == nil {
		t.Fatal("expected an unsupported crawl-log template to fail")
	}
	if err := server.ListPageLogs(&logV1.PageLogListRequest{
		QueryTemplate: &logV1.PageLog{JobExecutionId: "unsupported"},
	}, pageStream); err == nil {
		t.Fatal("expected an unsupported page-log template to fail")
	}
	if len(recent.crawlRecentReads) != 0 || len(recent.pageRecentReads) != 0 {
		t.Fatalf("unsupported templates must not run unfiltered queries: crawl=%d page=%d",
			len(recent.crawlRecentReads), len(recent.pageRecentReads))
	}
}

type archiveStub struct {
	crawlWrites int
	pageWrites  int
	crawlErr    error
	pageErr     error
}

func (s *archiveStub) WriteCrawlLog(*logV1.CrawlLog) error {
	s.crawlWrites++
	return s.crawlErr
}

func (s *archiveStub) WritePageLog(*logV1.PageLog) error {
	s.pageWrites++
	return s.pageErr
}

type recentStoreStub struct {
	crawlWrites        int
	pageWrites         int
	crawlErr           error
	pageErr            error
	crawlWarcReads     int
	pageExecutionReads int
	crawlRecentReads   []recentRead
	pageRecentReads    []recentRead
}

type recentRead struct {
	offset   int
	pageSize int
}

func (s *recentStoreStub) WriteCrawlLog(context.Context, *logV1.CrawlLog) error {
	s.crawlWrites++
	return s.crawlErr
}

func (s *recentStoreStub) WritePageLog(context.Context, *logV1.PageLog) error {
	s.pageWrites++
	return s.pageErr
}

func (s *recentStoreStub) ListCrawlLogsByWarcID(context.Context, []string, func(*logV1.CrawlLog) error) error {
	s.crawlWarcReads++
	return nil
}

func (s *recentStoreStub) ListCrawlLogsByExecutionID(context.Context, string, int, int, func(*logV1.CrawlLog) error) error {
	return nil
}

func (s *recentStoreStub) ListRecentCrawlLogs(_ context.Context, offset, pageSize int, _ func(*logV1.CrawlLog) error) error {
	s.crawlRecentReads = append(s.crawlRecentReads, recentRead{offset: offset, pageSize: pageSize})
	return nil
}

func (s *recentStoreStub) ListPageLogsByWarcID(context.Context, []string, func(*logV1.PageLog) error) error {
	return nil
}

func (s *recentStoreStub) ListPageLogsByExecutionID(context.Context, string, int, int, func(*logV1.PageLog) error) error {
	s.pageExecutionReads++
	return nil
}

func (s *recentStoreStub) ListRecentPageLogs(_ context.Context, offset, pageSize int, _ func(*logV1.PageLog) error) error {
	s.pageRecentReads = append(s.pageRecentReads, recentRead{offset: offset, pageSize: pageSize})
	return nil
}

type crawlListStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *crawlListStream) Context() context.Context {
	return s.ctx
}

func (s *crawlListStream) Send(*logV1.CrawlLog) error {
	return nil
}

type pageListStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *pageListStream) Context() context.Context {
	return s.ctx
}

func (s *pageListStream) Send(*logV1.PageLog) error {
	return nil
}
