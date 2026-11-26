package recorderproxy_test

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/recorderproxy"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
)

// An optional local Squid must be able to reach this test's loopback origins.
func captureStream(t testing.TB, secure bool, mode string) func() int {
	t.Helper()
	entity := bytes.Repeat([]byte("abcdef01"), 8192)
	var requests atomic.Int32
	origin := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seq := requests.Add(1)
		w.Header().Set("Cache-Control", "no-store")
		if mode == "fixed" {
			w.Header().Set("Content-Length", fmt.Sprint(len(entity)))
		} else {
			if mode == "trailers" {
				w.Header().Set("Trailer", "X-Origin")
			}
			w.(http.Flusher).Flush()
		}
		_, _ = w.Write(entity)
		if mode == "trailers" {
			w.Header().Set("X-Origin", fmt.Sprint(seq))
		}
	}))
	if secure {
		origin.StartTLS()
	} else {
		origin.Start()
	}
	t.Cleanup(origin.Close)
	services := testutil.NewGrpcServiceMock()
	t.Cleanup(services.Close)
	p := recorderproxy.NewRecorderProxy(1, services.ContentWriterClient, services.DnsResolverClient, os.Getenv("VEIDEMANN_TEST_SQUID"))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go p.Serve(ln)
	t.Cleanup(func() { p.Shutdown(context.Background()) })
	control := recorderproxy.NewControl(services, p)
	recovery, err := control.RecoverSlot(t.Context(), &recorderV1.RecoverSlotRequest{ProxyId: 1})
	if err != nil {
		t.Fatal(err)
	}
	const attempt = "stream-capture-000001"
	_, err = control.BeginHarvest(t.Context(), &recorderV1.BeginHarvestRequest{AdmissionToken: recovery.NextToken, ProxyId: 1, AttemptId: attempt, CrawlExecutionId: "crawl", CollectionRef: &configV1.ConfigRef{Id: "c1"}})
	if err != nil {
		t.Fatal(err)
	}
	proxyURL := &url.URL{Scheme: "http", Host: ln.Addr().String()}
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL), TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	t.Cleanup(client.CloseIdleConnections)
	return func() int {
		before := requests.Load()
		req, err := http.NewRequest(http.MethodGet, origin.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set(constants.HeaderAttemptId, attempt)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(entity, body) {
			t.Fatal("browser entity differs")
		}
		if requests.Load() != before+1 {
			t.Fatal("expected one origin fetch")
		}
		var header, payload []byte
		messages := 0
		completed := 0
		for _, msg := range services.Requests.ContentWriterRequests {
			if h := msg.GetProtocolHeader(); h != nil && h.RecordNum == 1 {
				header = h.Data
			}
			if p := msg.GetPayload(); p != nil && p.RecordNum == 1 {
				payload = append(payload, p.Data...)
				messages++
			}
			if m := msg.GetMeta(); m != nil {
				completed++
				r := m.RecordMeta[1]
				if int64(len(header)+len(payload)) != r.Size {
					t.Fatal("wrong archived size")
				}
				hash := sha1.New()
				hash.Write(header)
				hash.Write(payload)
				if fmt.Sprintf("sha1:%x", hash.Sum(nil)) != r.BlockDigest {
					t.Fatal("wrong archived block digest")
				}
			}
		}
		if completed != 1 {
			t.Fatalf("got %d completed writes for one capture", completed)
		}
		if len(header) == 0 {
			t.Fatal("missing response header")
		}
		if strings.Contains(strings.ToLower(string(header)), "\r\ntransfer-encoding:") {
			t.Fatal("archived transfer encoding")
		}
		if strings.Contains(strings.ToLower(string(header)), "\r\ntrailer:") {
			t.Fatal("archived trailer declaration")
		}
		if !bytes.Equal(entity, payload) {
			t.Fatal("archived entity differs")
		}
		services.Requests.ContentWriterRequests = nil
		return messages
	}
}

func TestCaptureStream(t *testing.T) {
	for _, secure := range []bool{false, true} {
		for _, mode := range []string{"fixed", "chunked", "trailers"} {
			t.Run(fmt.Sprintf("%s/tls=%t", mode, secure), func(t *testing.T) {
				fetch := captureStream(t, secure, mode)
				fetch()
				fetch()
			})
		}
	}
}

func BenchmarkCaptureStream(b *testing.B) {
	for _, mode := range []string{"fixed", "chunked", "trailers"} {
		b.Run(mode, func(b *testing.B) {
			fetch := captureStream(b, false, mode)
			b.SetBytes(64 << 10)
			b.ReportAllocs()
			count := 0
			for b.Loop() {
				count += fetch()
			}
			b.ReportMetric(float64(count)/float64(b.N), "payload-msgs/op")
		})
	}
}
