package recorderproxy_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/recorderproxy"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestRecoveredSlotRejectsOldBrowserTraffic(t *testing.T) {
	for _, secure := range []bool{false, true} {
		t.Run(map[bool]string{false: "http", true: "https"}[secure], func(t *testing.T) {
			services := testutil.NewGrpcServiceMock()
			defer services.Close()
			p := recorderproxy.NewRecorderProxy(1, services.ContentWriterClient, services.DnsResolverClient, "")
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			go p.Serve(ln)
			defer p.Shutdown(context.Background())
			control := recorderproxy.NewControl(services, p)
			var requests atomic.Int32
			origin := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Header.Get(constants.HeaderAttemptId) != "" || r.Header.Get(constants.HeaderMainDocument) != "" {
					t.Error("attempt token leaked to origin")
				}
				_, _ = io.WriteString(w, "recorded")
			}))
			if secure {
				origin.StartTLS()
			} else {
				origin.Start()
			}
			defer origin.Close()
			proxyURL, _ := url.Parse("http://" + ln.Addr().String())
			transport := &http.Transport{Proxy: http.ProxyURL(proxyURL), TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
			defer transport.CloseIdleConnections()
			client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
			begin := func(id string) {
				t.Helper()
				recovery, recoveryErr := control.RecoverSlot(t.Context(), &recorderV1.RecoverSlotRequest{ProxyId: 1})
				if recoveryErr != nil {
					t.Fatal(recoveryErr)
				}
				_, err := control.BeginHarvest(t.Context(), &recorderV1.BeginHarvestRequest{AdmissionToken: recovery.NextToken, ProxyId: 1, AttemptId: id, CrawlExecutionId: "crawl", CollectionRef: &configV1.ConfigRef{Id: "collection"}})
				if err != nil {
					t.Fatal(err)
				}
			}
			fetch := func(id string, want int) {
				t.Helper()
				req, _ := http.NewRequest(http.MethodGet, origin.URL, nil)
				req.Header.Set(constants.HeaderAttemptId, id)
				req.Header.Set(constants.HeaderMainDocument, id)
				resp, err := client.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if resp.StatusCode != want {
					t.Fatalf("status = %d, want %d", resp.StatusCode, want)
				}
			}
			const oldID, newID = "old-controller-attempt-0001", "new-controller-attempt-0002"
			begin(oldID)
			fetch(oldID, http.StatusOK)
			if _, err := control.RecoverSlot(t.Context(), &recorderV1.RecoverSlotRequest{ProxyId: 1}); err != nil {
				t.Fatal(err)
			}
			begin(newID)
			// Even malformed traffic from an unidentified connection must not
			// become a connection-level failure of the new harvest.
			unidentified, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
			if err != nil {
				t.Fatal(err)
			}
			_ = unidentified.SetDeadline(time.Now().Add(time.Second))
			_, _ = io.WriteString(unidentified, "not an HTTP request\r\n\r\n")
			_, _ = io.Copy(io.Discard, unidentified)
			_ = unidentified.Close()
			// The old browser reconnects to the same listening port after recovery.
			fetch(oldID, http.StatusForbidden)
			fetch("", http.StatusForbidden)
			fetch(newID, http.StatusOK)
			reply, err := control.FinishPage(t.Context(), &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: newID}, nil)
			if err != nil || reply.Failure != nil || !reply.Reusable || reply.UriCount != 1 {
				t.Fatalf("new attempt contaminated: %v, %v", reply, err)
			}
			for _, write := range services.Requests.ContentWriterRequests {
				if header := write.GetProtocolHeader(); header != nil && (strings.Contains(string(header.Data), constants.HeaderMainDocument) || strings.Contains(string(header.Data), constants.HeaderAttemptId)) {
					t.Fatal("internal header recorded")
				}
			}
			if requests.Load() != 2 || len(services.CrawlLogs()) != 2 {
				t.Fatal("stale traffic reached the origin or ledger")
			}
		})
	}
}

func TestHarvestHTTPBoundary(t *testing.T) {
	for _, secure := range []bool{false, true} {
		for _, scenario := range []string{"redirect", "post-redirect", "http-error", "stream", "storage-error", "transport-error"} {
			t.Run(scenario+map[bool]string{true: "-https", false: "-http"}[secure], func(t *testing.T) {
				services := testutil.NewGrpcServiceMock()
				defer services.Close()
				p := recorderproxy.NewRecorderProxy(1, services.ContentWriterClient, services.DnsResolverClient, "", recorderproxy.WithFinalizationTimeout(time.Second))
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				go p.Serve(ln)
				defer p.Shutdown(context.Background())
				listener := bufconn.Listen(1024 * 1024)
				server := grpc.NewServer()
				recorderV1.RegisterRecorderControlServer(server, recorderproxy.NewControl(services, p))
				go server.Serve(listener)
				defer server.Stop()
				cc, err := grpc.NewClient("passthrough:///control", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
				if err != nil {
					t.Fatal(err)
				}
				defer cc.Close()
				control := recorderV1.NewRecorderControlClient(cc)
				id := "attempt-000000000001"
				recovery, recoveryErr := control.RecoverSlot(t.Context(), &recorderV1.RecoverSlotRequest{ProxyId: 1})
				if recoveryErr != nil {
					t.Fatal(recoveryErr)
				}
				_, err = control.BeginHarvest(t.Context(), &recorderV1.BeginHarvestRequest{AdmissionToken: recovery.NextToken, ProxyId: 1, AttemptId: id, DiscoveryPath: "L", CrawlExecutionId: "crawl", CollectionRef: &configV1.ConfigRef{Id: "collection"}})
				if err != nil {
					t.Fatal(err)
				}
				origin := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasPrefix(scenario, "post") && r.Method == "POST" {
						w.Header().Set("Location", r.URL.String())
						w.WriteHeader(303)
						return
					}
					if scenario == "redirect" && r.Header.Get("Cookie") == "" {
						w.Header().Set("Location", r.URL.String())
						w.Header().Set("Set-Cookie", "redirected=yes")
						w.WriteHeader(302)
						return
					}
					if scenario == "transport-error" {
						w.Header().Set("Content-Length", "1000")
						w.Write([]byte("short"))
						return
					}
					if scenario == "http-error" {
						w.WriteHeader(404)
					}
					w.Write([]byte("recorded"))
					if scenario == "stream" {
						w.(http.Flusher).Flush()
						<-r.Context().Done()
					}
				}))
				if secure {
					origin.StartTLS()
				} else {
					origin.Start()
				}
				defer origin.Close()
				proxyURL, _ := url.Parse("http://" + ln.Addr().String())
				transport := &http.Transport{Proxy: http.ProxyURL(proxyURL), TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
				client := &http.Client{Transport: transport, Timeout: 5 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
					req.Header.Set("Cookie", "redirected=yes")
					return nil
				}}
				defer transport.CloseIdleConnections()
				target := origin.URL
				if scenario == "storage-error" {
					target += "/cwerr"
				}
				method := "GET"
				if scenario == "post-redirect" {
					method = "POST"
				}
				request, _ := http.NewRequest(method, target, strings.NewReader(""))
				request.Header.Set(constants.HeaderAttemptId, id)
				request.Header.Set(constants.HeaderMainDocument, id)
				request.Header.Set("Sec-Fetch-Mode", "navigate")
				response, requestErr := client.Do(request)
				if requestErr == nil {
					if scenario != "stream" {
						_, _ = io.Copy(io.Discard, response.Body)
						response.Body.Close()
					} else {
						defer response.Body.Close()
					}
				}
				ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
				defer cancel()
				finish, err := recorderV1.UploadFinish(ctx, control, &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: id}, nil)
				if err != nil {
					t.Fatal(err)
				}
				results := services.CrawlLogs()
				want := 1
				if scenario == "redirect" || scenario == "post-redirect" {
					want = 2
				}
				if len(results) != want || !finish.Reusable {
					t.Fatalf("results=%v summary=%v requestErr=%v", results, finish, requestErr)
				}
				shouldFail := scenario == "storage-error" || scenario == "transport-error"
				if (finish.Failure != nil) != shouldFail {
					t.Fatalf("summary=%v results=%v", finish, results)
				}
				for _, r := range results {
					wantPath := "L"
					if (scenario == "redirect" || scenario == "post-redirect") && r.StatusCode == http.StatusOK {
						wantPath = "LR"
					}
					if r.DiscoveryPath != wantPath {
						t.Fatalf("path = %q, want %q for %v", r.DiscoveryPath, wantPath, r)
					}
					if !shouldFail && scenario != "stream" && r.WarcId == "" {
						t.Fatal("missing confirmed WARC", r)
					}
				}
				if scenario == "post-redirect" && results[0].Method == results[1].Method {
					t.Fatal("redirect transfers collapsed")
				}
			})
		}
	}
}

func TestSubresourceFailurePreservesPageHTTP(t *testing.T) {
	for _, secure := range []bool{false, true} {
		for _, scenario := range []string{"connection-refused", "truncated", "unknown-body", "storage-error"} {
			t.Run(fmt.Sprintf("https=%v/%s", secure, scenario), func(t *testing.T) {
				services := testutil.NewGrpcServiceMock()
				defer services.Close()
				p := recorderproxy.NewRecorderProxy(1, services.ContentWriterClient, services.DnsResolverClient, "", recorderproxy.WithFinalizationTimeout(time.Second))
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				go p.Serve(ln)
				defer p.Shutdown(context.Background())
				control := recorderproxy.NewControl(services, p)
				origin := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if req.URL.Path == "/main" {
						_, _ = io.WriteString(w, "recorded main")
						return
					}
					switch scenario {
					case "truncated":
						w.Header().Set("Content-Length", "1000")
						_, _ = io.WriteString(w, "short")
					case "unknown-body":
						conn, _, hijackErr := w.(http.Hijacker).Hijack()
						if hijackErr != nil {
							t.Error(hijackErr)
							return
						}
						defer conn.Close()
						_, _ = io.WriteString(conn, "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\ninvalid\r\n")
					default:
						_, _ = io.WriteString(w, "image")
					}
				}))
				if secure {
					origin.StartTLS()
				} else {
					origin.Start()
				}
				defer origin.Close()
				mainURI, childURI := origin.URL+"/main", origin.URL+"/image"
				if scenario == "storage-error" {
					childURI = origin.URL + "/cwerr"
				}
				if scenario == "connection-refused" {
					dead, err := net.Listen("tcp", "127.0.0.1:0")
					if err != nil {
						t.Fatal(err)
					}
					u, _ := url.Parse(childURI)
					u.Host = dead.Addr().String()
					childURI = u.String()
					_ = dead.Close()
				}
				recovery, recoveryErr := control.RecoverSlot(t.Context(), &recorderV1.RecoverSlotRequest{ProxyId: 1})
				if recoveryErr != nil {
					t.Fatal(recoveryErr)
				}
				spec := &recorderV1.BeginHarvestRequest{AdmissionToken: recovery.NextToken, ProxyId: 1, AttemptId: "subresource-http-attempt-0001", CrawlExecutionId: "crawl", RequestedUri: mainURI, CollectionRef: &configV1.ConfigRef{Id: "collection"}}
				if _, err := control.BeginHarvest(t.Context(), spec); err != nil {
					t.Fatal(err)
				}
				proxyURL, _ := url.Parse("http://" + ln.Addr().String())
				transport := &http.Transport{Proxy: http.ProxyURL(proxyURL), TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
				defer transport.CloseIdleConnections()
				client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
				for _, uri := range []string{mainURI, childURI} {
					req, _ := http.NewRequest(http.MethodGet, uri, nil)
					req.Header.Set(constants.HeaderAttemptId, spec.AttemptId)
					req.Header.Set("Sec-Fetch-Dest", "image")
					if uri == mainURI {
						req.Header.Set(constants.HeaderMainDocument, spec.AttemptId)
						req.Header.Set("Sec-Fetch-Mode", "navigate")
					}
					response, fetchErr := client.Do(req)
					if response != nil {
						_, bodyErr := io.ReadAll(response.Body)
						_ = response.Body.Close()
						if fetchErr == nil {
							fetchErr = bodyErr
						}
					}
					if uri == mainURI && fetchErr != nil {
						t.Fatal(fetchErr)
					}
				}
				reply, err := control.FinishPage(t.Context(), &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: spec.AttemptId, PageUri: &mainURI}, []string{origin.URL + "/link"})
				if err != nil || (reply.GetFailure() != nil) != (scenario == "storage-error") || !reply.GetReusable() {
					t.Fatalf("reply=%v err=%v", reply, err)
				}
				logs, pages := services.CrawlLogs(), services.PageLogs()
				if len(logs) != 2 || len(pages) != 1 || len(pages[0].Resource) != 2 || len(pages[0].Outlink) != 1 {
					t.Fatalf("crawl logs=%v page logs=%v", logs, pages)
				}
				if reply.Main.GetRequestedUri() != mainURI || reply.Main.GetWarcId() == "" || reply.UriCount != 1 || reply.BytesDownloaded != reply.Main.Size {
					t.Fatalf("main or totals lost: %v", reply)
				}
				wantCode := int32(-2)
				if scenario == "truncated" {
					wantCode = -404
				}
				if scenario == "unknown-body" || scenario == "storage-error" {
					wantCode = -5
				}
				resource := pages[0].Resource[1]
				if resource.Uri != childURI || resource.GetError().GetCode() != wantCode || resource.WarcId != "" {
					t.Fatalf("failed resource lost: %v", resource)
				}
			})
		}
	}
}
