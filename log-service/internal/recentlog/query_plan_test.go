package recentlog

import (
	"context"
	"strings"
	"testing"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
)

// Every non-empty WARC-ID write does a duplicate lookup, even for a new ID.
// A table scan here makes ingestion cost grow with the entire retained window.
func TestWarcIDQueriesUseIndex(t *testing.T) {
	store := newTestStore(t, 3, 3, nil)
	for _, table := range []string{crawlLogTable, pageLogTable} {
		for _, query := range []string{
			deleteByWarcIDStatement(table),
			selectByWarcIDStatement(table, "missing"),
		} {
			t.Run(query, func(t *testing.T) {
				rows, err := store.db.Query("EXPLAIN QUERY PLAN "+query, "missing")
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = rows.Close() }()
				var plan []string
				for rows.Next() {
					var id, parent, unused int
					var detail string
					if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
						t.Fatal(err)
					}
					plan = append(plan, detail)
				}
				if err := rows.Err(); err != nil {
					t.Fatal(err)
				}
				joined := strings.Join(plan, "\n")
				if !strings.Contains(joined, table+"_warc_id_idx") || strings.Contains(joined, "SCAN "+table) {
					t.Fatalf("expected indexed WARC-ID lookup, got:\n%s", joined)
				}
			})
		}
	}
}

func TestEmptyWarcIDsRemainIndependent(t *testing.T) {
	store := newTestStore(t, 3, 3, nil)
	ctx := context.Background()
	for _, uri := range []string{"first", "second"} {
		if err := store.WriteCrawlLog(ctx, crawlLog("", "exec", uri)); err != nil {
			t.Fatal(err)
		}
		if err := store.WritePageLog(ctx, &logV1.PageLog{ExecutionId: "exec", Uri: uri}); err != nil {
			t.Fatal(err)
		}
	}
	// Non-empty IDs must continue to deduplicate alongside the anonymous logs.
	for range 2 {
		if err := store.WriteCrawlLog(ctx, crawlLog("crawl", "exec", "named")); err != nil {
			t.Fatal(err)
		}
		if err := store.WritePageLog(ctx, &logV1.PageLog{WarcId: "page", ExecutionId: "exec", Uri: "named"}); err != nil {
			t.Fatal(err)
		}
	}
	var crawlURIs, pageURIs []string
	if err := store.ListCrawlLogsByWarcID(ctx, []string{""}, func(log *logV1.CrawlLog) error {
		crawlURIs = append(crawlURIs, log.GetRequestedUri())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.ListPageLogsByWarcID(ctx, []string{""}, func(log *logV1.PageLog) error {
		pageURIs = append(pageURIs, log.GetUri())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(crawlURIs, ",") != "second,first" || strings.Join(pageURIs, ",") != "second,first" {
		t.Fatalf("expected both anonymous logs, newest first; crawl=%v page=%v", crawlURIs, pageURIs)
	}
	assertCrawlIDs(t, store, "exec", []string{"crawl", "", ""})
}
