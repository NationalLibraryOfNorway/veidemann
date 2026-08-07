package recentlog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

const (
	CrawlLogType = "crawl"
	PageLogType  = "page"

	crawlLogTable = "recent_crawl_logs"
	pageLogTable  = "recent_page_logs"

	maxOpenConnections      = 4
	cacheSizeKiB            = 2000
	busyTimeoutMilliseconds = 5000
	vacuumEvictionThreshold = 10000
)

type Config struct {
	Path            string
	CrawlMaxEntries int64
	PageMaxEntries  int64
	Metrics         *Metrics
}

type Store struct {
	db      *sql.DB
	metrics *Metrics

	writeMu              sync.Mutex
	counts               map[string]int64
	limits               map[string]int64
	evictionsSinceVacuum int64
}

func New(ctx context.Context, config Config) (*Store, error) {
	if strings.TrimSpace(config.Path) == "" {
		return nil, fmt.Errorf("recent log database path must not be empty")
	}
	if config.CrawlMaxEntries <= 0 {
		return nil, fmt.Errorf("recent crawl log max entries must be > 0")
	}
	if config.PageMaxEntries <= 0 {
		return nil, fmt.Errorf("recent page log max entries must be > 0")
	}

	databasePath, err := filepath.Abs(config.Path)
	if err != nil {
		return nil, fmt.Errorf("resolve recent log database path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o755); err != nil {
		return nil, fmt.Errorf("create recent log database directory: %w", err)
	}

	dsn := sqliteDSN(databasePath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open recent log database: %w", err)
	}
	db.SetMaxOpenConns(maxOpenConnections)
	db.SetMaxIdleConns(maxOpenConnections)

	store := &Store{
		db:      db,
		metrics: config.Metrics,
		counts:  make(map[string]int64, 2),
		limits: map[string]int64{
			CrawlLogType: config.CrawlMaxEntries,
			PageLogType:  config.PageMaxEntries,
		},
	}
	if err := store.initialize(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func sqliteDSN(path string) string {
	values := url.Values{}
	values.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", busyTimeoutMilliseconds))
	values.Add("_pragma", fmt.Sprintf("cache_size(-%d)", cacheSizeKiB))
	values.Add("_pragma", "mmap_size(0)")
	values.Add("_pragma", "synchronous(NORMAL)")
	values.Add("_pragma", "wal_autocheckpoint(1000)")
	return (&url.URL{Scheme: "file", Path: path, RawQuery: values.Encode()}).String()
}

func (s *Store) initialize(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping recent log database: %w", err)
	}
	statements := []string{
		"PRAGMA auto_vacuum=INCREMENTAL",
		"PRAGMA journal_mode=WAL",
		`CREATE TABLE IF NOT EXISTS recent_crawl_logs (
			ingestion_sequence INTEGER PRIMARY KEY AUTOINCREMENT,
			warc_id TEXT NOT NULL,
			execution_id TEXT NOT NULL,
			payload BLOB NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS recent_crawl_logs_warc_id_idx
			ON recent_crawl_logs(warc_id) WHERE warc_id <> ''`,
		`CREATE INDEX IF NOT EXISTS recent_crawl_logs_execution_sequence_idx
			ON recent_crawl_logs(execution_id, ingestion_sequence DESC)`,
		`CREATE TABLE IF NOT EXISTS recent_page_logs (
			ingestion_sequence INTEGER PRIMARY KEY AUTOINCREMENT,
			warc_id TEXT NOT NULL,
			execution_id TEXT NOT NULL,
			payload BLOB NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS recent_page_logs_warc_id_idx
			ON recent_page_logs(warc_id) WHERE warc_id <> ''`,
		`CREATE INDEX IF NOT EXISTS recent_page_logs_execution_sequence_idx
			ON recent_page_logs(execution_id, ingestion_sequence DESC)`,
		"PRAGMA user_version=1",
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("initialize recent log database: %w", err)
		}
	}
	return s.pruneAtStartup(ctx)
}

func (s *Store) pruneAtStartup(ctx context.Context) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin recent log startup prune: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	type result struct {
		logType string
		count   int64
		evicted int64
	}
	results := make([]result, 0, 2)
	for _, logType := range []string{CrawlLogType, PageLogType} {
		table := tableForType(logType)
		var count int64
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			return fmt.Errorf("count recent %s logs: %w", logType, err)
		}
		overflow := count - s.limits[logType]
		if overflow > 0 {
			if _, err := tx.ExecContext(ctx, pruneStatement(table), overflow); err != nil {
				return fmt.Errorf("prune recent %s logs: %w", logType, err)
			}
			count -= overflow
		}
		results = append(results, result{logType: logType, count: count, evicted: max(overflow, 0)})
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit recent log startup prune: %w", err)
	}
	for _, result := range results {
		s.counts[result.logType] = result.count
		s.metrics.setEntries(result.logType, result.count)
		s.metrics.addEvicted(result.logType, result.evicted)
	}
	return nil
}

func (s *Store) WriteCrawlLog(ctx context.Context, crawlLog *logV1.CrawlLog) (err error) {
	defer func() {
		if err != nil {
			s.metrics.recordWriteFailure(CrawlLogType)
		}
	}()
	if crawlLog == nil {
		return nil
	}
	return s.write(ctx, CrawlLogType, crawlLog.GetWarcId(), crawlLog.GetExecutionId(), crawlLog)
}

func (s *Store) WritePageLog(ctx context.Context, pageLog *logV1.PageLog) (err error) {
	defer func() {
		if err != nil {
			s.metrics.recordWriteFailure(PageLogType)
		}
	}()
	if pageLog == nil {
		return nil
	}
	return s.write(ctx, PageLogType, pageLog.GetWarcId(), pageLog.GetExecutionId(), pageLog)
}

func (s *Store) write(ctx context.Context, logType, warcID, executionID string, message proto.Message) error {
	payload, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal recent %s log: %w", logType, err)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	table := tableForType(logType)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin recent %s log write: %w", logType, err)
	}
	defer func() { _ = tx.Rollback() }()

	newCount := s.counts[logType]
	if warcID != "" {
		result, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE warc_id = ?", warcID)
		if err != nil {
			return fmt.Errorf("replace recent %s log: %w", logType, err)
		}
		deleted, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("count replaced recent %s logs: %w", logType, err)
		}
		newCount -= deleted
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO "+table+" (warc_id, execution_id, payload) VALUES (?, ?, ?)",
		warcID, executionID, payload,
	); err != nil {
		return fmt.Errorf("insert recent %s log: %w", logType, err)
	}
	newCount++

	overflow := newCount - s.limits[logType]
	if overflow > 0 {
		if _, err := tx.ExecContext(ctx, pruneStatement(table), overflow); err != nil {
			return fmt.Errorf("prune recent %s logs: %w", logType, err)
		}
		newCount -= overflow
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit recent %s log write: %w", logType, err)
	}

	s.counts[logType] = newCount
	s.metrics.setEntries(logType, newCount)
	s.metrics.addEvicted(logType, max(overflow, 0))
	s.metrics.observePayload(logType, len(payload))
	s.evictionsSinceVacuum += max(overflow, 0)
	if s.evictionsSinceVacuum >= vacuumEvictionThreshold {
		if _, err := s.db.ExecContext(ctx, "PRAGMA incremental_vacuum(256)"); err != nil {
			slog.Warn("Failed to incrementally vacuum recent log database", "error", err)
		} else {
			s.evictionsSinceVacuum = 0
		}
	}
	return nil
}

func pruneStatement(table string) string {
	return "DELETE FROM " + table + " WHERE ingestion_sequence IN (" +
		"SELECT ingestion_sequence FROM " + table + " ORDER BY ingestion_sequence ASC LIMIT ?" +
		")"
}

func tableForType(logType string) string {
	if logType == CrawlLogType {
		return crawlLogTable
	}
	return pageLogTable
}

func (s *Store) ListCrawlLogsByWarcID(ctx context.Context, warcIDs []string, emit func(*logV1.CrawlLog) error) error {
	found := 0
	for _, warcID := range warcIDs {
		count, err := listByWarcID(ctx, s.db, crawlLogTable, warcID, func(payload []byte) error {
			crawlLog := &logV1.CrawlLog{}
			if err := proto.Unmarshal(payload, crawlLog); err != nil {
				return fmt.Errorf("unmarshal recent crawl log: %w", err)
			}
			return emit(crawlLog)
		})
		if err != nil {
			return err
		}
		found += count
	}
	if found == 0 {
		return fmt.Errorf("crawl log not found")
	}
	return nil
}

func (s *Store) ListPageLogsByWarcID(ctx context.Context, warcIDs []string, emit func(*logV1.PageLog) error) error {
	found := 0
	for _, warcID := range warcIDs {
		count, err := listByWarcID(ctx, s.db, pageLogTable, warcID, func(payload []byte) error {
			pageLog := &logV1.PageLog{}
			if err := proto.Unmarshal(payload, pageLog); err != nil {
				return fmt.Errorf("unmarshal recent page log: %w", err)
			}
			return emit(pageLog)
		})
		if err != nil {
			return err
		}
		found += count
	}
	if found == 0 {
		return fmt.Errorf("page log not found")
	}
	return nil
}

func (s *Store) ListRecentCrawlLogs(ctx context.Context, offset, pageSize int, emit func(*logV1.CrawlLog) error) error {
	return listRecent(ctx, s.db, crawlLogTable, offset, pageSize, func(payload []byte) error {
		crawlLog := &logV1.CrawlLog{}
		if err := proto.Unmarshal(payload, crawlLog); err != nil {
			return fmt.Errorf("unmarshal recent crawl log: %w", err)
		}
		return emit(crawlLog)
	})
}

func (s *Store) ListRecentPageLogs(ctx context.Context, offset, pageSize int, emit func(*logV1.PageLog) error) error {
	return listRecent(ctx, s.db, pageLogTable, offset, pageSize, func(payload []byte) error {
		pageLog := &logV1.PageLog{}
		if err := proto.Unmarshal(payload, pageLog); err != nil {
			return fmt.Errorf("unmarshal recent page log: %w", err)
		}
		return emit(pageLog)
	})
}

func listRecent(ctx context.Context, db *sql.DB, table string, offset, pageSize int, emit func([]byte) error) error {
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 {
		pageSize = 1
	}
	rows, err := db.QueryContext(ctx,
		"SELECT payload FROM "+table+" ORDER BY ingestion_sequence DESC LIMIT ? OFFSET ?",
		pageSize, offset,
	)
	if err != nil {
		return fmt.Errorf("query recent logs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return fmt.Errorf("scan recent log: %w", err)
		}
		if err := emit(payload); err != nil {
			return err
		}
	}
	return rows.Err()
}

func listByWarcID(ctx context.Context, db *sql.DB, table, warcID string, emit func([]byte) error) (int, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT payload FROM "+table+" WHERE warc_id = ? ORDER BY ingestion_sequence DESC",
		warcID,
	)
	if err != nil {
		return 0, fmt.Errorf("query recent logs by warc ID: %w", err)
	}
	defer func() { _ = rows.Close() }()

	found := 0
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return found, fmt.Errorf("scan recent log: %w", err)
		}
		if err := emit(payload); err != nil {
			return found, err
		}
		found++
	}
	return found, rows.Err()
}

func (s *Store) ListCrawlLogsByExecutionID(ctx context.Context, executionID string, offset, pageSize int, emit func(*logV1.CrawlLog) error) error {
	return listByExecutionID(ctx, s.db, crawlLogTable, executionID, offset, pageSize, func(payload []byte) error {
		crawlLog := &logV1.CrawlLog{}
		if err := proto.Unmarshal(payload, crawlLog); err != nil {
			return fmt.Errorf("unmarshal recent crawl log: %w", err)
		}
		return emit(crawlLog)
	})
}

func (s *Store) ListPageLogsByExecutionID(ctx context.Context, executionID string, offset, pageSize int, emit func(*logV1.PageLog) error) error {
	return listByExecutionID(ctx, s.db, pageLogTable, executionID, offset, pageSize, func(payload []byte) error {
		pageLog := &logV1.PageLog{}
		if err := proto.Unmarshal(payload, pageLog); err != nil {
			return fmt.Errorf("unmarshal recent page log: %w", err)
		}
		return emit(pageLog)
	})
}

func listByExecutionID(ctx context.Context, db *sql.DB, table, executionID string, offset, pageSize int, emit func([]byte) error) error {
	if offset < 0 {
		offset = 0
	}
	limit := pageSize
	if limit <= 0 {
		limit = -1
	}
	rows, err := db.QueryContext(ctx,
		"SELECT payload FROM "+table+" WHERE execution_id = ? "+
			"ORDER BY ingestion_sequence DESC LIMIT ? OFFSET ?",
		executionID, limit, offset,
	)
	if err != nil {
		return fmt.Errorf("query recent logs by execution ID: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return fmt.Errorf("scan recent log: %w", err)
		}
		if err := emit(payload); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *Store) Close() error {
	var checkpointErr error
	if _, err := s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		checkpointErr = fmt.Errorf("checkpoint recent log database: %w", err)
	}
	return errors.Join(checkpointErr, s.db.Close())
}
