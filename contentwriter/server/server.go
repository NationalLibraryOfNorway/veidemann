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

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/database"
	"github.com/nlnwa/gowarc"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ContentWriterService struct {
	configCache        database.ConfigAdapter
	warcWriterRegistry *warcWriterRegistry
	recordOptions      []gowarc.WarcRecordOption
}

func (s *ContentWriterService) Close() {
	s.warcWriterRegistry.Shutdown()
}

func (s *ContentWriterService) Write(stream contentwriterV1.ContentWriter_WriteServer) (err error) {
	ScopechecksTotal.Inc()
	//telemetry.ScopecheckResponseTotal.With(prometheus.Labels{"code": strconv.Itoa(int(result.ExcludeReason))}).Inc()
	ctx := newWriteSessionContext(s.configCache, s.recordOptions)
	defer ctx.cancelSession()
	defer func() {
		if err != nil {
			log.Error().Err(err).Str("code", status.Code(err).String()).Msg("")
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
			log.Trace().Msgf("Got API request %T for %d records", v, len(v.Meta.RecordMeta))
			ctx.setWriteRequestMeta(v.Meta)
		case *contentwriterV1.WriteRequest_ProtocolHeader:
			log.Trace().Msgf("Got API request %T for record #%d. Size: %d", v, v.ProtocolHeader.RecordNum, len(v.ProtocolHeader.GetData()))
			if err := ctx.writeProtocolHeader(v.ProtocolHeader); err != nil {
				return status.Errorf(codes.Unknown, "failed to write protocol header: %v", err)
			}
		case *contentwriterV1.WriteRequest_Payload:
			log.Trace().Msgf("Got API request %T for record #%d. Size: %d", v, v.Payload.RecordNum, len(v.Payload.GetData()))
			if err := ctx.writePayload(v.Payload); err != nil {
				return status.Errorf(codes.Unknown, "failed to write payload: %v", err)
			}
		case *contentwriterV1.WriteRequest_Cancel:
			log.Trace().Msgf("Got API request %T", v)
			log.Debug().Str("reason", v.Cancel).Msg("Write request cancelled")
			return stream.SendAndClose(new(contentwriterV1.WriteReply))
		default:
			return status.Errorf(codes.InvalidArgument, "invalid write request: %v", v)
		}
	}

	if err := ctx.validateSession(); err != nil {
		log.Warn().Err(err).Msg("Validate session")
		return status.Errorf(codes.Unknown, "validation failed: %v", err)
	}

	records := make([]gowarc.WarcRecord, len(ctx.records))
	for i := 0; i < len(records); i++ {
		records[i] = ctx.records[int32(i)]
	}
	writer := s.warcWriterRegistry.GetWarcWriter(ctx.collectionConfig, ctx.meta.RecordMeta[0])
	writeReply, err := writer.Write(ctx.meta, records...)
	if err != nil {
		return status.Errorf(codes.Unknown, "failed writing record(s): %v", err)
	}

	return stream.SendAndClose(writeReply)
}
