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

package logservice

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	TableCrawlLog = "crawl_log"
	TablePageLog  = "page_log"
	TableResource = "resource"

	// ForwardedMetadataKey marks writes sent by a writer-mode log service. It
	// prevents the recent service from replacing the writer-assigned timestamp
	// or collecting the same ingestion metrics twice.
	ForwardedMetadataKey = "veidemann-log-forwarded"
)

type LogService interface {
	WriteCrawlLog(stream logV1.Log_WriteCrawlLogServer) error
	WritePageLog(stream logV1.Log_WritePageLogServer) error
	ListPageLogs(req *logV1.PageLogListRequest, stream logV1.Log_ListPageLogsServer) error
	ListCrawlLogs(req *logV1.CrawlLogListRequest, stream logV1.Log_ListCrawlLogsServer) error
}

type ArchiveWriter interface {
	WriteCrawlLog(crawlLog *logV1.CrawlLog) error
	WritePageLog(pageLog *logV1.PageLog) error
}

type RecentLogWriter interface {
	WriteCrawlLog(ctx context.Context, crawlLog *logV1.CrawlLog) error
	WritePageLog(ctx context.Context, pageLog *logV1.PageLog) error
}

type RecentLogStore interface {
	RecentLogWriter
	ListCrawlLogsByWarcID(ctx context.Context, warcIDs []string, emit func(*logV1.CrawlLog) error) error
	ListCrawlLogsByExecutionID(ctx context.Context, executionID string, offset, pageSize int, emit func(*logV1.CrawlLog) error) error
	ListRecentCrawlLogs(ctx context.Context, offset, pageSize int, emit func(*logV1.CrawlLog) error) error
	ListPageLogsByWarcID(ctx context.Context, warcIDs []string, emit func(*logV1.PageLog) error) error
	ListPageLogsByExecutionID(ctx context.Context, executionID string, offset, pageSize int, emit func(*logV1.PageLog) error) error
	ListRecentPageLogs(ctx context.Context, offset, pageSize int, emit func(*logV1.PageLog) error) error
}

type RecentForwarder interface {
	EnqueueCrawlLog(crawlLog *logV1.CrawlLog)
	EnqueuePageLog(pageLog *logV1.PageLog)
}

// LogServer is the backwards-compatible combined Parquet and SQLite server.
type LogServer struct {
	logV1.UnimplementedLogServer
	archive ArchiveWriter
	recent  RecentLogStore
}

// WriterServer archives writes and optionally schedules an asynchronous copy
// to a separate recent-log service. Read methods are deliberately unsupported.
type WriterServer struct {
	logV1.UnimplementedLogServer
	archive   ArchiveWriter
	forwarder RecentForwarder
}

// RecentServer stores writes in SQLite and serves recent-log reads.
type RecentServer struct {
	logV1.UnimplementedLogServer
	recent RecentLogStore
}

var (
	_ LogService = (*LogServer)(nil)
	_ LogService = (*WriterServer)(nil)
	_ LogService = (*RecentServer)(nil)
)

func New(archive ArchiveWriter, recent RecentLogStore) *LogServer {
	return &LogServer{archive: archive, recent: recent}
}

func NewWriter(archive ArchiveWriter, forwarder RecentForwarder) *WriterServer {
	return &WriterServer{archive: archive, forwarder: forwarder}
}

func NewRecent(recent RecentLogStore) *RecentServer {
	return &RecentServer{recent: recent}
}

func (l *LogServer) WriteCrawlLog(stream logV1.Log_WriteCrawlLogServer) error {
	return receiveCrawlLogs(stream, func(crawlLog *logV1.CrawlLog) error {
		normalizeCrawlLog(crawlLog)
		CollectCrawlLog(crawlLog)
		return writeCrawlLog(stream.Context(), l.archive, l.recent, crawlLog)
	})
}

func (l *LogServer) WritePageLog(stream logV1.Log_WritePageLogServer) error {
	pageLog, err := receivePageLog(stream)
	if err != nil {
		return err
	}
	CollectPageLog(pageLog)
	if err := writePageLog(stream.Context(), l.archive, l.recent, pageLog); err != nil {
		return fmt.Errorf("error writing page log: %w", err)
	}
	return stream.SendAndClose(&emptypb.Empty{})
}

func (l *LogServer) ListPageLogs(req *logV1.PageLogListRequest, stream logV1.Log_ListPageLogsServer) error {
	return listPageLogs(l.recent, req, stream)
}

func (l *LogServer) ListCrawlLogs(req *logV1.CrawlLogListRequest, stream logV1.Log_ListCrawlLogsServer) error {
	return listCrawlLogs(l.recent, req, stream)
}

func (l *WriterServer) WriteCrawlLog(stream logV1.Log_WriteCrawlLogServer) error {
	return receiveCrawlLogs(stream, func(crawlLog *logV1.CrawlLog) error {
		normalizeCrawlLog(crawlLog)
		CollectCrawlLog(crawlLog)
		if err := l.archive.WriteCrawlLog(crawlLog); err != nil {
			return err
		}
		if l.forwarder != nil {
			l.forwarder.EnqueueCrawlLog(crawlLog)
		}
		return nil
	})
}

func (l *WriterServer) WritePageLog(stream logV1.Log_WritePageLogServer) error {
	pageLog, err := receivePageLog(stream)
	if err != nil {
		return err
	}
	CollectPageLog(pageLog)
	if err := l.archive.WritePageLog(pageLog); err != nil {
		return fmt.Errorf("error writing page log: %w", err)
	}
	if l.forwarder != nil {
		l.forwarder.EnqueuePageLog(pageLog)
	}
	return stream.SendAndClose(&emptypb.Empty{})
}

func (l *WriterServer) ListPageLogs(*logV1.PageLogListRequest, logV1.Log_ListPageLogsServer) error {
	return status.Error(codes.Unimplemented, "log-service writer does not serve recent-log reads")
}

func (l *WriterServer) ListCrawlLogs(*logV1.CrawlLogListRequest, logV1.Log_ListCrawlLogsServer) error {
	return status.Error(codes.Unimplemented, "log-service writer does not serve recent-log reads")
}

func (l *RecentServer) WriteCrawlLog(stream logV1.Log_WriteCrawlLogServer) error {
	forwarded := isForwarded(stream.Context())
	return receiveCrawlLogs(stream, func(crawlLog *logV1.CrawlLog) error {
		if !forwarded {
			normalizeCrawlLog(crawlLog)
			CollectCrawlLog(crawlLog)
		}
		if err := l.recent.WriteCrawlLog(stream.Context(), crawlLog); err != nil {
			return fmt.Errorf("write crawl log to recent store: %w", err)
		}
		return nil
	})
}

func (l *RecentServer) WritePageLog(stream logV1.Log_WritePageLogServer) error {
	pageLog, err := receivePageLog(stream)
	if err != nil {
		return err
	}
	if !isForwarded(stream.Context()) {
		CollectPageLog(pageLog)
	}
	if err := l.recent.WritePageLog(stream.Context(), pageLog); err != nil {
		return fmt.Errorf("write page log to recent store: %w", err)
	}
	return stream.SendAndClose(&emptypb.Empty{})
}

func (l *RecentServer) ListPageLogs(req *logV1.PageLogListRequest, stream logV1.Log_ListPageLogsServer) error {
	return listPageLogs(l.recent, req, stream)
}

func (l *RecentServer) ListCrawlLogs(req *logV1.CrawlLogListRequest, stream logV1.Log_ListCrawlLogsServer) error {
	return listCrawlLogs(l.recent, req, stream)
}

func receiveCrawlLogs(stream logV1.Log_WriteCrawlLogServer, write func(*logV1.CrawlLog) error) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&emptypb.Empty{})
		}
		if err != nil {
			return err
		}
		if err := write(req.GetCrawlLog()); err != nil {
			return fmt.Errorf("error writing crawl log: %w", err)
		}
	}
}

func receivePageLog(stream logV1.Log_WritePageLogServer) (*logV1.PageLog, error) {
	pageLog := &logV1.PageLog{}
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return pageLog, nil
		}
		if err != nil {
			return nil, err
		}
		switch req.Value.(type) {
		case *logV1.WritePageLogRequest_Outlink:
			pageLog.Outlink = append(pageLog.Outlink, req.GetOutlink())
		case *logV1.WritePageLogRequest_Resource:
			pageLog.Resource = append(pageLog.Resource, req.GetResource())
		case *logV1.WritePageLogRequest_CrawlLog:
			crawlLog := req.GetCrawlLog()
			pageLog.Uri = crawlLog.RequestedUri
			pageLog.ExecutionId = crawlLog.ExecutionId
			pageLog.Method = crawlLog.Method
			pageLog.CollectionFinalName = crawlLog.CollectionFinalName
			pageLog.Referrer = crawlLog.Referrer
			pageLog.JobExecutionId = crawlLog.JobExecutionId
			pageLog.WarcId = crawlLog.WarcId
		}
	}
}

func normalizeCrawlLog(crawlLog *logV1.CrawlLog) {
	// Preserve the existing millisecond precision and server-assigned timestamp.
	crawlLog.TimeStamp = timestamppb.New(time.Now().UTC().Truncate(time.Millisecond))
	crawlLog.FetchTimeStamp = timestamppb.New(crawlLog.FetchTimeStamp.AsTime().Truncate(time.Millisecond))
}

func isForwarded(ctx context.Context) bool {
	values := metadata.ValueFromIncomingContext(ctx, ForwardedMetadataKey)
	for _, value := range values {
		if value == "true" {
			return true
		}
	}
	return false
}

func writeCrawlLog(ctx context.Context, archive ArchiveWriter, recent RecentLogWriter, crawlLog *logV1.CrawlLog) error {
	if err := archive.WriteCrawlLog(crawlLog); err != nil {
		return err
	}
	if err := recent.WriteCrawlLog(ctx, crawlLog); err != nil {
		slog.Error("Failed to write crawl log to recent read store",
			"error", err,
			"warcId", crawlLog.GetWarcId(),
			"executionId", crawlLog.GetExecutionId(),
		)
	}
	return nil
}

func writePageLog(ctx context.Context, archive ArchiveWriter, recent RecentLogWriter, pageLog *logV1.PageLog) error {
	if err := archive.WritePageLog(pageLog); err != nil {
		return err
	}
	if err := recent.WritePageLog(ctx, pageLog); err != nil {
		slog.Error("Failed to write page log to recent read store",
			"error", err,
			"warcId", pageLog.GetWarcId(),
			"executionId", pageLog.GetExecutionId(),
		)
	}
	return nil
}

func listPageLogs(recent RecentLogStore, req *logV1.PageLogListRequest, stream logV1.Log_ListPageLogsServer) error {
	if len(req.GetWarcId()) > 0 {
		return recent.ListPageLogsByWarcID(stream.Context(), req.GetWarcId(), stream.Send)
	}
	if executionID := req.GetQueryTemplate().GetExecutionId(); executionID != "" {
		return recent.ListPageLogsByExecutionID(
			stream.Context(), executionID, int(req.GetOffset()), int(req.GetPageSize()), stream.Send,
		)
	}
	if hasQueryFilter(req.GetQueryTemplate(), req.GetQueryMask().GetPaths()) {
		return fmt.Errorf("request must provide warcId or executionId")
	}
	return recent.ListRecentPageLogs(stream.Context(), int(req.GetOffset()), int(req.GetPageSize()), stream.Send)
}

func listCrawlLogs(recent RecentLogStore, req *logV1.CrawlLogListRequest, stream logV1.Log_ListCrawlLogsServer) error {
	if len(req.GetWarcId()) > 0 {
		return recent.ListCrawlLogsByWarcID(stream.Context(), req.GetWarcId(), stream.Send)
	}
	if executionID := req.GetQueryTemplate().GetExecutionId(); executionID != "" {
		return recent.ListCrawlLogsByExecutionID(
			stream.Context(), executionID, int(req.GetOffset()), int(req.GetPageSize()), stream.Send,
		)
	}
	if hasQueryFilter(req.GetQueryTemplate(), req.GetQueryMask().GetPaths()) {
		return fmt.Errorf("request must provide warcId or executionId")
	}
	return recent.ListRecentCrawlLogs(stream.Context(), int(req.GetOffset()), int(req.GetPageSize()), stream.Send)
}

func hasQueryFilter(template proto.Message, queryPaths []string) bool {
	return proto.Size(template) > 0 || len(queryPaths) > 0
}
