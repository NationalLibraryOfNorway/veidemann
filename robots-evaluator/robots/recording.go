package robots

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	writerV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Recorder writes captured exchanges and crawl logs before a refresh returns.
type Recorder struct {
	timeout time.Duration
	writer  writerV1.ContentWriterClient
	logs    logV1.LogClient
}

func NewRecorder(writer writerV1.ContentWriterClient, logs logV1.LogClient, timeout time.Duration) *Recorder {
	return &Recorder{timeout: timeout, writer: writer, logs: logs}
}

func digest(data ...[]byte) string {
	h := sha1.New()
	for _, p := range data {
		_, _ = h.Write(p)
	}
	return fmt.Sprintf("sha1:%x", h.Sum(nil))
}

func (r *Recorder) record(ctx context.Context, metadata fetchMetadata) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	c := metadata.capture
	recorded := 0
	for i, x := range c.exchanges {
		if ctx.Err() != nil {
			c.skipped += len(c.exchanges) - i
			break
		}
		cl, err := r.writeExchange(ctx, metadata, x)
		if err != nil {
			c.skip("ContentWriter: " + err.Error())
			continue
		}
		if err := r.writeLog(ctx, cl); err != nil {
			c.skip("log-service: " + err.Error())
		} else {
			recorded++
		}
	}
	slog.Info("Robots recording summary",
		"executionId", metadata.execution,
		"jobExecutionId", metadata.job,
		"collectionId", metadata.collection,
		"recorded", recorded,
		"recordingContextError", ctx.Err(),
		"skipped", c.skipped,
		"causes", c.causes,
		"duration", time.Since(c.started),
	)
}

func (r *Recorder) writeExchange(ctx context.Context, m fetchMetadata, x exchange) (*logV1.CrawlLog, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := r.writer.Write(ctx)
	if err != nil {
		return nil, err
	}
	// Per-stream cancellation also terminates failed streams before the next pair.
	send := func(req *writerV1.WriteRequest) error {
		err := stream.Send(req)
		if err == io.EOF {
			_, err = stream.CloseAndRecv()
			if err == nil {
				err = io.ErrUnexpectedEOF
			}
		}
		return err
	}
	meta := &writerV1.WriteRequestMeta{
		TargetUri:      x.uri,
		ExecutionId:    m.execution,
		CollectionRef:  &configV1.ConfigRef{Kind: configV1.Kind_collection, Id: m.collection},
		FetchTimeStamp: timestamppb.New(x.started),
		IpAddress:      x.ip,
		RecordMeta: map[int32]*writerV1.WriteRequestMeta_RecordMeta{
			0: {
				RecordNum:         0,
				Type:              writerV1.RecordType_REQUEST,
				RecordContentType: "application/http; msgtype=request",
				Size:              int64(len(x.request)),
				BlockDigest:       digest(x.request),
				PayloadDigest:     digest(nil),
			},
			1: {
				RecordNum:         1,
				Type:              writerV1.RecordType_RESPONSE,
				RecordContentType: "application/http; msgtype=response",
				Size:              int64(len(x.response) + len(x.body)),
				BlockDigest:       digest(x.response, x.body),
				PayloadDigest:     digest(x.body),
			},
		},
	}
	// ContentWriter accepts metadata before protocol headers/payloads.
	if err := send(&writerV1.WriteRequest{Value: &writerV1.WriteRequest_Meta{Meta: meta}}); err != nil {
		return nil, err
	}
	if err := send(&writerV1.WriteRequest{
		Value: &writerV1.WriteRequest_ProtocolHeader{
			ProtocolHeader: &writerV1.Data{RecordNum: 0, Data: x.request},
		},
	}); err != nil {
		return nil, err
	}
	if err := send(&writerV1.WriteRequest{
		Value: &writerV1.WriteRequest_ProtocolHeader{
			ProtocolHeader: &writerV1.Data{RecordNum: 1, Data: x.response},
		},
	}); err != nil {
		return nil, err
	}
	for remaining := x.body; len(remaining) > 0; {
		n := min(len(remaining), 64<<10)
		if err := send(&writerV1.WriteRequest{
			Value: &writerV1.WriteRequest_Payload{
				Payload: &writerV1.Data{RecordNum: 1, Data: remaining[:n]},
			},
		}); err != nil {
			return nil, err
		}
		remaining = remaining[n:]
	}
	reply, err := stream.CloseAndRecv()
	if err != nil {
		return nil, err
	}
	if len(reply.GetMeta().GetRecordMeta()) != 2 {
		return nil, fmt.Errorf("ContentWriter must confirm exactly two records")
	}
	for i := int32(0); i <= 1; i++ {
		got := reply.GetMeta().GetRecordMeta()[i]
		want := meta.RecordMeta[i]
		if err := validateRecord(got, want); err != nil {
			return nil, fmt.Errorf("invalid confirmed record %d: %w", i, err)
		}
	}
	got := reply.Meta.RecordMeta[1]
	return &logV1.CrawlLog{
		ExecutionId:         m.execution,
		JobExecutionId:      m.job,
		RequestedUri:        x.uri,
		Method:              http.MethodGet,
		DiscoveryPath:       "P",
		StatusCode:          int32(x.status),
		ContentType:         x.mime,
		FetchTimeStamp:      timestamppb.New(x.started),
		TimeStamp:           timestamppb.Now(),
		FetchTimeMs:         x.duration.Milliseconds(),
		IpAddress:           x.ip,
		Size:                meta.RecordMeta[1].Size,
		WarcId:              got.WarcId,
		StorageRef:          got.StorageRef,
		CollectionFinalName: got.CollectionFinalName,
		RecordType:          strings.ToLower(got.Type.String()),
		BlockDigest:         got.BlockDigest,
		PayloadDigest:       got.PayloadDigest,
		WarcRefersTo:        got.RevisitReferenceId,
	}, nil
}

func validateRecord(got *writerV1.WriteResponseMeta_RecordMeta, want *writerV1.WriteRequestMeta_RecordMeta) error {
	switch {
	case got == nil:
		return fmt.Errorf("missing metadata")
	case got.RecordNum != want.RecordNum:
		return fmt.Errorf("unexpected record number %d", got.RecordNum)
	case !validWarcID(got.WarcId):
		return fmt.Errorf("invalid WARC ID %q", got.WarcId)
	case !validStorageRef(got.StorageRef):
		return fmt.Errorf("invalid storage reference %q", got.StorageRef)
	case got.CollectionFinalName == "":
		return fmt.Errorf("missing collection name")
	case got.BlockDigest == "":
		return fmt.Errorf("missing block digest")
	case got.PayloadDigest != want.PayloadDigest:
		return fmt.Errorf("payload digest mismatch")
	}
	if got.Type == writerV1.RecordType_REVISIT {
		if want.Type != writerV1.RecordType_RESPONSE || !validWarcID(got.RevisitReferenceId) {
			return fmt.Errorf("invalid revisit reference")
		}
		return nil
	}
	if got.Type != want.Type {
		return fmt.Errorf("unexpected record type %s", got.Type)
	}
	if got.BlockDigest != want.BlockDigest {
		return fmt.Errorf("block digest mismatch")
	}
	return nil
}

func (r *Recorder) writeLog(ctx context.Context, cl *logV1.CrawlLog) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := r.logs.WriteCrawlLog(ctx)
	if err != nil {
		return err
	}
	if err := stream.Send(&logV1.WriteCrawlLogRequest{CrawlLog: cl}); err != nil {
		if err == io.EOF {
			_, err = stream.CloseAndRecv()
			if err == nil {
				err = io.ErrUnexpectedEOF
			}
		}
		return err
	}
	_, err = stream.CloseAndRecv()
	return err
}

func validWarcID(id string) bool {
	parsed, err := uuid.Parse(strings.Trim(id, "<>"))
	return err == nil && parsed != uuid.Nil
}
func validStorageRef(ref string) bool {
	if !strings.HasPrefix(ref, "warcfile:") {
		return false
	}
	pathOffset := strings.TrimPrefix(ref, "warcfile:")
	separator := strings.LastIndexByte(pathOffset, ':')
	if separator <= 0 {
		return false
	}
	n, err := strconv.ParseInt(pathOffset[separator+1:], 10, 64)
	return err == nil && n >= 0
}
