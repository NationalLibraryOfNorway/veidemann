package robots

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	writerV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	dnsV1 "github.com/NationalLibraryOfNorway/veidemann/api/dnsresolver/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/google/uuid"
	"github.com/nationallibraryofnorway/gowarc/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type storedPair struct {
	warc            []byte
	meta            *writerV1.WriteRequestMeta
	headers, bodies map[int32][]byte
}
type recordingServices struct {
	failPhase string
	dnsV1.UnimplementedDnsResolverServer
	writerV1.UnimplementedContentWriterServer
	logV1.UnimplementedLogServer
	warcMu                                                  sync.Mutex
	mu                                                      sync.Mutex
	dnsRequests                                             []*dnsV1.ResolveRequest
	pairs                                                   []storedPair
	logs                                                    []*logV1.CrawlLog
	dnsError, negative, writerError, invalidReply, logError bool
	writerStarted, releaseWriter                            chan struct{}
	logStarted, releaseLog                                  chan struct{}
}

func (s *recordingServices) Resolve(_ context.Context, req *dnsV1.ResolveRequest) (*dnsV1.ResolveReply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dnsRequests = append(s.dnsRequests, proto.Clone(req).(*dnsV1.ResolveRequest))
	if s.dnsError {
		return nil, status.Error(codes.Unavailable, "DNS down")
	}
	if s.negative {
		return &dnsV1.ResolveReply{Error: &commonsV1.Error{Code: 3}}, nil
	}
	return &dnsV1.ResolveReply{TextualIp: "127.0.0.1"}, nil
}
func (s *recordingServices) Write(stream writerV1.ContentWriter_WriteServer) error {
	pair := storedPair{headers: map[int32][]byte{}, bodies: map[int32][]byte{}}
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch v := req.Value.(type) {
		case *writerV1.WriteRequest_Meta:
			pair.meta = v.Meta
		case *writerV1.WriteRequest_ProtocolHeader:
			pair.headers[v.ProtocolHeader.RecordNum] = append(pair.headers[v.ProtocolHeader.RecordNum], v.ProtocolHeader.Data...)
		case *writerV1.WriteRequest_Payload:
			pair.bodies[v.Payload.RecordNum] = append(pair.bodies[v.Payload.RecordNum], v.Payload.Data...)
		default:
			return status.Error(codes.InvalidArgument, "unexpected message")
		}
	}
	if s.writerStarted != nil {
		select {
		case s.writerStarted <- struct{}{}:
		default:
		}
		select {
		case <-s.releaseWriter:
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
	if s.writerError || (s.failPhase == "writer" && strings.HasSuffix(pair.meta.TargetUri, "/fail")) {
		return status.Error(codes.Unavailable, "storage down")
	}
	s.warcMu.Lock()
	defer s.warcMu.Unlock()
	reply := &writerV1.WriteReply{Meta: &writerV1.WriteResponseMeta{RecordMeta: map[int32]*writerV1.WriteResponseMeta_RecordMeta{}}}
	for i := int32(0); i <= 1; i++ {
		m := pair.meta.RecordMeta[i]
		if m == nil || m.BlockDigest != digest(pair.headers[i], pair.bodies[i]) || m.PayloadDigest != digest(pair.bodies[i]) || m.Size != int64(len(pair.headers[i])+len(pair.bodies[i])) {
			return status.Error(codes.InvalidArgument, "invalid digest or size")
		}
		typ := gowarc.Request
		if i == 1 {
			typ = gowarc.Response
		}
		rb := gowarc.NewRecordBuilder(typ, gowarc.WithStrictValidation(), gowarc.WithFixDigest(false))
		rb.AddWarcHeader(gowarc.WarcRecordID, "<urn:uuid:"+uuid.NewSHA1(uuid.NameSpaceURL, []byte(fmt.Sprintf("%s/%s/%d/%d", pair.meta.ExecutionId, pair.meta.TargetUri, i, len(s.pairs)))).String()+">")
		rb.AddWarcHeader(gowarc.WarcTargetURI, pair.meta.TargetUri)
		rb.AddWarcHeaderTime(gowarc.WarcDate, pair.meta.FetchTimeStamp.AsTime())
		rb.AddWarcHeader(gowarc.WarcIPAddress, pair.meta.IpAddress)
		rb.AddWarcHeader(gowarc.ContentType, m.RecordContentType)
		rb.AddWarcHeaderInt64(gowarc.ContentLength, m.Size)
		rb.AddWarcHeader(gowarc.WarcBlockDigest, m.BlockDigest)
		rb.AddWarcHeader(gowarc.WarcPayloadDigest, m.PayloadDigest)
		if _, err := rb.Write(pair.headers[i]); err != nil {
			return errors.Join(err, rb.Close())
		}
		if _, err := rb.Write(pair.bodies[i]); err != nil {
			return errors.Join(err, rb.Close())
		}
		record, validation, err := rb.Build()
		if err != nil || len(validation) != 0 {
			if record != nil {
				_ = record.Close()
			} else {
				_ = rb.Close()
			}
			return status.Errorf(codes.InvalidArgument, "WARC validation: %v %v", err, validation)
		}
		var serialized bytes.Buffer
		_, _, err = gowarc.NewMarshaler().Marshal(&serialized, record, 0)
		if err := errors.Join(err, record.Close()); err != nil {
			return err
		}
		offset := len(pair.warc)
		pair.warc = append(pair.warc, serialized.Bytes()...)
		reply.Meta.RecordMeta[i] = &writerV1.WriteResponseMeta_RecordMeta{RecordNum: i, Type: m.Type, WarcId: strings.TrimSuffix(strings.TrimPrefix(record.RecordId(), "<urn:uuid:"), ">"), StorageRef: fmt.Sprintf("warcfile:fixture.warc:%d", offset), CollectionFinalName: pair.meta.CollectionRef.Id, BlockDigest: m.BlockDigest, PayloadDigest: m.PayloadDigest}

	}
	if s.invalidReply {
		reply.Meta.RecordMeta[0].WarcId = ""
	}
	s.mu.Lock()
	s.pairs = append(s.pairs, pair)
	s.mu.Unlock()
	return stream.SendAndClose(reply)
}
func (s *recordingServices) WriteCrawlLog(stream logV1.Log_WriteCrawlLogServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&emptypb.Empty{})
		}
		if err != nil {
			return err
		}
		if s.logStarted != nil {
			s.logStarted <- struct{}{}
			select {
			case <-s.releaseLog:
			case <-stream.Context().Done():
				return stream.Context().Err()
			}
		}
		if s.logError || (s.failPhase == "log" && strings.HasSuffix(req.CrawlLog.RequestedUri, "/fail")) {
			return status.Error(codes.Unavailable, "logs down")
		}
		s.mu.Lock()
		s.logs = append(s.logs, req.CrawlLog)
		s.mu.Unlock()
	}
}
func recordingFixture(t *testing.T, s *recordingServices, secure bool, handler http.Handler) (*Evaluator, string) {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	dnsV1.RegisterDnsResolverServer(server, s)
	writerV1.RegisterContentWriterServer(server, s)
	logV1.RegisterLogServer(server, s)
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("serve gRPC: %v", err)
		}
	}()
	t.Cleanup(server.Stop)
	cc, err := grpc.NewClient("passthrough:///recording", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cc.Close() })
	origin := httptest.NewUnstartedServer(handler)
	if secure {
		origin.StartTLS()
	} else {
		origin.Start()
	}
	t.Cleanup(origin.Close)
	transport := &Transport{Base: origin.Client().Transport, DNS: dnsV1.NewDnsResolverClient(cc)}
	if transport.Base == nil {
		transport.Base = http.DefaultTransport
	}
	rec := NewRecorder(writerV1.NewContentWriterClient(cc), logV1.NewLogClient(cc), 30*time.Second)
	e := NewEvaluator(newMemoryCache(), &http.Client{Transport: transport, Timeout: time.Second}, time.Hour, time.Hour)
	e.Recorder = rec
	return e, origin.URL
}
func TestRecordingHTTPAndHTTPS(t *testing.T) {
	for _, secure := range []bool{false, true} {
		t.Run(fmt.Sprint(secure), func(t *testing.T) {
			s := &recordingServices{}
			e, uri := recordingFixture(t, s, secure, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Header.Get(collectionID) != "" || req.Header.Get(executionID) != "" || req.Header.Get(jobExecutionID) != "" {
					t.Error("metadata reached origin")
				}
				if req.Header.Get("Accept-Encoding") != "identity" || req.Header.Get("Cache-Control") != "no-cache, no-store" {
					t.Error(req.Header)
				}
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				_, _ = io.WriteString(w, blockedRules)
			}))
			req := allowedRequest(uri+"/blocked", "collection", "execution", "job")
			for i := 0; i < 2; i++ {
				allowed, err := e.IsAllowed(t.Context(), req)
				if err != nil || allowed {
					t.Fatalf("decision %v %v", allowed, err)
				}
			}
			if len(s.pairs) != 1 || len(s.logs) != 1 || len(s.dnsRequests) != 1 {
				t.Fatalf("pairs/logs/DNS %d/%d/%d", len(s.pairs), len(s.logs), len(s.dnsRequests))
			}
			pair := s.pairs[0]
			reader, err := gowarc.NewWarcFileReaderFromStream(bytes.NewReader(pair.warc), 0, gowarc.WithStrictValidation(), gowarc.WithFixDigest(false))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = reader.Close() }()
			for i := 0; i < 2; i++ {
				entry, err := reader.Next()
				if err != nil || len(entry.Validation) != 0 {
					t.Fatalf("stored WARC: %v %v", err, entry.Validation)
				}
				record := entry.WarcRecord
				typ := gowarc.Request
				if i == 1 {
					typ = gowarc.Response
				}
				raw, err := record.Block().RawBytes()
				if err != nil {
					t.Fatal(err)
				}
				data, err := io.ReadAll(raw)
				if err != nil {
					t.Fatal(err)
				}
				want := append(bytes.Clone(pair.headers[int32(i)]), pair.bodies[int32(i)]...)
				if record.Type() != typ || record.WarcHeader().Get(gowarc.WarcTargetURI) != uri+"/robots.txt" || !bytes.Equal(data, want) || record.WarcHeader().Get(gowarc.WarcBlockDigest) != digest(want) {
					t.Fatal(record)
				}
				if i == 1 && strings.TrimSuffix(strings.TrimPrefix(record.RecordId(), "<urn:uuid:"), ">") != s.logs[0].WarcId {
					t.Fatal("log reference does not match WARC")
				}
				if err := record.Close(); err != nil {
					t.Fatal(err)
				}
			}
			cl := s.logs[0]
			if string(pair.bodies[1]) != blockedRules || !bytes.HasPrefix(pair.headers[0], []byte("GET /robots.txt HTTP/1.1\r\n")) || bytes.Contains(pair.headers[0], []byte("veidemann_")) {
				t.Fatal(pair)
			}
			if cl.WarcId == "" || cl.StorageRef == "" || cl.StatusCode != 200 || cl.ExecutionId != "execution" || cl.JobExecutionId != "job" || cl.ContentType != "text/plain; charset=utf-8" || pair.meta.CollectionRef.Id != "collection" || !cl.FetchTimeStamp.IsValid() || cl.PayloadDigest != digest([]byte(blockedRules)) {
				t.Fatal(cl)
			}
			if s.dnsRequests[0].CollectionRef.Id != "collection" || s.dnsRequests[0].ExecutionId != "execution" {
				t.Fatal(s.dnsRequests)
			}
		})
	}
}
func TestRecordingBestEffortDoesNotAllowDisallowedURI(t *testing.T) {
	for _, failure := range []string{"writer", "invalid-reply", "logs"} {
		t.Run(failure, func(t *testing.T) {
			s := &recordingServices{writerError: failure == "writer", invalidReply: failure == "invalid-reply", logError: failure == "logs"}
			e, uri := recordingFixture(t, s, false, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, blockedRules)
			}))
			req := allowedRequest(uri+"/blocked", "collection", "execution", "job")
			allowed, err := e.IsAllowed(t.Context(), req)
			if err != nil || allowed {
				t.Fatalf("decision %v %v", allowed, err)
			}
			allowed, err = e.IsAllowed(t.Context(), req)
			if err != nil || allowed {
				t.Fatalf("cached decision %v %v", allowed, err)
			}
			if len(s.logs) != 0 {
				t.Fatal("unconfirmed log")
			}
		})
	}
}
func TestRecordingRedirectsAndIncompleteTransfers(t *testing.T) {
	for _, scenario := range []string{"repeated-url", "unread-redirect", "later-dns-failure", "truncated", "oversized", "oversized-headers", "http-error", "cache-hit", "unscoped"} {
		t.Run(scenario, func(t *testing.T) {
			s := &recordingServices{}
			var requests atomic.Int32
			e, uri := recordingFixture(t, s, false, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				count := requests.Add(1)
				if scenario == "repeated-url" && count < 3 {
					w.Header().Set("Location", "/robots.txt")
					w.WriteHeader(302)
					_, _ = io.WriteString(w, strings.Repeat("redirect", 100))
					return
				}
				if scenario == "unread-redirect" && req.URL.Path == "/robots.txt" {
					w.Header().Set("Location", "/final")
					w.Header().Set("Content-Length", "1000000")
					w.WriteHeader(302)
					_, _ = io.WriteString(w, "unfinished redirect")
					w.(http.Flusher).Flush()
					<-req.Context().Done()
					return
				}
				if scenario == "later-dns-failure" {
					s.mu.Lock()
					s.negative = true
					s.mu.Unlock()
					w.Header().Set("Location", "http://missing.example/robots.txt")
					w.WriteHeader(302)
					_, _ = io.WriteString(w, "redirect")
					return
				}
				if scenario == "truncated" {
					w.Header().Set("Content-Length", "999")
					_, _ = io.WriteString(w, "short")
					return
				}
				if scenario == "oversized-headers" {
					w.Header().Set("X-Oversized", strings.Repeat("x", maxExchangeBytes))
				}
				if scenario == "oversized" {
					_, _ = io.WriteString(w, blockedRules+"#"+strings.Repeat("a", maxExchangeBytes))
					return
				}
				if scenario == "http-error" {
					w.WriteHeader(404)
					_, _ = io.WriteString(w, "missing")
					return
				}
				if scenario == "cache-hit" {
					w.Header().Set("X-Cache", "HIT from veidemann_cache")
				}
				_, _ = io.WriteString(w, blockedRules)
			}))
			req := allowedRequest(uri+"/blocked", "collection", "execution", "job")
			if scenario == "unscoped" {
				req.CollectionId = ""
			}
			allowed, err := e.IsAllowed(t.Context(), req)
			if scenario == "truncated" || scenario == "later-dns-failure" {
				if err == nil {
					t.Fatal("fetch should fail")
				}
			} else if err != nil || allowed != (scenario == "http-error") {
				t.Fatalf("decision %v %v", allowed, err)
			}
			want := 0
			switch scenario {
			case "repeated-url":
				want = 3
			case "unread-redirect", "later-dns-failure", "http-error":
				want = 1
			}
			if len(s.pairs) != want || len(s.logs) != want {
				t.Fatalf("recorded %d/%d want %d", len(s.pairs), len(s.logs), want)
			}
			if scenario == "repeated-url" {
				for _, p := range s.pairs {
					if p.meta.TargetUri != uri+"/robots.txt" {
						t.Fatal(p.meta)
					}
				}
			}
			if scenario == "http-error" && s.logs[0].StatusCode != 404 {
				t.Fatal(s.logs[0])
			}
		})
	}
}
func TestRecordingDNSFailure(t *testing.T) {
	for _, negative := range []bool{true, false} {
		t.Run(fmt.Sprint(negative), func(t *testing.T) {
			s := &recordingServices{negative: negative, dnsError: !negative}
			e, uri := recordingFixture(t, s, false, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("DNS failure reached origin") }))
			allowed, err := e.IsAllowed(t.Context(), allowedRequest(uri+"/blocked", "c", "e", "j"))
			if err == nil || allowed {
				t.Fatal(allowed, err)
			}
			if len(s.pairs) != 0 {
				t.Fatal("DNS failure recorded as HTTP")
			}
		})
	}
}
func TestRecordingWaitsForStorageWhileCacheRemainsUsable(t *testing.T) {
	for _, phase := range []string{"writer", "log"} {
		t.Run(phase, func(t *testing.T) {
			started := make(chan struct{}, 1)
			release := make(chan struct{})
			s := &recordingServices{}
			if phase == "writer" {
				s.writerStarted, s.releaseWriter = started, release
			} else {
				s.logStarted, s.releaseLog = started, release
			}
			e, uri := recordingFixture(t, s, false, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, blockedRules)
			}))
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			req := allowedRequest(uri+"/blocked", "collection", "execution", "job")
			done := make(chan struct{})
			go func() {
				defer close(done)
				allowed, err := e.IsAllowed(ctx, req)
				if err != nil || allowed {
					t.Error(allowed, err)
				}
			}()
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("recording did not reach", phase)
			}
			select {
			case <-done:
				t.Fatal("refresh returned before recording finished")
			case <-time.After(20 * time.Millisecond):
			}

			// The refresh already cached the rules. A cache hit needs no recording.
			allowed, err := e.IsAllowed(ctx, req)
			if err != nil || allowed {
				t.Fatal("cached decision", allowed, err)
			}
			close(release)
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("refresh did not return after recording finished")
			}
			if len(s.pairs) != 1 || len(s.logs) != 1 || len(s.dnsRequests) != 1 {
				t.Fatalf("pairs/logs/DNS %d/%d/%d", len(s.pairs), len(s.logs), len(s.dnsRequests))
			}
		})
	}
}

func TestRecordingUsesCallerCancellation(t *testing.T) {
	for _, phase := range []string{"writer", "log"} {
		t.Run(phase, func(t *testing.T) {
			started := make(chan struct{}, 1)
			s := &recordingServices{}
			if phase == "writer" {
				s.writerStarted = started
			} else {
				s.logStarted = started
			}
			e, uri := recordingFixture(t, s, false, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, blockedRules)
			}))
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			req := allowedRequest(uri+"/blocked", "collection", "execution", "job")
			done := make(chan struct{})
			go func() {
				defer close(done)
				allowed, err := e.IsAllowed(ctx, req)
				if err != nil || allowed {
					t.Error("recording cancellation changed the decision", allowed, err)
				}
			}()
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("recording did not reach", phase)
			}
			cancel()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("recording ignored caller cancellation")
			}
			allowed, err := e.IsAllowed(t.Context(), req)
			if err != nil || allowed {
				t.Fatal("recording cancellation invalidated cached rules", allowed, err)
			}
		})
	}
}

func TestRecordingConcurrentCollectionRefreshes(t *testing.T) {
	s := &recordingServices{}
	arrived := make(chan struct{}, 6)
	release := make(chan struct{})
	e, uri := recordingFixture(t, s, false, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		arrived <- struct{}{}
		<-release
		_, _ = io.WriteString(w, blockedRules)
	}))
	var wg sync.WaitGroup
	for _, c := range []string{"a", "b", "c", "d", "e", "f"} {
		wg.Go(func() {
			allowed, err := e.IsAllowed(t.Context(), allowedRequest(uri+"/blocked", c, c+"-execution", "job"))
			if err != nil || allowed {
				t.Error(allowed, err)
			}
		})
	}
	for i := 0; i < 6; i++ {
		<-arrived
	}
	close(release)
	wg.Wait()
	seen := map[string]bool{}
	for _, p := range s.pairs {
		if p.meta.ExecutionId != p.meta.CollectionRef.Id+"-execution" {
			t.Fatal(p.meta)
		}
		seen[p.meta.CollectionRef.Id] = true
	}
	if len(seen) != 6 {
		t.Fatal(seen)
	}
}

func TestExplicitSquidTransport(t *testing.T) {
	for _, secure := range []bool{false, true} {
		t.Run(fmt.Sprint(secure), func(t *testing.T) {
			s := &recordingServices{}
			e, uri := recordingFixture(t, s, secure, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Header.Get(executionID) != "" || req.Header.Get(jobExecutionID) != "" || req.Header.Get(collectionID) != "" {
					t.Error("internal cache header reached origin")
				}
				if req.Header.Get("Accept-Encoding") != "identity" {
					t.Error("identity encoding lost")
				}
				_, _ = io.WriteString(w, blockedRules)
			}))
			old := e.client.Transport.(*Transport)
			// A small Squid protocol fixture strips internal headers and, for HTTPS,
			// terminates CONNECT TLS like the deployment's ssl-bump configuration.
			certificate, key, err := testCertificate()
			if err != nil {
				t.Fatal(err)
			}
			pair, err := tls.X509KeyPair(certificate, key)
			if err != nil {
				t.Fatal(err)
			}
			seen := make(chan http.Header, 1)
			forward := func(req *http.Request) (*http.Response, error) {
				seen <- req.Header.Clone()
				req.Header.Del(executionID)
				req.Header.Del(jobExecutionID)
				req.Header.Del(collectionID)
				if !req.URL.IsAbs() {
					target, _ := url.Parse(uri)
					req.URL.Scheme = target.Scheme
					req.URL.Host = target.Host
				}
				req.RequestURI = ""
				return old.Base.RoundTrip(req)
			}
			proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodConnect {
					conn, rw, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
						return
					}
					defer func() { _ = conn.Close() }()
					if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
						t.Error(err)
						return
					}
					if _, err := rw.WriteString("HTTP/1.1 200 Connection established\r\n\r\n"); err != nil {
						t.Error(err)
						return
					}
					if err := rw.Flush(); err != nil {
						t.Error(err)
						return
					}
					secureConn := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{pair}})
					if err := secureConn.Handshake(); err != nil {
						t.Error(err)
						return
					}
					inner, err := http.ReadRequest(bufio.NewReader(secureConn))
					if err != nil {
						t.Error(err)
						return
					}
					resp, err := forward(inner)
					if err != nil {
						t.Error(err)
						return
					}
					defer func() { _ = resp.Body.Close() }()
					_ = resp.Write(secureConn)
					return
				}
				resp, err := forward(req)
				if err != nil {
					t.Error(err)
					return
				}
				defer func() { _ = resp.Body.Close() }()
				for key, values := range resp.Header {
					w.Header()[key] = values
				}
				w.WriteHeader(resp.StatusCode)
				if _, err := io.Copy(w, resp.Body); err != nil {
					t.Error(err)
				}
			}))
			defer proxy.Close()
			t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
			t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
			t.Setenv("NO_PROXY", "*")
			configured, err := NewTransport(old.DNS, strings.TrimPrefix(proxy.URL, "http://"))
			if err != nil {
				t.Fatal(err)
			}
			defer configured.Base.(*http.Transport).CloseIdleConnections()
			e.client.Transport = configured
			allowed, err := e.IsAllowed(t.Context(), allowedRequest(uri+"/blocked", "collection", "execution", "job"))
			if err != nil || allowed {
				t.Fatal(allowed, err)
			}
			got := <-seen
			if got.Get(executionID) != "execution" || got.Get(jobExecutionID) != "job" || got.Get(collectionID) != "" {
				t.Fatal(got)
			}
			if len(s.pairs) != 1 || bytes.Contains(s.pairs[0].headers[0], []byte("veidemann_")) {
				t.Fatal("cache headers archived")
			}
		})
	}
}

// Generate an untrusted leaf through httptest's fixed test identity.
func testCertificate() ([]byte, []byte, error) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	cert := server.TLS.Certificates[0]
	key, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		return nil, nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}), nil
}

func TestSharedRefreshUsesFetchingCallerMetadata(t *testing.T) {
	s := &recordingServices{}
	arrived := make(chan struct{})
	release := make(chan struct{})
	e, uri := recordingFixture(t, s, false, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(arrived)
		<-release
		_, _ = io.WriteString(w, blockedRules)
	}))
	firstDone := make(chan struct{})
	secondDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		allowed, err := e.IsAllowed(t.Context(), allowedRequest(uri+"/blocked", "collection", "first", "job"))
		if err != nil || allowed {
			t.Error(allowed, err)
		}
	}()
	<-arrived
	go func() {
		defer close(secondDone)
		allowed, err := e.IsAllowed(t.Context(), allowedRequest(uri+"/blocked", "collection", "second", "job"))
		if err != nil || allowed {
			t.Error(allowed, err)
		}
	}()
	close(release)
	<-firstDone
	<-secondDone
	if len(s.pairs) != 1 || s.pairs[0].meta.ExecutionId != "first" || len(s.dnsRequests) != 1 {
		t.Fatal("shared refresh lost attribution", s.pairs)
	}
}

func TestArchivePersistsConfirmedResultsAfterOtherFailures(t *testing.T) {
	for _, phase := range []string{"writer", "log"} {
		t.Run(phase, func(t *testing.T) {
			s := &recordingServices{failPhase: phase}
			e, uri := recordingFixture(t, s, false, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				switch req.URL.Path {
				case "/robots.txt":
					w.Header().Set("Location", "/fail")
					w.WriteHeader(302)
				case "/fail":
					w.Header().Set("Location", "/final")
					w.WriteHeader(302)
				default:
					_, _ = io.WriteString(w, blockedRules)
				}
			}))
			allowed, err := e.IsAllowed(t.Context(), allowedRequest(uri+"/blocked", "collection", "execution", "job"))
			if err != nil || allowed {
				t.Fatal(allowed, err)
			}
			if len(s.logs) != 2 || s.logs[0].RequestedUri != uri+"/robots.txt" || s.logs[1].RequestedUri != uri+"/final" {
				t.Fatal("confirmed results discarded", s.logs)
			}

		})
	}
}

// Observe each client-side deadline before gRPC converts it to a relative timeout.
type recordingDeadlineWriter struct {
	writerV1.ContentWriterClient
	deadlines []time.Time
}

func (w *recordingDeadlineWriter) Write(ctx context.Context, opts ...grpc.CallOption) (writerV1.ContentWriter_WriteClient, error) {
	deadline, _ := ctx.Deadline()
	w.deadlines = append(w.deadlines, deadline)
	return w.ContentWriterClient.Write(ctx, opts...)
}

func TestRedirectRecordingSharesDeadlineAndPreservesDecision(t *testing.T) {
	for _, phase := range []string{"writer", "log"} {
		t.Run(phase, func(t *testing.T) {
			services := &recordingServices{}
			started := make(chan struct{}, 1)
			if phase == "writer" {
				services.writerStarted = started
			} else {
				services.logStarted = started
			}
			e, uri := recordingFixture(t, services, false, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/robots.txt":
					http.Redirect(w, r, "/second", http.StatusFound)
				case "/second":
					http.Redirect(w, r, "/rules", http.StatusFound)
				default:
					_, _ = io.WriteString(w, blockedRules)
				}
			}))
			writer := &recordingDeadlineWriter{ContentWriterClient: e.Recorder.writer}
			e.Recorder.writer = writer
			e.Recorder.timeout = 150 * time.Millisecond
			// Allow both redirects through, then stall persistence of the rules.
			release := make(chan struct{})
			if phase == "writer" {
				services.releaseWriter = release
			} else {
				services.releaseLog = release
			}
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			go func() {
				for range 2 {
					select {
					case <-started:
					case <-ctx.Done():
						return
					}
					select {
					case release <- struct{}{}:
					case <-ctx.Done():
						return
					}
				}
			}()
			start := time.Now()
			allowed, err := e.IsAllowed(ctx, allowedRequest(uri+"/blocked", "collection", "execution", "job"))
			elapsed := time.Since(start)
			if allowed || err != nil || elapsed < e.Recorder.timeout || elapsed > time.Second {
				t.Fatalf("decision=%v err=%v elapsed=%v", allowed, err, elapsed)
			}
			if len(writer.deadlines) != 3 || writer.deadlines[0].IsZero() || writer.deadlines[0] != writer.deadlines[1] || writer.deadlines[1] != writer.deadlines[2] {
				t.Fatalf("recording deadlines %v", writer.deadlines)
			}
		})
	}
}
