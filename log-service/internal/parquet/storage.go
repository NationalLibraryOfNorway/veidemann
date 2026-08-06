/*
 * Copyright 2021 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
