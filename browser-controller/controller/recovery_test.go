package controller

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/recorderproxy"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

// The recorder survives both client connections. Exercise the actual controller
// startup barrier over gRPC; this does not simulate a container restart.
func TestStartupRecoveryAfterClientReconnect(t *testing.T) {
	for _, timedOut := range []bool{false, true} {
		t.Run(map[bool]string{false: "disconnect-before-finish", true: "disconnect-after-timeout"}[timedOut], func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			logs := &blockedCrawlLogs{entered: make(chan struct{}), release: make(chan struct{})}
			var release sync.Once
			unblock := func() { release.Do(func() { close(logs.release) }) }
			services := testutil.NewGrpcServiceMock()
			defer services.Close()
			proxy := recorderproxy.NewRecorderProxy(1, services.ContentWriterClient, services.DnsResolverClient, "", recorderproxy.WithFinalizationTimeout(50*time.Millisecond))
			defer func() { _ = proxy.Shutdown(ctx) }()
			defer unblock()
			control := recorderproxy.NewControl(logs, proxy, recorderproxy.NewRecorderProxy(2, nil, nil, ""))
			listener := bufconn.Listen(1024 * 1024)
			server := grpc.NewServer()
			recorderV1.RegisterRecorderControlServer(server, control)
			go func() {
				if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
					t.Errorf("serve gRPC: %v", err)
				}
			}()
			defer server.Stop()
			connect := func() (*grpc.ClientConn, recorderV1.RecorderControlClient) {
				t.Helper()
				conn, err := grpc.NewClient("passthrough:///recorder", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }))
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = conn.Close() })
				return conn, recorderV1.NewRecorderControlClient(conn)
			}
			oldConn, old := connect()
			tokens, err := RecoverSlots(ctx, old, 2, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			begin := func(client recorderV1.RecorderControlClient, slot int32, id string) {
				t.Helper()
				_, err := client.BeginHarvest(ctx, &recorderV1.BeginHarvestRequest{AdmissionToken: tokens[slot-1], ProxyId: slot, AttemptId: id, CrawlExecutionId: "crawl", CollectionRef: &configV1.ConfigRef{Id: "collection"}})
				if err != nil {
					t.Fatal(err)
				}
			}
			const oldID = "orphaned-attempt-0001"
			begin(old, 1, oldID)
			begin(old, 2, "orphaned-attempt-0002")
			finish := &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: oldID}
			var published *recorderV1.FinishHarvestReply
			var fetched chan struct{}
			if timedOut {
				origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "recorded") }))
				defer origin.Close()
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				go func() {
					if err := proxy.Serve(ln); err != nil && !errors.Is(err, net.ErrClosed) {
						t.Errorf("serve recorder proxy: %v", err)
					}
				}()
				proxyURL, _ := url.Parse("http://" + ln.Addr().String())
				transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
				defer transport.CloseIdleConnections()
				fetched = make(chan struct{})
				go func() {
					defer close(fetched)
					req, _ := http.NewRequestWithContext(ctx, http.MethodGet, origin.URL, nil)
					req.Header.Set(constants.HeaderAttemptId, oldID)
					resp, err := (&http.Client{Transport: transport}).Do(req)
					if err == nil {
						_, _ = io.Copy(io.Discard, resp.Body)
						_ = resp.Body.Close()
					}
				}()
				select {
				case <-logs.entered:
				case <-ctx.Done():
					t.Fatal("request never reached crawl-log persistence")
				}
				published, err = recorderV1.UploadFinish(ctx, old, finish, nil)
				if err != nil || published.Reusable || published.GetFailure().GetCode() != -5 {
					t.Fatalf("finish = %v, %v", published, err)
				}
			}
			if err := oldConn.Close(); err != nil {
				t.Fatal(err)
			}
			_, fresh := connect()
			if timedOut {
				if _, err := RecoverSlots(ctx, fresh, 2, time.Second); status.Code(err) != codes.DeadlineExceeded {
					t.Fatalf("recovered undrained work: %v", err)
				}
				unblock()
				select {
				case <-fetched:
				case <-ctx.Done():
					t.Fatal("old request did not terminate")
				}
			}
			tokens, err = RecoverSlots(ctx, fresh, 2, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			if timedOut {
				replay, err := recorderV1.UploadFinish(ctx, fresh, finish, nil)
				if err != nil || !proto.Equal(published, replay) {
					t.Fatalf("replay changed: %v, %v", replay, err)
				}
				if len(logs.CrawlLogs()) != 1 {
					t.Fatal("lost late crawl log")
				}
			}
			begin(fresh, 1, "restarted-attempt-0001")
			begin(fresh, 2, "restarted-attempt-0002")
		})
	}
}

type blockedCrawlLogs struct {
	testutil.LogWriterMock
	entered, release chan struct{}
}

func (l *blockedCrawlLogs) WriteCrawlLog(ctx context.Context, cl *logV1.CrawlLog) error {
	close(l.entered)
	// Deliberately outlive the deadline to model late terminal work.
	<-l.release
	return l.LogWriterMock.WriteCrawlLog(ctx, cl)
}

type blockedRecoveryClient struct {
	recorderV1.RecorderControlClient
	slots []int32
}

func (c *blockedRecoveryClient) RecoverSlot(ctx context.Context, req *recorderV1.RecoverSlotRequest, _ ...grpc.CallOption) (*recorderV1.RecoverSlotReply, error) {
	c.slots = append(c.slots, req.ProxyId)
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestStartupRecoveryIsBoundedAndFailsClosed(t *testing.T) {
	client := &blockedRecoveryClient{}
	if _, err := RecoverSlots(t.Context(), client, 2, time.Millisecond); err == nil {
		t.Fatal("ignored failed recovery")
	}
	if len(client.slots) != 1 || client.slots[0] != 1 {
		t.Fatalf("continued after failure: %v", client.slots)
	}
}
