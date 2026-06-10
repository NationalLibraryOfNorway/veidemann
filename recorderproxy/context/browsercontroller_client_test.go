package context_test

import (
	stdcontext "context"
	"net/http"
	"net/url"
	"testing"

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	proxyerrors "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
)

func TestSendRequestErrorSkipsCompleteForCanceledProxyRequestWithoutRequestID(t *testing.T) {
	grpcServices := testutil.NewGrpcServiceMock()
	defer grpcServices.Close()

	uri, err := url.Parse("https://www.example.com/background")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	req, err := http.NewRequestWithContext(rpcontext.RecordProxyDataAware(stdcontext.Background()), http.MethodGet, uri.String(), nil)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}
	rc := rpcontext.NewRecordContext().Init(1, grpcServices.ClientConn, req, uri)

	err = rc.SendRequestError(nil, proxyerrors.Error(proxyerrors.CanceledByBrowser, "CANCELLED_BY_BROWSER", "Cancelled by browser controller"))
	if proxyerrors.Code(err) != proxyerrors.CanceledByBrowser {
		t.Fatalf("SendRequestError() code = %v, want %v", proxyerrors.Code(err), proxyerrors.CanceledByBrowser)
	}

	if got := len(grpcServices.Requests.BrowserControllerRequests); got != 0 {
		t.Fatalf("BrowserController request count = %d, want 0", got)
	}
}

func TestSendRequestErrorCompletesCanceledTrackedRequest(t *testing.T) {
	grpcServices := testutil.NewGrpcServiceMock()
	defer grpcServices.Close()

	uri, err := url.Parse("https://www.example.com/tracked")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	req, err := http.NewRequestWithContext(rpcontext.RecordProxyDataAware(stdcontext.Background()), http.MethodGet, uri.String(), nil)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}
	req.Header.Set(constants.HeaderRequestId, "interception-job-1.0")
	rc := rpcontext.NewRecordContext().Init(1, grpcServices.ClientConn, req, uri)

	err = rc.SendRequestError(nil, proxyerrors.Error(proxyerrors.CanceledByBrowser, "CANCELLED_BY_BROWSER", "Cancelled by browser controller"))
	if proxyerrors.Code(err) != proxyerrors.CanceledByBrowser {
		t.Fatalf("SendRequestError() code = %v, want %v", proxyerrors.Code(err), proxyerrors.CanceledByBrowser)
	}

	if got := len(grpcServices.Requests.BrowserControllerRequests); got != 1 {
		t.Fatalf("BrowserController request count = %d, want 1", got)
	}
	complete := grpcServices.Requests.BrowserControllerRequests[0].CompleteResource
	if complete == nil {
		t.Fatal("CompleteResource request was not sent")
	}
	if complete.RequestId != "interception-job-1.0" {
		t.Fatalf("CompleteResource requestId = %q, want %q", complete.RequestId, "interception-job-1.0")
	}
	if complete.CrawlLog.GetStatusCode() != int32(proxyerrors.CanceledByBrowser) {
		t.Fatalf("CompleteResource status = %d, want %d", complete.CrawlLog.GetStatusCode(), proxyerrors.CanceledByBrowser)
	}
}
