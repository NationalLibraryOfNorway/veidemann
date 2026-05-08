package parquet

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
)

const (
	tableCrawlLog = "crawl_log"
	tablePageLog  = "page_log"
	tableResource = "resource"
)

type FinalizedParquetFile struct {
	Table       string
	Collection  string
	Path        string
	RowCount    int64
	FinalizedAt time.Time
}

type PostCloseHandoff interface {
	HandoffFinalizedFile(file FinalizedParquetFile) error
}

type PostCloseHandoffFunc func(file FinalizedParquetFile) error

func (f PostCloseHandoffFunc) HandoffFinalizedFile(file FinalizedParquetFile) error {
	if f == nil {
		return nil
	}
	return f(file)
}

type Option func(*Storage)

func WithPostCloseHandoff(handoff PostCloseHandoff) Option {
	return func(storage *Storage) {
		storage.handoff = handoff
	}
}

type Storage struct {
	baseDir         string
	maxLinesPerFile int64
	mu              sync.Mutex
	writers         map[string]*writerState
	handoff         PostCloseHandoff
}

func New(baseDir string, maxLinesPerFile int64, opts ...Option) (*Storage, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("parquet directory must not be empty")
	}
	if maxLinesPerFile <= 0 {
		return nil, fmt.Errorf("max lines per file must be > 0")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}

	storage := &Storage{
		baseDir:         baseDir,
		maxLinesPerFile: maxLinesPerFile,
		writers:         make(map[string]*writerState),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(storage)
		}
	}
	return storage, nil
}

func (s *Storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var multiErr error
	for _, writer := range s.writers {
		if err := s.closeWriterLocked(writer); err != nil {
			multiErr = errors.Join(multiErr, err)
		}
	}
	s.writers = make(map[string]*writerState)
	return multiErr
}

func (s *Storage) WriteCrawlLog(crawlLog *logV1.CrawlLog) error {
	if crawlLog == nil {
		return nil
	}
	return s.writeRow(tableCrawlLog, crawlLog.GetCollectionFinalName(), crawlLogToRow(crawlLog))
}

func (s *Storage) WritePageLog(pageLog *logV1.PageLog) error {
	if pageLog == nil {
		return nil
	}

	pageRow, err := pageLogToRow(pageLog)
	if err != nil {
		return err
	}
	if err := s.writeRow(tablePageLog, pageLog.GetCollectionFinalName(), pageRow); err != nil {
		return err
	}
	for _, resource := range pageLog.GetResource() {
		if err := s.writeRow(tableResource, pageLog.GetCollectionFinalName(), pageLogResourceToRow(pageLog.GetWarcId(), resource)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Storage) ListCrawlLogsByWarcID(warcIDs []string) ([]*logV1.CrawlLog, error) {
	if len(warcIDs) == 0 {
		return nil, nil
	}

	wanted := make(map[string]struct{}, len(warcIDs))
	for _, id := range warcIDs {
		wanted[id] = struct{}{}
	}

	rows, err := s.readAllCrawlRows()
	if err != nil {
		return nil, err
	}

	result := make([]*logV1.CrawlLog, 0, len(warcIDs))
	for i := range rows {
		if _, ok := wanted[rows[i].WarcID]; !ok {
			continue
		}
		result = append(result, crawlLogRowToProto(&rows[i]))
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("crawl log not found")
	}
	return result, nil
}

func (s *Storage) ListCrawlLogsByExecutionID(executionID string, offset, pageSize int) ([]*logV1.CrawlLog, error) {
	rows, err := s.readAllCrawlRows()
	if err != nil {
		return nil, err
	}

	filtered := make([]*crawlLogRow, 0)
	for i := range rows {
		if rows[i].ExecutionID == executionID {
			filtered = append(filtered, &rows[i])
		}
	}

	start, end := paginate(len(filtered), offset, pageSize)
	result := make([]*logV1.CrawlLog, 0, end-start)
	for _, row := range filtered[start:end] {
		result = append(result, crawlLogRowToProto(row))
	}
	return result, nil
}

func (s *Storage) ListPageLogsByWarcID(warcIDs []string) ([]*logV1.PageLog, error) {
	if len(warcIDs) == 0 {
		return nil, nil
	}

	wanted := make(map[string]struct{}, len(warcIDs))
	for _, id := range warcIDs {
		wanted[id] = struct{}{}
	}

	rows, err := s.readAllPageRows()
	if err != nil {
		return nil, err
	}
	resourcesByPage, err := s.readResourcesByPageID()
	if err != nil {
		return nil, err
	}

	result := make([]*logV1.PageLog, 0, len(warcIDs))
	for i := range rows {
		if _, ok := wanted[rows[i].WarcID]; !ok {
			continue
		}
		pageLog := pageLogRowToProto(&rows[i])
		pageLog.Resource = resourcesByPage[pageLog.GetWarcId()]
		result = append(result, pageLog)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("page log not found")
	}
	return result, nil
}

func (s *Storage) ListPageLogsByExecutionID(executionID string, offset, pageSize int) ([]*logV1.PageLog, error) {
	rows, err := s.readAllPageRows()
	if err != nil {
		return nil, err
	}
	resourcesByPage, err := s.readResourcesByPageID()
	if err != nil {
		return nil, err
	}

	filtered := make([]*pageLogRow, 0)
	for i := range rows {
		if rows[i].ExecutionID == executionID {
			filtered = append(filtered, &rows[i])
		}
	}

	start, end := paginate(len(filtered), offset, pageSize)
	result := make([]*logV1.PageLog, 0, end-start)
	for _, row := range filtered[start:end] {
		pageLog := pageLogRowToProto(row)
		pageLog.Resource = resourcesByPage[pageLog.GetWarcId()]
		result = append(result, pageLog)
	}
	return result, nil
}

func paginate(total, offset, pageSize int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		return total, total
	}
	if pageSize <= 0 {
		return offset, total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return offset, end
}
