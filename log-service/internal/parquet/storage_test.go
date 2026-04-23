package parquet

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
)

func TestRotationAtLineBoundary(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := New(dir, 2)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 3 {
		if err := store.WriteCrawlLog(&logV1.CrawlLog{
			WarcId:              "warc-" + string(rune('a'+i)),
			ExecutionId:         "exec-1",
			CollectionFinalName: "collection-a",
		}); err != nil {
			t.Fatal(err)
		}
	}
	err = store.Close()
	if err != nil {
		t.Fatal(err)
	}

	files, err := filepath.Glob(filepath.Join(dir, tableCrawlLog, "collection-a", "*.parquet"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 parquet files after rotation, got %d", len(files))
	}
	for _, file := range files {
		if filepath.Ext(file) != ".parquet" {
			t.Fatalf("expected only finalized parquet files, got %q", file)
		}
	}
}

func TestExecutionIDPaginationAcrossFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := New(dir, 2)
	if err != nil {
		t.Fatal(err)
	}

	warcIDs := []string{"w1", "w2", "w3", "w4", "w5"}
	for _, id := range warcIDs {
		if err := store.WriteCrawlLog(&logV1.CrawlLog{
			WarcId:              id,
			ExecutionId:         "exec-pagination",
			CollectionFinalName: "collection-b",
		}); err != nil {
			t.Fatal(err)
		}
	}
	err = store.Close()
	if err != nil {
		t.Fatal(err)
	}

	page, err := store.ListCrawlLogsByExecutionID("exec-pagination", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 {
		t.Fatalf("expected page size 2, got %d", len(page))
	}
	if page[0].GetWarcId() != "w2" || page[1].GetWarcId() != "w3" {
		t.Fatalf("unexpected page content: got [%s, %s]", page[0].GetWarcId(), page[1].GetWarcId())
	}
}

func TestReadsIgnoreOpenFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := New(dir, 100)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.WriteCrawlLog(&logV1.CrawlLog{
		WarcId:              "w-open",
		ExecutionId:         "exec-open",
		CollectionFinalName: "collection-open",
	}); err != nil {
		t.Fatal(err)
	}

	page, err := store.ListCrawlLogsByExecutionID("exec-open", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 0 {
		t.Fatalf("expected no visible rows from open parquet files, got %d", len(page))
	}

	files, err := filepath.Glob(filepath.Join(dir, tableCrawlLog, "collection-open", "*.parquet"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no finalized parquet files before close, got %d", len(files))
	}

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	page, err = store.ListCrawlLogsByExecutionID("exec-open", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 1 || page[0].GetWarcId() != "w-open" {
		t.Fatalf("expected closed parquet file to become visible after close, got %+v", page)
	}
}

func TestPageLogOutlinksRoundTripLosslessly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := New(dir, 100)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"https://a.example/x,y", "", "https://b.example"}
	if err := store.WritePageLog(&logV1.PageLog{
		WarcId:              "page-1",
		ExecutionId:         "exec-outlinks",
		CollectionFinalName: "collection-outlinks",
		Outlink:             expected,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	pageLogs, err := store.ListPageLogsByWarcID([]string{"page-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(pageLogs) != 1 {
		t.Fatalf("expected one page log, got %d", len(pageLogs))
	}
	if len(pageLogs[0].GetOutlink()) != len(expected) {
		t.Fatalf("expected %d outlinks, got %d", len(expected), len(pageLogs[0].GetOutlink()))
	}
	for i := range expected {
		if pageLogs[0].GetOutlink()[i] != expected[i] {
			t.Fatalf("expected outlink %d to be %q, got %q", i, expected[i], pageLogs[0].GetOutlink()[i])
		}
	}
}

func TestDistinctCollectionNamesDoNotCollide(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := New(dir, 100)
	if err != nil {
		t.Fatal(err)
	}

	collections := []string{"a/b", "a_b"}
	for _, collection := range collections {
		if err := store.WriteCrawlLog(&logV1.CrawlLog{
			WarcId:              collection,
			ExecutionId:         "exec-collision",
			CollectionFinalName: collection,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(filepath.Join(dir, tableCrawlLog))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected two distinct collection directories, got %d", len(entries))
	}
}

func TestPostCloseHandoffReceivesFinalizedFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var (
		mu        sync.Mutex
		handedOff []FinalizedParquetFile
	)

	store, err := New(dir, 2, WithPostCloseHandoff(PostCloseHandoffFunc(func(file FinalizedParquetFile) error {
		if _, err := os.Stat(file.Path); err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		handedOff = append(handedOff, file)
		return nil
	})))
	if err != nil {
		t.Fatal(err)
	}

	for i := range 3 {
		if err := store.WriteCrawlLog(&logV1.CrawlLog{
			WarcId:              "handoff-" + string(rune('a'+i)),
			ExecutionId:         "exec-handoff",
			CollectionFinalName: "collection-handoff",
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(handedOff) != 2 {
		t.Fatalf("expected 2 handoff callbacks, got %d", len(handedOff))
	}
	if handedOff[0].Table != tableCrawlLog || handedOff[1].Table != tableCrawlLog {
		t.Fatalf("expected crawl log handoff entries, got %+v", handedOff)
	}
	if handedOff[0].Collection != "collection-handoff" || handedOff[1].Collection != "collection-handoff" {
		t.Fatalf("expected collection-handoff handoff entries, got %+v", handedOff)
	}
	if handedOff[0].RowCount != 2 || handedOff[1].RowCount != 1 {
		t.Fatalf("expected row counts [2 1], got [%d %d]", handedOff[0].RowCount, handedOff[1].RowCount)
	}
}
