package logservice

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/log-service/internal/parquet"
	"github.com/NationalLibraryOfNorway/veidemann/log-service/internal/recentlog"
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
	crawlWrites int
	pageWrites  int
	crawlErr    error
	pageErr     error
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
	return nil
}

func (s *recentStoreStub) ListCrawlLogsByExecutionID(context.Context, string, int, int, func(*logV1.CrawlLog) error) error {
	return nil
}

func (s *recentStoreStub) ListPageLogsByWarcID(context.Context, []string, func(*logV1.PageLog) error) error {
	return nil
}

func (s *recentStoreStub) ListPageLogsByExecutionID(context.Context, string, int, int, func(*logV1.PageLog) error) error {
	return nil
}
