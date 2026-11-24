package testutil

import (
	"context"
	"sync"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"google.golang.org/protobuf/proto"
)

// LogWriterMock retains the actual persisted outcomes, independent of Finish.
type LogWriterMock struct {
	mu                    sync.Mutex
	logs                  []*logV1.CrawlLog
	pages                 []*logV1.PageLog
	CrawlError, PageError error
}

func (w *LogWriterMock) WriteCrawlLog(_ context.Context, cl *logV1.CrawlLog) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.CrawlError != nil {
		return w.CrawlError
	}
	w.logs = append(w.logs, proto.Clone(cl).(*logV1.CrawlLog))
	return nil
}
func (w *LogWriterMock) WritePageLog(_ context.Context, page *logV1.PageLog) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.PageError != nil {
		return w.PageError
	}
	w.pages = append(w.pages, proto.Clone(page).(*logV1.PageLog))
	return nil
}
func (w *LogWriterMock) CrawlLogs() []*logV1.CrawlLog {
	w.mu.Lock()
	defer w.mu.Unlock()
	result := make([]*logV1.CrawlLog, len(w.logs))
	for i, cl := range w.logs {
		result[i] = proto.Clone(cl).(*logV1.CrawlLog)
	}
	return result
}
func (w *LogWriterMock) PageLogs() []*logV1.PageLog {
	w.mu.Lock()
	defer w.mu.Unlock()
	result := make([]*logV1.PageLog, len(w.pages))
	for i, page := range w.pages {
		result[i] = proto.Clone(page).(*logV1.PageLog)
	}
	return result
}
