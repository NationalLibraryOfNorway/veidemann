package recentlog

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"google.golang.org/protobuf/proto"
)

// Keep the store full: benchmarks of an empty/growing database miss retention
// and checkpoint costs, which dominate the steady-state write workload.
func BenchmarkStoreAtRetention(b *testing.B) {
	const retainedEntries = 100000
	for _, writers := range []int{1, 6} {
		b.Run(fmt.Sprintf("writers-%d", writers), func(b *testing.B) {
			ctx := context.Background()
			store, err := New(ctx, Config{
				Path: filepath.Join(b.TempDir(), "logs.db"), CrawlMaxEntries: retainedEntries, PageMaxEntries: 1,
			})
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = store.Close() }()
			payload := strings.Repeat("x", 1024)
			seed, err := proto.Marshal(&logV1.CrawlLog{ExecutionId: "exec", RequestedUri: payload})
			if err != nil {
				b.Fatal(err)
			}
			// Seed in one transaction so setup does not spend time in the write
			// path being measured. Only measured writes need distinct payloads.
			if _, err := store.db.ExecContext(ctx, `
				WITH RECURSIVE entries(n) AS (
					VALUES(1) UNION ALL SELECT n + 1 FROM entries WHERE n < ?
				)
				INSERT INTO recent_crawl_logs (warc_id, execution_id, payload)
				SELECT printf('seed-%d', n), 'exec', ? FROM entries`, retainedEntries, seed); err != nil {
				b.Fatal(err)
			}
			store.counts[CrawlLogType] = retainedEntries

			latencies := make([]time.Duration, b.N)
			var wg sync.WaitGroup
			b.ResetTimer()
			for worker := range writers {
				wg.Go(func() {
					for i := worker; i < b.N; i += writers {
						start := time.Now()
						err := store.WriteCrawlLog(ctx, &logV1.CrawlLog{
							WarcId: fmt.Sprintf("crawl-%d", i), ExecutionId: "exec", RequestedUri: payload,
						})
						latencies[i] = time.Since(start)
						if err != nil {
							b.Error(err)
							return
						}
					}
				})
			}
			wg.Wait()
			b.StopTimer()
			slices.Sort(latencies)
			b.ReportMetric(float64(latencies[(len(latencies)-1)*99/100].Microseconds())/1000, "p99-ms")
		})
	}
}
