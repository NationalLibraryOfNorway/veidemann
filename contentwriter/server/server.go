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

package server

import (
	"io"
	"log/slog"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/database"
	"github.com/nlnwa/gowarc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ContentWriterService struct {
	configAdapter      database.ConfigAdapter
	warcWriterRegistry *warcWriterRegistry
	recordOptions      []gowarc.WarcRecordOption
}

func (s *ContentWriterService) Write(stream contentwriterV1.ContentWriter_WriteServer) (err error) {
	ctx := newWriteSessionContext(s.configAdapter, s.recordOptions)
	defer ctx.cancelSession()
	defer func() {
		if err != nil {
			slog.Error("Write request failed", "error", err, "code", status.Code(err).String())
		}
	}()

	for {
		request, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch v := request.Value.(type) {
		case *contentwriterV1.WriteRequest_Meta:
			slog.Debug("Got API request", "requestType", "meta", "recordCount", len(v.Meta.RecordMeta))
			ctx.setWriteRequestMeta(v.Meta)
		case *contentwriterV1.WriteRequest_ProtocolHeader:
			slog.Debug("Got API request", "requestType", "protocol_header", "recordNum", v.ProtocolHeader.RecordNum, "size", len(v.ProtocolHeader.GetData()))
			if err := ctx.writeProtocolHeader(v.ProtocolHeader); err != nil {
				return status.Errorf(codes.Unknown, "failed to write protocol header: %v", err)
			}
		case *contentwriterV1.WriteRequest_Payload:
			slog.Debug("Got API request", "requestType", "payload", "recordNum", v.Payload.RecordNum, "size", len(v.Payload.GetData()))
			if err := ctx.writePayload(v.Payload); err != nil {
				return status.Errorf(codes.Unknown, "failed to write payload: %v", err)
			}
		case *contentwriterV1.WriteRequest_Cancel:
			slog.Debug("Got API request", "requestType", "cancel", "type", v.Cancel)
			return stream.SendAndClose(new(contentwriterV1.WriteReply))
		default:
			return status.Errorf(codes.InvalidArgument, "invalid write request: %v", v)
		}
	}

	if err := ctx.validateSession(); err != nil {
		slog.Error("Validation failed", "error", err)
		return status.Errorf(codes.Unknown, "validation failed: %v", err)
	}

	records := make([]gowarc.WarcRecord, len(ctx.records))
	for i := range records {
		records[i] = ctx.records[int32(i)]
	}
	writer := s.warcWriterRegistry.GetWarcWriter(ctx.collectionConfig, ctx.meta.RecordMeta[0])
	writeReply, err := writer.Write(ctx.meta, records...)
	if err != nil {
		return status.Errorf(codes.Unknown, "failed writing record(s): %v", err)
	}

	return stream.SendAndClose(writeReply)
}
