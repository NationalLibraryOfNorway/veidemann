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
	"fmt"
	"io"
	"time"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/log-service/internal/parquet"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	TableCrawlLog = "crawl_log"
	TablePageLog  = "page_log"
	TableResource = "resource"
)

type LogWriter interface {
	WriteCrawlLog(stream logV1.Log_WriteCrawlLogServer) error
	WritePageLog(stream logV1.Log_WritePageLogServer) error
	ListPageLogs(req *logV1.PageLogListRequest, stream logV1.Log_ListPageLogsServer) error
	ListCrawlLogs(req *logV1.CrawlLogListRequest, stream logV1.Log_ListCrawlLogsServer) error
	Close() error
}

type LogServer struct {
	logV1.UnimplementedLogServer
	storage *parquet.Storage
}

// Assert that LogServer implements LogWriter.
var _ LogWriter = (*LogServer)(nil)

func New(storage *parquet.Storage) *LogServer {
	return &LogServer{
		storage: storage,
	}
}

// Close flushes and closes any open parquet writers.
func (l *LogServer) Close() error {
	return l.storage.Close()
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
		if err := writeCrawlLog(l.storage, crawlLog); err != nil {
			return fmt.Errorf("error writing crawl log: %w", err)
		}
	}
}

func writeCrawlLog(storage *parquet.Storage, crawlLog *logV1.CrawlLog) error {
	// Generate timestamp with millisecond precision.
	// Preserve existing behavior and ensure deterministic timestamp precision.
	crawlLog.TimeStamp = timestamppb.New(time.Now().UTC().Truncate(time.Millisecond))
	// Convert FetchTimeStamp to millisecond precision
	crawlLog.FetchTimeStamp = timestamppb.New(crawlLog.FetchTimeStamp.AsTime().Truncate(time.Millisecond))
	return storage.WriteCrawlLog(crawlLog)
}

func (l *LogServer) WritePageLog(stream logV1.Log_WritePageLogServer) error {
	pageLog := &logV1.PageLog{}
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			CollectPageLog(pageLog)
			if err := writePageLog(l.storage, pageLog); err != nil {
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

func writePageLog(storage *parquet.Storage, pageLog *logV1.PageLog) error {
	return storage.WritePageLog(pageLog)
}

func (l *LogServer) ListPageLogs(req *logV1.PageLogListRequest, stream logV1.Log_ListPageLogsServer) error {
	var rows []*logV1.PageLog
	var err error
	if len(req.GetWarcId()) > 0 {
		rows, err = l.storage.ListPageLogsByWarcID(req.GetWarcId())
	} else if len(req.GetQueryTemplate().GetExecutionId()) > 0 {
		rows, err = l.storage.ListPageLogsByExecutionID(req.GetQueryTemplate().GetExecutionId(), int(req.GetOffset()), int(req.GetPageSize()))
	} else {
		return fmt.Errorf("request must provide warcId or executionId")
	}
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := stream.Send(row); err != nil {
			return err
		}
	}
	return nil
}

func (l *LogServer) ListCrawlLogs(req *logV1.CrawlLogListRequest, stream logV1.Log_ListCrawlLogsServer) error {
	var rows []*logV1.CrawlLog
	var err error
	if len(req.GetWarcId()) > 0 {
		rows, err = l.storage.ListCrawlLogsByWarcID(req.GetWarcId())
	} else if len(req.GetQueryTemplate().GetExecutionId()) > 0 {
		rows, err = l.storage.ListCrawlLogsByExecutionID(req.GetQueryTemplate().GetExecutionId(), int(req.GetOffset()), int(req.GetPageSize()))
	} else {
		return fmt.Errorf("request must provide warcId or executionId")
	}
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := stream.Send(row); err != nil {
			return err
		}
	}
	return nil
}
