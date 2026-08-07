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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	TableCrawlLog = "crawl_log"
	TablePageLog  = "page_log"
	TableResource = "resource"
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

type RecentLogStore interface {
	WriteCrawlLog(ctx context.Context, crawlLog *logV1.CrawlLog) error
	WritePageLog(ctx context.Context, pageLog *logV1.PageLog) error
	ListCrawlLogsByWarcID(ctx context.Context, warcIDs []string, emit func(*logV1.CrawlLog) error) error
	ListCrawlLogsByExecutionID(ctx context.Context, executionID string, offset, pageSize int, emit func(*logV1.CrawlLog) error) error
	ListRecentCrawlLogs(ctx context.Context, offset, pageSize int, emit func(*logV1.CrawlLog) error) error
	ListPageLogsByWarcID(ctx context.Context, warcIDs []string, emit func(*logV1.PageLog) error) error
	ListPageLogsByExecutionID(ctx context.Context, executionID string, offset, pageSize int, emit func(*logV1.PageLog) error) error
	ListRecentPageLogs(ctx context.Context, offset, pageSize int, emit func(*logV1.PageLog) error) error
}

type LogServer struct {
	logV1.UnimplementedLogServer
	archive ArchiveWriter
	recent  RecentLogStore
}

// Assert that LogServer implements LogService.
var _ LogService = (*LogServer)(nil)

func New(archive ArchiveWriter, recent RecentLogStore) *LogServer {
	return &LogServer{
		archive: archive,
		recent:  recent,
	}
}

func (l *LogServer) WriteCrawlLog(stream logV1.Log_WriteCrawlLogServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&emptypb.Empty{})
		}
		if err != nil {
			return err
		}
		crawlLog := req.GetCrawlLog()
		CollectCrawlLog(crawlLog)
		if err := writeCrawlLog(stream.Context(), l.archive, l.recent, crawlLog); err != nil {
			return fmt.Errorf("error writing crawl log: %w", err)
		}
	}
}

func writeCrawlLog(ctx context.Context, archive ArchiveWriter, recent RecentLogStore, crawlLog *logV1.CrawlLog) error {
	// Generate timestamp with millisecond precision.
	// Preserve existing behavior and ensure deterministic timestamp precision.
	crawlLog.TimeStamp = timestamppb.New(time.Now().UTC().Truncate(time.Millisecond))
	// Convert FetchTimeStamp to millisecond precision
	crawlLog.FetchTimeStamp = timestamppb.New(crawlLog.FetchTimeStamp.AsTime().Truncate(time.Millisecond))
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

func (l *LogServer) WritePageLog(stream logV1.Log_WritePageLogServer) error {
	pageLog := &logV1.PageLog{}
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			CollectPageLog(pageLog)
			if err := writePageLog(stream.Context(), l.archive, l.recent, pageLog); err != nil {
				return fmt.Errorf("error writing page log: %w", err)
			}
			return stream.SendAndClose(&emptypb.Empty{})
		}
		if err != nil {
			return err
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

func writePageLog(ctx context.Context, archive ArchiveWriter, recent RecentLogStore, pageLog *logV1.PageLog) error {
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

func (l *LogServer) ListPageLogs(req *logV1.PageLogListRequest, stream logV1.Log_ListPageLogsServer) error {
	if len(req.GetWarcId()) > 0 {
		return l.recent.ListPageLogsByWarcID(stream.Context(), req.GetWarcId(), stream.Send)
	}
	if executionID := req.GetQueryTemplate().GetExecutionId(); executionID != "" {
		return l.recent.ListPageLogsByExecutionID(
			stream.Context(), executionID, int(req.GetOffset()), int(req.GetPageSize()), stream.Send,
		)
	}
	if hasQueryFilter(req.GetQueryTemplate(), req.GetQueryMask().GetPaths()) {
		return fmt.Errorf("request must provide warcId or executionId")
	}
	return l.recent.ListRecentPageLogs(
		stream.Context(), int(req.GetOffset()), int(req.GetPageSize()), stream.Send,
	)
}

func (l *LogServer) ListCrawlLogs(req *logV1.CrawlLogListRequest, stream logV1.Log_ListCrawlLogsServer) error {
	if len(req.GetWarcId()) > 0 {
		return l.recent.ListCrawlLogsByWarcID(stream.Context(), req.GetWarcId(), stream.Send)
	}
	if executionID := req.GetQueryTemplate().GetExecutionId(); executionID != "" {
		return l.recent.ListCrawlLogsByExecutionID(
			stream.Context(), executionID, int(req.GetOffset()), int(req.GetPageSize()), stream.Send,
		)
	}
	if hasQueryFilter(req.GetQueryTemplate(), req.GetQueryMask().GetPaths()) {
		return fmt.Errorf("request must provide warcId or executionId")
	}
	return l.recent.ListRecentCrawlLogs(
		stream.Context(), int(req.GetOffset()), int(req.GetPageSize()), stream.Send,
	)
}

func hasQueryFilter(template proto.Message, queryPaths []string) bool {
	return proto.Size(template) > 0 || len(queryPaths) > 0
}
