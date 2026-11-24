package context_test

import (
	stdcontext "context"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	rperrors "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
)

var completions []*logV1.CrawlLog

func initializedRecordContext(t *testing.T, services *testutil.GrpcServiceMock, timeout time.Duration) *rpcontext.RecordContext {
	t.Helper()
	completions = nil
	uri, err := url.Parse("https://example.com/resource")
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(rpcontext.RecordProxyDataAware(stdcontext.Background()), http.MethodGet, uri.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	rc := rpcontext.NewRecordContext(timeout)
	rc.Complete = func(cl *logV1.CrawlLog, cached bool, _ *commonsV1.Error) { completions = append(completions, cl) }
	rc.Init(1, services.ContentWriterClient, req, uri)
	rc.Meta = &contentwriterV1.WriteRequest_Meta{Meta: &contentwriterV1.WriteRequestMeta{
		RecordMeta: map[int32]*contentwriterV1.WriteRequestMeta_RecordMeta{},
	}}
	return rc
}

func TestFinalizeStoredResponseTimesOutAndClosesSession(t *testing.T) {
	services := testutil.NewGrpcServiceMock(testutil.WithContentWriterWriteFunc(
		func(server contentwriterV1.ContentWriter_WriteServer) error {
			for {
				_, err := server.Recv()
				if err == io.EOF {
					<-server.Context().Done()
					return server.Context().Err()
				}
				if err != nil {
					return err
				}
			}
		},
	))
	defer services.Close()

	baseline := rpcontext.OpenSessions()
	rc := initializedRecordContext(t, services, 20*time.Millisecond)
	started := time.Now()
	err := rc.FinalizeStoredResponse(1, 0, "sha1:empty")
	if time.Since(started) > time.Second {
		t.Fatalf("finalization was not bounded: %s", time.Since(started))
	}
	if rperrors.Code(err) != rperrors.RuntimeException {
		t.Fatalf("FinalizeStoredResponse() error = %v, want RuntimeException", err)
	}
	if got := rpcontext.OpenSessions(); got != baseline {
		t.Fatalf("OpenSessions() = %d, want %d", got, baseline)
	}
	if got := len(completions); got != 1 {
		t.Fatalf("Ledger completion count = %d, want 1", got)
	}
}

func TestConcurrentFirstWritesCreateOneContentWriterStream(t *testing.T) {
	var streams atomic.Int32
	services := testutil.NewGrpcServiceMock(testutil.WithContentWriterWriteFunc(
		func(server contentwriterV1.ContentWriter_WriteServer) error {
			streams.Add(1)
			for {
				if _, err := server.Recv(); err != nil {
					return err
				}
			}
		},
	))
	defer services.Close()
	rc := initializedRecordContext(t, services, time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rc.SendProtocolHeader(0, []byte("GET / HTTP/1.1\r\n\r\n"))
		}()
	}
	wg.Wait()
	_ = rc.SendRequestError(stdcontext.Background(), rperrors.Error(rperrors.CanceledByBrowser, "CANCELED_BY_BROWSER", "test complete"))
	if got := streams.Load(); got != 1 {
		t.Fatalf("ContentWriter streams = %d, want 1", got)
	}
}

func TestTerminalCoordinatorCompletesAtMostOnce(t *testing.T) {
	services := testutil.NewGrpcServiceMock()
	defer services.Close()
	rc := initializedRecordContext(t, services, time.Second)

	errOne := rperrors.Error(rperrors.ConnectFailed, "CONNECT_FAILED", "first")
	errTwo := rperrors.Error(rperrors.CanceledByBrowser, "CANCELED_BY_BROWSER", "second")
	var wg sync.WaitGroup
	for _, terminalErr := range []error{errOne, errTwo} {
		wg.Add(1)
		go func(err error) {
			defer wg.Done()
			_ = rc.SendRequestError(stdcontext.Background(), err)
		}(terminalErr)
	}
	wg.Wait()
	if got := len(completions); got != 1 {
		t.Fatalf("Ledger completion count = %d, want 1", got)
	}
}

func TestFinalizationRejectsUnconfirmedWARC(t *testing.T) {
	services := testutil.NewGrpcServiceMock(testutil.WithContentWriterWriteFunc(
		func(stream contentwriterV1.ContentWriter_WriteServer) error {
			for {
				_, err := stream.Recv()
				if err == io.EOF {
					return stream.SendAndClose(&contentwriterV1.WriteReply{Meta: &contentwriterV1.WriteResponseMeta{RecordMeta: map[int32]*contentwriterV1.WriteResponseMeta_RecordMeta{1: {}}}})
				}
				if err != nil {
					return err
				}
			}
		},
	))
	defer services.Close()
	rc := initializedRecordContext(t, services, time.Second)
	if err := rc.FinalizeStoredResponse(1, 0, "sha1:empty"); rperrors.Code(err) != rperrors.RuntimeException {
		t.Fatal(err)
	}
	if len(completions) != 1 {
		t.Fatal("missing terminal outcome")
	}
	cl := completions[0]
	if cl.WarcId != "" || cl.GetError().GetCode() != -5 {
		t.Fatal(cl)
	}
}
