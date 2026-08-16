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

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	rperrors "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
)

func initializedRecordContext(t *testing.T, services *testutil.GrpcServiceMock, timeout time.Duration) *rpcontext.RecordContext {
	t.Helper()
	uri, err := url.Parse("https://example.com/resource")
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(rpcontext.RecordProxyDataAware(stdcontext.Background()), http.MethodGet, uri.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	rc := rpcontext.NewRecordContext(timeout).Init(1, services.ClientConn, req, uri)
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
	if got := len(services.Requests.BrowserControllerRequests); got != 1 {
		t.Fatalf("BrowserController completion count = %d, want 1", got)
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
	rc.RequestId = "request-1"

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
	if got := len(services.Requests.BrowserControllerRequests); got != 1 {
		t.Fatalf("BrowserController completion count = %d, want 1", got)
	}
}
