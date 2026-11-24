package recorderproxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	rperrors "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	proxy "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/internal/proxy"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func admitFailureResource(t *testing.T, a *harvestAttempt, writer contentwriterV1.ContentWriterClient, main bool) (*rpcontext.RecordContext, *http.Request) {
	t.Helper()
	uri := a.spec.RequestedUri
	if !main {
		uri += "image.png"
	}
	req, err := http.NewRequestWithContext(rpcontext.RecordProxyDataAware(t.Context()), http.MethodGet, uri, strings.NewReader("request"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Sec-Fetch-Dest", "image")
	if main {
		req.Header.Set(constants.HeaderMainDocument, a.spec.AttemptId)
		req.Header.Set("Sec-Fetch-Mode", "navigate")
	}
	rc := rpcontext.NewRecordContext(time.Second)
	if !a.admit(rc, func() {
		a.lifecycle.addRecord(rc)
		rc.Init(1, writer, req, req.URL)
	}) {
		t.Fatal("admission failed")
	}
	return rc, req
}

func beginRecordedPage(t *testing.T) (*Control, *harvestAttempt, *recorderV1.FinishHarvestHeader) {
	t.Helper()
	c, p := testControl()
	spec := beginSpec("attempt-failure-policy-0001")
	if _, err := c.BeginHarvest(t.Context(), spec); err != nil {
		t.Fatal(err)
	}
	main := &rpcontext.RecordContext{MainDocument: true}
	p.attempt.admit(main, func() {})
	main.Complete(&logV1.CrawlLog{RequestedUri: spec.RequestedUri, WarcId: "recorded-main", StatusCode: 200, Size: 42}, false, nil)
	return c, p.attempt, &recorderV1.FinishHarvestHeader{
		ProxyId: 1, AttemptId: spec.AttemptId,
		PageUri: &spec.RequestedUri,
	}
}

func TestFetchFailurePolicy(t *testing.T) {
	for _, main := range []bool{false, true} {
		for _, cause := range []error{
			errors.New("connection refused"), context.DeadlineExceeded,
			&net.DNSError{Err: "no such host", Name: "missing.example"},
			io.ErrUnexpectedEOF, errors.New("tls: bad record MAC"),
		} {
			t.Run(fmt.Sprintf("main=%v/%s", main, cause), func(t *testing.T) {
				c, a, finish := beginRecordedPage(t)
				rc, _ := admitFailureResource(t, a, nil, main)
				failure := classifyFailure(cause, FailureScopeResource)
				_ = rc.SendFetchError(failure.asError())
				reply, err := c.FinishPage(t.Context(), finish, []string{"https://example.com/link"})
				if err != nil || (reply.Failure != nil) != main || !reply.Reusable || reply.UriCount != 1 || reply.BytesDownloaded != 42 {
					t.Fatalf("reply=%v err=%v", reply, err)
				}
				logs := c.logs.(*testutil.LogWriterMock)
				if got := logs.CrawlLogs(); len(got) != 2 || got[1].GetError().GetCode() != int32(failure.Code) || got[1].WarcId != "" {
					t.Fatalf("resource outcome lost: %v", got)
				}
				pages := logs.PageLogs()
				if main {
					if len(pages) != 0 || reply.Main.GetError() == nil || reply.Main.GetWarcId() != "" {
						t.Fatal("failed main replaced by an older recording", reply)
					}
				} else if len(pages) != 1 || len(pages[0].Resource) != 2 || pages[0].Resource[1].GetError().GetCode() != int32(failure.Code) || len(pages[0].Outlink) != 1 || reply.Main.WarcId != "recorded-main" {
					t.Fatalf("page outcome lost: %v", pages)
				}
				replay, err := c.FinishPage(t.Context(), finish, []string{"https://example.com/link"})
				if err != nil || !proto.Equal(reply, replay) || len(logs.CrawlLogs()) != 2 || len(logs.PageLogs()) != len(pages) {
					t.Fatal("Finish replay changed the outcome")
				}
			})
		}
	}
}

func TestUnknownResourceFailureRemainsFatal(t *testing.T) {
	c, a, finish := beginRecordedPage(t)
	rc, req := admitFailureResource(t, a, nil, false)
	_ = rc.SendRequestError(req.Context(), errors.New("unclassified internal failure"))
	reply, err := c.FinishPage(t.Context(), finish, []string{"https://example.com/link"})
	if err != nil || reply.GetFailure().GetCode() != -5 {
		t.Fatalf("reply=%v err=%v", reply, err)
	}
}

// Inject failures at the client boundary so Send failures are deterministic,
// rather than depending on when gRPC receives a server-side stream failure.
type failingWriter struct {
	contentwriterV1.ContentWriterClient
	contentwriterV1.ContentWriter_WriteClient
	failAt        string
	terminalCalls int
}

func (w *failingWriter) Write(context.Context, ...grpc.CallOption) (contentwriterV1.ContentWriter_WriteClient, error) {
	if w.failAt == "open" {
		return nil, errors.New("writer unavailable")
	}
	return w, nil
}

func (w *failingWriter) Send(req *contentwriterV1.WriteRequest) error {
	stage := ""
	switch value := req.Value.(type) {
	case *contentwriterV1.WriteRequest_ProtocolHeader:
		stage = "header"
	case *contentwriterV1.WriteRequest_Payload:
		stage = fmt.Sprintf("payload-%d", value.Payload.RecordNum)
	case *contentwriterV1.WriteRequest_Meta:
		stage = "meta"
		w.terminalCalls++
	case *contentwriterV1.WriteRequest_Cancel:
		stage = "cancel"
		w.terminalCalls++
	}
	if stage == w.failAt {
		return errors.New("injected writer " + stage + " failure")
	}
	return nil
}

func (w *failingWriter) CloseAndRecv() (*contentwriterV1.WriteReply, error) {
	if w.failAt == "close" {
		return nil, errors.New("writer close failed")
	}
	// Deliberately omit a confirmed WARC for the metadata validation case.
	return &contentwriterV1.WriteReply{Meta: &contentwriterV1.WriteResponseMeta{}}, nil
}

func TestResourceRecordingFailuresRemainFatal(t *testing.T) {
	for _, stage := range []string{"open", "header", "payload-0", "payload-1", "meta", "close", "missing-warc", "cancel"} {
		t.Run(stage, func(t *testing.T) {
			c, a, finish := beginRecordedPage(t)
			writer := &failingWriter{failAt: stage}
			rc, req := admitFailureResource(t, a, writer, false)
			request, err := newRequestRecorder(req.Context(), rc).Wrap(req)
			if err != nil {
				_ = rc.SendRequestError(req.Context(), err)
			} else if stage == "cancel" {
				_ = rc.SendFetchError(rperrors.Error(rperrors.ConnectFailed, "CONNECT_FAILED", "connection refused"))
			} else {
				_, err = io.ReadAll(request.Body)
				_ = request.Body.Close()
				if err != nil {
					// The transport sees this after request-body recording failed.
					_ = rc.SendFetchError(classifyFailure(err, FailureScopeResource).asError())
				} else {
					resp := &http.Response{StatusCode: 200, Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("response"))}
					response, wrapErr := newResponseRecorder(req.Context(), rc).Wrap(resp)
					if wrapErr != nil {
						_ = rc.SendResponseError(req.Context(), wrapErr)
					} else {
						_, _ = io.ReadAll(response.Body)
						_ = response.Body.Close()
					}
				}
			}
			reply, err := c.FinishPage(t.Context(), finish, []string{"https://example.com/link"})
			if err != nil || reply.GetFailure().GetCode() != -5 || !reply.Reusable {
				t.Fatalf("reply=%v err=%v", reply, err)
			}
			logs := c.logs.(*testutil.LogWriterMock).CrawlLogs()
			wantCode := int32(-5)
			if stage == "cancel" {
				wantCode = -2
			}
			if len(logs) != 2 || logs[1].GetError().GetCode() != wantCode || logs[1].WarcId != "" {
				t.Fatalf("resource outcome=%v", logs)
			}
			wantTerminal := 1
			if stage == "open" {
				wantTerminal = 0
			}
			if writer.terminalCalls != wantTerminal {
				t.Fatalf("terminal operations=%d want=%d", writer.terminalCalls, wantTerminal)
			}
		})
	}
}

func TestFetchFailureRequiresKnownBoundary(t *testing.T) {
	for _, tc := range []struct {
		name  string
		err   error
		fetch bool
	}{
		{"unknown source", errors.New("connection refused"), false},
		{"origin TLS read", proxy.NewPhaseError(proxy.PhaseHTTPRoundTrip, errors.New("tls: bad record MAC")), true},
		{"origin dial", proxy.NewPhaseError(proxy.PhaseConnectDial, errors.New("connection refused")), true},
		{"writer inside transport", proxy.NewPhaseError(proxy.PhaseHTTPRoundTrip, rperrors.ErrorInternal(rperrors.RuntimeException, "writer failed", "")), false},
		{"proxy service dial", proxy.NewPhaseError(proxy.PhaseHTTPRoundTrip, (&RecorderProxy{nextProxy: "cache:3128"}).dialFailure(errors.New("connection refused"))), false},
		{"known Squid origin", proxy.NewPhaseError(proxy.PhaseUpstreamProxyConnect, handleSquidErrorString("ERR_CONNECT_FAIL 111")), true},
		{"unknown Squid failure", proxy.NewPhaseError(proxy.PhaseUpstreamProxyConnect, handleSquidErrorString("ERR_UNKNOWN 0")), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isFetchFailure(tc.err); got != tc.fetch {
				t.Fatalf("isFetchFailure=%v want=%v", got, tc.fetch)
			}
		})
	}
}
