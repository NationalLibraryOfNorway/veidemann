package recorderproxy_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	writerV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/mitmcert"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/recorderproxy"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
)

func init() { _ = logger.InitLog("warn", "text", false) }

// These fixtures exercise the same HTTP/HTTPS direct and chained transports as
// browser harvests, with metadata supplied exclusively by BeginHarvest.
func TestRecorderProxy(t *testing.T)             { testRecordingTransport(t, false) }
func TestRecorderProxyThroughProxy(t *testing.T) { testRecordingTransport(t, true) }
func testRecordingTransport(t *testing.T, chained bool) {
	for _, secure := range []bool{false, true} {
		for _, scenario := range []string{"success", "not-found", "cache-hit", "storage-error", "truncated", "client-timeout", "connection-refused", "upstream-tls-failure"} {
			if scenario == "upstream-tls-failure" && !secure {
				continue
			}
			t.Run(map[bool]string{false: "http", true: "https"}[secure]+"/"+scenario, func(t *testing.T) {
				origins := testutil.NewHttpServers(t)
				defer origins.Close()
				services := testutil.NewGrpcServiceMock()
				defer services.Close()
				nextProxy := ""
				if chained {
					secondary, addr := testutil.NewSecondaryProxy(t, origins)
					defer secondary.Close()
					nextProxy = addr
				}
				p := recorderproxy.NewRecorderProxy(1, services.ContentWriterClient, services.DnsResolverClient, nextProxy, recorderproxy.WithFinalizationTimeout(time.Second))
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				go p.Serve(ln)
				defer p.Shutdown(context.Background())
				control := recorderproxy.NewControl(services, p)
				recovery, recoveryErr := control.RecoverSlot(t.Context(), &recorderV1.RecoverSlotRequest{ProxyId: 1})
				if recoveryErr != nil {
					t.Fatal(recoveryErr)
				}
				spec := &recorderV1.BeginHarvestRequest{AdmissionToken: recovery.NextToken, ProxyId: 1, AttemptId: "integration-attempt-0001", CrawlExecutionId: "execution", JobExecutionId: "job", CollectionRef: &configV1.ConfigRef{Kind: configV1.Kind_collection, Id: "collection"}}
				if _, err := control.BeginHarvest(t.Context(), spec); err != nil {
					t.Fatal(err)
				}
				proxyURL, _ := url.Parse("http://" + ln.Addr().String())
				transport := &http.Transport{Proxy: http.ProxyURL(proxyURL), TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
				defer transport.CloseIdleConnections()
				client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
				uri := origins.SrvHttp.URL + "/a"
				if secure {
					uri = origins.SrvHttps.URL + "/b"
				}
				wantStatus := 200
				wantError := int32(0)
				switch scenario {
				case "not-found":
					uri += "-missing"
					wantStatus = 404
				case "cache-hit":
					uri = strings.TrimSuffix(uri, "a")
					uri = strings.TrimSuffix(uri, "b")
					uri += "cached"
				case "storage-error":
					uri = strings.TrimSuffix(uri, "a")
					uri = strings.TrimSuffix(uri, "b")
					uri += "cwerr"
					wantError = -5
				case "client-timeout":
					uri = strings.TrimSuffix(uri, "a")
					uri = strings.TrimSuffix(uri, "b")
					uri += "slow"
					client.Timeout = 500 * time.Millisecond
					wantError = -5011
				case "connection-refused":
					closed, e := net.Listen("tcp", "127.0.0.1:0")
					if e != nil {
						t.Fatal(e)
					}
					addr := closed.Addr().String()
					closed.Close()
					scheme := "http"
					if secure {
						scheme = "https"
					}
					uri = scheme + "://" + addr + "/resource"
					wantStatus = 503
					wantError = -2
				case "upstream-tls-failure":
					uri = origins.SrvHttpsBrokenTLS.URL + "/b"
					wantStatus = 503
					wantError = -2
				case "truncated":
					origin := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.Header().Set("Content-Length", "100")
						io.WriteString(w, "short")
					}))
					if secure {
						origin.StartTLS()
					} else {
						origin.Start()
					}
					defer origin.Close()
					uri = origin.URL
					wantError = -404
				}
				request, _ := http.NewRequest(http.MethodGet, uri, nil)
				request.Header.Set(constants.HeaderAttemptId, spec.AttemptId)
				if scenario == "success" {
					request.Header.Set(constants.HeaderCrawlExecutionId, "forged-execution")
					request.Header.Set(constants.HeaderJobExecutionId, "forged-job")
					request.Header.Set(constants.HeaderCollectionId, "forged-collection")
				}
				response, fetchErr := client.Do(request)
				if response != nil {
					_, bodyErr := io.ReadAll(response.Body)
					response.Body.Close()
					if fetchErr == nil {
						fetchErr = bodyErr
					}
					if scenario != "storage-error" && scenario != "truncated" && scenario != "client-timeout" && response.StatusCode != wantStatus {
						t.Fatalf("HTTP status %d want %d", response.StatusCode, wantStatus)
					}
				}
				if scenario == "success" && fetchErr != nil {
					t.Fatal(fetchErr)
				}
				reply, err := control.FinishPage(t.Context(), &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: spec.AttemptId}, nil)
				if err != nil {
					t.Fatal(err)
				}
				logs := services.CrawlLogs()
				if len(logs) != 1 || !reply.Reusable {
					t.Fatalf("logs=%v reply=%v", logs, reply)
				}
				cl := logs[0]
				// These requests carry no main-navigation metadata. Origin failures
				// remain resource outcomes; recording failures fail the attempt.
				if (reply.Failure != nil) != (scenario == "storage-error") {
					t.Fatalf("unexpected attempt failure: %v", reply)
				}

				if cl.ExecutionId != "execution" || cl.JobExecutionId != "job" {
					t.Fatal(cl)
				}
				if wantError != 0 {
					if cl.GetError().GetCode() != wantError {
						t.Fatalf("terminal error %v want %d", cl, wantError)
					}
				} else if cl.GetError() != nil {
					t.Fatal(cl)
				}
				if scenario == "success" || scenario == "not-found" {
					headers := map[int32]bool{}
					var meta *writerV1.WriteRequestMeta
					for index, message := range services.Requests.ContentWriterRequests {
						switch value := message.Value.(type) {
						case *writerV1.WriteRequest_ProtocolHeader:
							if headers[value.ProtocolHeader.RecordNum] {
								t.Fatal("duplicate protocol header")
							}
							headers[value.ProtocolHeader.RecordNum] = true
							if bytes.Contains(value.ProtocolHeader.Data, []byte("forged-")) {
								t.Fatal("crawler header was archived")
							}
							if bytes.Contains(bytes.ToLower(value.ProtocolHeader.Data), []byte(strings.ToLower(constants.HeaderAttemptId))) {
								t.Fatal("attempt token was archived")
							}
						case *writerV1.WriteRequest_Payload:
							if !headers[value.Payload.RecordNum] {
								t.Fatal("payload preceded its header")
							}
						case *writerV1.WriteRequest_Meta:
							if index != len(services.Requests.ContentWriterRequests)-1 {
								t.Fatal("final metadata did not terminate stream")
							}
							meta = value.Meta
						default:
							t.Fatal("unexpected ContentWriter terminal message", message)
						}
					}
					if !headers[0] || !headers[1] || len(headers) != 2 || meta == nil || meta.ExecutionId != "execution" || meta.CollectionRef.Id != "collection" || meta.TargetUri != uri || meta.RecordMeta[1].BlockDigest != cl.BlockDigest || meta.RecordMeta[1].Size != cl.Size {
						t.Fatal("request/response stream lost attribution or confirmation", meta, cl)
					}
					if cl.WarcId == "" || cl.StorageRef == "" || cl.BlockDigest == "" || cl.PayloadDigest == "" {
						t.Fatal("missing confirmed archive", cl)
					}
				}
				if scenario == "cache-hit" && cl.WarcId != "" {
					t.Fatal("cache hit was recorded")
				}
				for _, dns := range services.Requests.DnsResolverRequests {
					if dns.ExecutionId != "execution" || dns.CollectionRef.Id != "collection" {
						t.Fatal(dns)
					}
				}
			})
		}
	}
}

func TestEveryHarvestListenerRejectsWithoutAttempt(t *testing.T) {
	services := testutil.NewGrpcServiceMock()
	defer services.Close()
	for _, slot := range []int{1, 2, 3} {
		p := recorderproxy.NewRecorderProxy(slot, services.ContentWriterClient, services.DnsResolverClient, "")
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		go p.Serve(ln)
		conn, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
		if err != nil {
			t.Fatal(err)
		}
		conn.SetDeadline(time.Now().Add(time.Second))
		_, _ = io.WriteString(conn, "GET http://example.com/ HTTP/1.1\r\nHost: example.com\r\n\r\n")
		b := make([]byte, 1)
		if n, _ := conn.Read(b); n != 0 {
			t.Fatal("idle slot accepted HTTP")
		}
		conn.Close()
		p.Shutdown(context.Background())
	}
	if len(services.Requests.DnsResolverRequests) != 0 || len(services.Requests.ContentWriterRequests) != 0 {
		t.Fatal("idle connection reached services")
	}
}

func TestRecorderProxyChainedTLSIdentity(t *testing.T) {
	services := testutil.NewGrpcServiceMock()
	defer services.Close()
	origins := testutil.NewHttpServers(t)
	defer origins.Close()
	secondary, addr := testutil.NewSecondaryProxy(t, origins)
	defer secondary.Close()
	cert, key, err := mitmcert.Generate(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	identity, err := mitmcert.ParseIdentity(cert, key)
	if err != nil {
		t.Fatal(err)
	}
	p := recorderproxy.NewRecorderProxy(1, services.ContentWriterClient, services.DnsResolverClient, addr, recorderproxy.WithMITMIdentity(identity))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go p.Serve(ln)
	defer p.Shutdown(context.Background())
	control := recorderproxy.NewControl(services, p)
	recovery, recoveryErr := control.RecoverSlot(t.Context(), &recorderV1.RecoverSlotRequest{ProxyId: 1})
	if recoveryErr != nil {
		t.Fatal(recoveryErr)
	}
	_, err = control.BeginHarvest(t.Context(), &recorderV1.BeginHarvestRequest{AdmissionToken: recovery.NextToken, ProxyId: 1, AttemptId: "identity-attempt-0001", CrawlExecutionId: "execution", CollectionRef: &configV1.ConfigRef{Id: "collection"}})
	if err != nil {
		t.Fatal(err)
	}
	proxyURL, _ := url.Parse("http://" + ln.Addr().String())
	tr := &http.Transport{Proxy: http.ProxyURL(proxyURL), TLSClientConfig: &tls.Config{InsecureSkipVerify: true, VerifyConnection: func(state tls.ConnectionState) error {
		if string(state.PeerCertificates[0].RawSubjectPublicKeyInfo) != string(identity.Certificate().RawSubjectPublicKeyInfo) {
			t.Error("wrong MITM identity")
		}
		return nil
	}}}
	defer tr.CloseIdleConnections()
	req, _ := http.NewRequest(http.MethodGet, origins.SrvHttps.URL+"/b", nil)
	req.Header.Set(constants.HeaderAttemptId, "identity-attempt-0001")
	resp, err := (&http.Client{Transport: tr, Timeout: time.Second}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}
