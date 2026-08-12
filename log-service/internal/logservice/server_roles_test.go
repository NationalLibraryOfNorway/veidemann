package logservice

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type forwarderStub struct {
	crawlLogs []*logV1.CrawlLog
	pageLogs  []*logV1.PageLog
}

func (f *forwarderStub) EnqueueCrawlLog(crawlLog *logV1.CrawlLog) {
	f.crawlLogs = append(f.crawlLogs, crawlLog)
}

func (f *forwarderStub) EnqueuePageLog(pageLog *logV1.PageLog) {
	f.pageLogs = append(f.pageLogs, pageLog)
}

type crawlWriteStream struct {
	grpc.ServerStream
	ctx      context.Context
	requests []*logV1.WriteCrawlLogRequest
	index    int
	closed   bool
}

func (s *crawlWriteStream) Context() context.Context { return s.ctx }

func (s *crawlWriteStream) Recv() (*logV1.WriteCrawlLogRequest, error) {
	if s.index == len(s.requests) {
		return nil, io.EOF
	}
	req := s.requests[s.index]
	s.index++
	return req, nil
}

func (s *crawlWriteStream) SendAndClose(*emptypb.Empty) error {
	s.closed = true
	return nil
}

type pageWriteStream struct {
	grpc.ServerStream
	ctx      context.Context
	requests []*logV1.WritePageLogRequest
	index    int
	closed   bool
}

func (s *pageWriteStream) Context() context.Context { return s.ctx }

func (s *pageWriteStream) Recv() (*logV1.WritePageLogRequest, error) {
	if s.index == len(s.requests) {
		return nil, io.EOF
	}
	req := s.requests[s.index]
	s.index++
	return req, nil
}

func (s *pageWriteStream) SendAndClose(*emptypb.Empty) error {
	s.closed = true
	return nil
}

func TestWriterServerArchivesThenEnqueues(t *testing.T) {
	archive := &archiveStub{}
	forwarder := &forwarderStub{}
	server := NewWriter(archive, forwarder)
	fetchTime := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 123456789, time.UTC))
	stream := &crawlWriteStream{
		ctx: context.Background(),
		requests: []*logV1.WriteCrawlLogRequest{{CrawlLog: &logV1.CrawlLog{
			WarcId:         "crawl-1",
			FetchTimeStamp: fetchTime,
		}}},
	}

	if err := server.WriteCrawlLog(stream); err != nil {
		t.Fatal(err)
	}
	if archive.crawlWrites != 1 || len(forwarder.crawlLogs) != 1 || !stream.closed {
		t.Fatalf("expected archive, enqueue, and acknowledgement; archive=%d queued=%d closed=%v",
			archive.crawlWrites, len(forwarder.crawlLogs), stream.closed)
	}
	forwarded := forwarder.crawlLogs[0]
	if forwarded.GetTimeStamp() == nil {
		t.Fatal("expected writer-assigned timestamp")
	}
	if got := forwarded.GetFetchTimeStamp().AsTime().Nanosecond(); got != 123000000 {
		t.Fatalf("expected millisecond fetch timestamp precision, got %d", got)
	}
}

func TestWriterServerParquetFailureSkipsForward(t *testing.T) {
	wantErr := errors.New("parquet unavailable")
	archive := &archiveStub{crawlErr: wantErr, pageErr: wantErr}
	forwarder := &forwarderStub{}
	server := NewWriter(archive, forwarder)

	crawlStream := &crawlWriteStream{
		ctx:      context.Background(),
		requests: []*logV1.WriteCrawlLogRequest{{CrawlLog: &logV1.CrawlLog{WarcId: "crawl"}}},
	}
	if err := server.WriteCrawlLog(crawlStream); !errors.Is(err, wantErr) {
		t.Fatalf("expected parquet error, got %v", err)
	}

	pageStream := pageStreamFor(&logV1.PageLog{WarcId: "page"}, context.Background())
	if err := server.WritePageLog(pageStream); !errors.Is(err, wantErr) {
		t.Fatalf("expected parquet error, got %v", err)
	}
	if len(forwarder.crawlLogs) != 0 || len(forwarder.pageLogs) != 0 {
		t.Fatalf("parquet failures must not be forwarded: crawl=%d page=%d",
			len(forwarder.crawlLogs), len(forwarder.pageLogs))
	}
}

func TestWriterServerReadMethodsAreUnimplemented(t *testing.T) {
	server := NewWriter(&archiveStub{}, nil)
	if code := status.Code(server.ListCrawlLogs(&logV1.CrawlLogListRequest{}, &crawlListStream{ctx: context.Background()})); code != codes.Unimplemented {
		t.Fatalf("expected Unimplemented crawl read, got %s", code)
	}
	if code := status.Code(server.ListPageLogs(&logV1.PageLogListRequest{}, &pageListStream{ctx: context.Background()})); code != codes.Unimplemented {
		t.Fatalf("expected Unimplemented page read, got %s", code)
	}
}

func TestRecentServerPropagatesSQLiteWriteFailures(t *testing.T) {
	wantErr := errors.New("sqlite unavailable")
	server := NewRecent(&recentStoreStub{crawlErr: wantErr, pageErr: wantErr})
	crawlStream := &crawlWriteStream{
		ctx:      context.Background(),
		requests: []*logV1.WriteCrawlLogRequest{{CrawlLog: &logV1.CrawlLog{WarcId: "crawl"}}},
	}
	if err := server.WriteCrawlLog(crawlStream); !errors.Is(err, wantErr) {
		t.Fatalf("expected SQLite crawl error, got %v", err)
	}
	pageStream := pageStreamFor(&logV1.PageLog{WarcId: "page"}, context.Background())
	if err := server.WritePageLog(pageStream); !errors.Is(err, wantErr) {
		t.Fatalf("expected SQLite page error, got %v", err)
	}
}

func TestRecentServerPreservesForwardedTimestamp(t *testing.T) {
	recent := &capturingRecentStore{}
	server := NewRecent(recent)
	wantTimestamp := timestamppb.New(time.Date(2026, 2, 3, 4, 5, 6, 7000000, time.UTC))
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(ForwardedMetadataKey, "true"))
	stream := &crawlWriteStream{
		ctx: ctx,
		requests: []*logV1.WriteCrawlLogRequest{{CrawlLog: &logV1.CrawlLog{
			WarcId:    "forwarded",
			TimeStamp: wantTimestamp,
		}}},
	}
	if err := server.WriteCrawlLog(stream); err != nil {
		t.Fatal(err)
	}
	if got := recent.crawlLog.GetTimeStamp(); !got.AsTime().Equal(wantTimestamp.AsTime()) {
		t.Fatalf("expected forwarded timestamp %s, got %s", wantTimestamp.AsTime(), got.AsTime())
	}
}

func pageStreamFor(pageLog *logV1.PageLog, ctx context.Context) *pageWriteStream {
	return &pageWriteStream{
		ctx: ctx,
		requests: []*logV1.WritePageLogRequest{{
			Value: &logV1.WritePageLogRequest_CrawlLog{CrawlLog: &logV1.CrawlLog{
				WarcId:              pageLog.GetWarcId(),
				RequestedUri:        pageLog.GetUri(),
				ExecutionId:         pageLog.GetExecutionId(),
				CollectionFinalName: pageLog.GetCollectionFinalName(),
			}},
		}},
	}
}

type capturingRecentStore struct {
	recentStoreStub
	crawlLog *logV1.CrawlLog
}

func (s *capturingRecentStore) WriteCrawlLog(_ context.Context, crawlLog *logV1.CrawlLog) error {
	s.crawlLog = crawlLog
	return nil
}
