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
	"io"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log"
)

type LogWriter struct {
	logV1.LogClient
}

func (l *LogWriter) WriteCrawlLog(ctx context.Context, crawlLog *logV1.CrawlLog) error {
	return l.WriteCrawlLogs(ctx, []*logV1.CrawlLog{crawlLog})
}

func (l *LogWriter) WriteCrawlLogs(ctx context.Context, crawlLogs []*logV1.CrawlLog) error {
	stream, err := l.LogClient.WriteCrawlLog(ctx)
	if err != nil {
		return err
	}
	for _, crawlLog := range crawlLogs {
		req := &logV1.WriteCrawlLogRequest{CrawlLog: crawlLog}
		if err := stream.Send(req); err != nil {
			if err == io.EOF {
				_, err = stream.CloseAndRecv()
			}
			return err
		}
	}
	if _, err = stream.CloseAndRecv(); err != io.EOF {
		return err
	}
	return nil
}

func (l *LogWriter) WritePageLog(ctx context.Context, pageLog *logV1.PageLog) error {
	stream, err := l.LogClient.WritePageLog(ctx)
	if err != nil {
		return err
	}
	// send metadata using crawllog as transport
	metadata := &logV1.WritePageLogRequest{
		Value: &logV1.WritePageLogRequest_CrawlLog{
			CrawlLog: &logV1.CrawlLog{
				WarcId:              pageLog.WarcId,
				RequestedUri:        pageLog.Uri,
				Referrer:            pageLog.Referrer,
				ExecutionId:         pageLog.ExecutionId,
				JobExecutionId:      pageLog.JobExecutionId,
				CollectionFinalName: pageLog.CollectionFinalName,
				Method:              pageLog.Method,
			},
		},
	}
	if err := stream.Send(metadata); err != nil {
		if err == io.EOF {
			_, err = stream.CloseAndRecv()
		}
		return err
	}
	// send resources
	for _, resource := range pageLog.GetResource() {
		req := &logV1.WritePageLogRequest{
			Value: &logV1.WritePageLogRequest_Resource{Resource: resource},
		}
		if err := stream.Send(req); err != nil {
			if err == io.EOF {
				_, err = stream.CloseAndRecv()
			}
			return err
		}
	}
	// send outlinks
	for _, outlink := range pageLog.GetOutlink() {
		req := &logV1.WritePageLogRequest{
			Value: &logV1.WritePageLogRequest_Outlink{Outlink: outlink},
		}
		if err := stream.Send(req); err != nil {
			if err == io.EOF {
				_, err = stream.CloseAndRecv()
			}
			return err
		}
	}

	if _, err = stream.CloseAndRecv(); err != io.EOF {
		return err
	}
	return nil
}
