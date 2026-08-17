package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/mitmcert"
)

func TestHandlerUsesOneCertificateForEverySNI(t *testing.T) {
	certPEM, keyPEM, err := mitmcert.Generate(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	identity, err := mitmcert.ParseIdentity(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := New(Config{
		Identity: identity,
		Filter: FilterFunc(func(s *State, req *http.Request, next Next) (*http.Response, *State, error) {
			return next(s, req)
		}),
		Dial: func(context.Context, bool, string, string) (net.Conn, error) {
			return nil, fmt.Errorf("unused")
		},
		OnError: func(*State, *http.Request, ErrorPhase, error) *http.Response { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	var first *tls.Certificate
	for i := range 5000 {
		certificate, err := handler.serverTLSConfig.GetCertificate(&tls.ClientHelloInfo{
			ServerName: fmt.Sprintf("unique-%d.example", i),
		})
		if err != nil {
			t.Fatal(err)
		}
		if first == nil {
			first = certificate
		} else if certificate != first {
			t.Fatal("handler returned a different certificate")
		}
	}
	certificate, err := handler.serverTLSConfig.GetCertificate(&tls.ClientHelloInfo{})
	if err != nil {
		t.Fatalf("empty SNI was rejected: %v", err)
	}
	if certificate != first {
		t.Fatal("empty SNI returned a different certificate")
	}
}

func TestHandlerCancelsStalledConnectWhenBrowserCloses(t *testing.T) {
	certPEM, keyPEM, err := mitmcert.Generate(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	identity, err := mitmcert.ParseIdentity(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}

	dialStarted := make(chan struct{})
	dialCause := make(chan error, 1)
	var onErrorCalls atomic.Int32
	handler, err := New(Config{
		Identity: identity,
		Filter: FilterFunc(func(s *State, req *http.Request, next Next) (*http.Response, *State, error) {
			return next(s, req)
		}),
		Dial: func(ctx context.Context, _ bool, _, _ string) (net.Conn, error) {
			close(dialStarted)
			<-ctx.Done()
			cause := context.Cause(ctx)
			dialCause <- cause
			return nil, cause
		},
		OnError: func(*State, *http.Request, ErrorPhase, error) *http.Response {
			onErrorCalls.Add(1)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	browser, downstream := net.Pipe()
	done := make(chan error, 1)
	go func() { done <- handler.Handle(context.Background(), downstream, downstream) }()

	connectReq, err := http.NewRequest(http.MethodConnect, "http://example.test:443", nil)
	if err != nil {
		t.Fatal(err)
	}
	connectReq.Host = "example.test:443"
	if err := connectReq.Write(browser); err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(browser), connectReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	select {
	case <-dialStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("upstream dial did not start")
	}
	if err := browser.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case cause := <-dialCause:
		if !errors.Is(cause, io.EOF) {
			t.Fatalf("dial cancellation cause = %v, want EOF", cause)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("upstream dial was not canceled")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not exit")
	}
	if got := onErrorCalls.Load(); got != 0 {
		t.Fatalf("OnError calls = %d, want 0", got)
	}
}

func TestHandlerExpectedResponseWriteDisconnectDoesNotCallOnError(t *testing.T) {
	certPEM, keyPEM, err := mitmcert.Generate(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	identity, err := mitmcert.ParseIdentity(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}

	filterEntered := make(chan struct{})
	allowResponse := make(chan struct{})
	var onErrorCalls atomic.Int32
	handler, err := New(Config{
		Identity: identity,
		Filter: FilterFunc(func(s *State, req *http.Request, _ Next) (*http.Response, *State, error) {
			close(filterEntered)
			<-allowResponse
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, s, nil
		}),
		Dial: func(context.Context, bool, string, string) (net.Conn, error) {
			return nil, fmt.Errorf("unused")
		},
		OnError: func(*State, *http.Request, ErrorPhase, error) *http.Response {
			onErrorCalls.Add(1)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	browser, downstream := net.Pipe()
	done := make(chan error, 1)
	go func() { done <- handler.Handle(context.Background(), downstream, downstream) }()
	requestWritten := make(chan error, 1)
	go func() {
		_, err := browser.Write([]byte("GET http://example.test/ HTTP/1.1\r\nHost: example.test\r\n\r\n"))
		requestWritten <- err
	}()
	if err := <-requestWritten; err != nil {
		t.Fatal(err)
	}
	<-filterEntered
	if err := browser.Close(); err != nil {
		t.Fatal(err)
	}
	close(allowResponse)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not exit")
	}
	if got := onErrorCalls.Load(); got != 0 {
		t.Fatalf("OnError calls = %d, want 0", got)
	}
}

func TestPrepareRequestPreservesDownstreamCloseIntent(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.test/original", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Close = true
	req.Header.Set("X-Test", "original")

	forwarded := prepareRequest(req)
	if forwarded == req {
		t.Fatal("prepareRequest returned the downstream request")
	}
	if forwarded.Close {
		t.Fatal("forwarded request retained downstream close intent")
	}
	if !req.Close {
		t.Fatal("prepareRequest cleared downstream close intent")
	}

	forwarded.Header.Set("X-Test", "forwarded")
	forwarded.URL.Path = "/forwarded"
	if got := req.Header.Get("X-Test"); got != "original" {
		t.Fatalf("downstream request header = %q, want original", got)
	}
	if got := req.URL.Path; got != "/original" {
		t.Fatalf("downstream request path = %q, want /original", got)
	}
}

func TestHandlerCompletesConnectionCloseResponse(t *testing.T) {
	tests := []struct {
		name    string
		request string
		mitming bool
	}{
		{
			name:    "plain HTTP",
			request: "GET http://example.test/ HTTP/1.1\r\nHost: example.test\r\nConnection: close\r\n\r\n",
		},
		{
			name:    "tunneled HTTPS request",
			request: "GET / HTTP/1.1\r\nHost: example.test\r\nConnection: close\r\n\r\n",
			mitming: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestHandler(t, Config{
				Filter: FilterFunc(func(s *State, req *http.Request, _ Next) (*http.Response, *State, error) {
					// Connection policy belongs to the loop and must survive filters
					// and upstream preparation mutating their request copy.
					req.Close = false
					return &http.Response{
						StatusCode:    http.StatusOK,
						Body:          io.NopCloser(strings.NewReader("ok")),
						ContentLength: 2,
					}, s, nil
				}),
			})

			browser, downstream := net.Pipe()
			defer browser.Close()
			done := make(chan error, 1)
			go func() {
				if tt.mitming {
					done <- handler.handle(context.Background(), downstream, downstream, nil, nil, true)
					_ = downstream.Close()
					return
				}
				done <- handler.Handle(context.Background(), downstream, downstream)
			}()

			if _, err := browser.Write([]byte(tt.request)); err != nil {
				t.Fatal(err)
			}
			request, err := http.NewRequest(http.MethodGet, "http://example.test/", nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := http.ReadResponse(bufio.NewReader(browser), request)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if got := string(body); got != "ok" {
				t.Fatalf("response body = %q, want ok", got)
			}
			if !resp.Close {
				t.Fatal("downstream response did not request connection closure")
			}

			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("handler error = %v", err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("handler waited for another request after Connection: close")
			}
		})
	}
}

func TestHandlerIdleTimeoutEndsHeaderWaitNormally(t *testing.T) {
	tests := []struct {
		name    string
		mitming bool
	}{
		{name: "initial request"},
		{name: "first tunneled request", mitming: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var idleTimeoutCalls atomic.Int32
			var onErrorCalls atomic.Int32
			handler := newTestHandler(t, Config{
				IdleTimeout:   25 * time.Millisecond,
				OnIdleTimeout: func() { idleTimeoutCalls.Add(1) },
				OnError: func(*State, *http.Request, ErrorPhase, error) *http.Response {
					onErrorCalls.Add(1)
					return nil
				},
			})

			browser, downstream := net.Pipe()
			defer browser.Close()
			done := make(chan error, 1)
			go func() {
				if tt.mitming {
					done <- handler.handle(context.Background(), downstream, downstream, nil, nil, true)
					_ = downstream.Close()
					return
				}
				done <- handler.Handle(context.Background(), downstream, downstream)
			}()

			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("handler error = %v", err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("handler did not stop after idle timeout")
			}
			if got := idleTimeoutCalls.Load(); got != 1 {
				t.Fatalf("idle timeout calls = %d, want 1", got)
			}
			if got := onErrorCalls.Load(); got != 0 {
				t.Fatalf("OnError calls = %d, want 0", got)
			}
		})
	}
}

func TestHandlerIdleTimeoutBetweenKeepAliveRequests(t *testing.T) {
	var idleTimeoutCalls atomic.Int32
	var onErrorCalls atomic.Int32
	handler := newTestHandler(t, Config{
		IdleTimeout:   25 * time.Millisecond,
		OnIdleTimeout: func() { idleTimeoutCalls.Add(1) },
		OnError: func(*State, *http.Request, ErrorPhase, error) *http.Response {
			onErrorCalls.Add(1)
			return nil
		},
	})

	browser, downstream := net.Pipe()
	defer browser.Close()
	done := make(chan error, 1)
	go func() { done <- handler.Handle(context.Background(), downstream, downstream) }()

	if _, err := browser.Write([]byte("GET http://example.test/ HTTP/1.1\r\nHost: example.test\r\n\r\n")); err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodGet, "http://example.test/", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(browser), request)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("handler error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not stop after keep-alive idle timeout")
	}
	if got := idleTimeoutCalls.Load(); got != 1 {
		t.Fatalf("idle timeout calls = %d, want 1", got)
	}
	if got := onErrorCalls.Load(); got != 0 {
		t.Fatalf("OnError calls = %d, want 0", got)
	}
}

func TestHandlerClearsIdleTimeoutBeforeRequestBody(t *testing.T) {
	const idleTimeout = 25 * time.Millisecond
	var idleTimeoutCalls atomic.Int32
	bodyRead := make(chan error, 1)
	handler := newTestHandler(t, Config{
		IdleTimeout:   idleTimeout,
		OnIdleTimeout: func() { idleTimeoutCalls.Add(1) },
		Filter: FilterFunc(func(s *State, req *http.Request, _ Next) (*http.Response, *State, error) {
			_, err := io.ReadAll(req.Body)
			bodyRead <- err
			return &http.Response{StatusCode: http.StatusOK}, s, nil
		}),
	})

	browser, downstream := net.Pipe()
	defer browser.Close()
	done := make(chan error, 1)
	go func() { done <- handler.Handle(context.Background(), downstream, downstream) }()

	headers := "POST http://example.test/ HTTP/1.1\r\nHost: example.test\r\nContent-Length: 4\r\nConnection: close\r\n\r\n"
	if _, err := browser.Write([]byte(headers)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * idleTimeout)
	if _, err := browser.Write([]byte("body")); err != nil {
		t.Fatalf("request body write failed after header timeout elapsed: %v", err)
	}
	if err := <-bodyRead; err != nil {
		t.Fatalf("request body read failed after header timeout elapsed: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, "http://example.test/", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(browser), request)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("handler error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not exit after Connection: close response")
	}
	if got := idleTimeoutCalls.Load(); got != 0 {
		t.Fatalf("idle timeout calls = %d, want 0", got)
	}
}

func TestHandlerNonPositiveIdleTimeoutIsDisabled(t *testing.T) {
	for _, idleTimeout := range []time.Duration{0, -time.Second} {
		t.Run(idleTimeout.String(), func(t *testing.T) {
			var idleTimeoutCalls atomic.Int32
			handler := newTestHandler(t, Config{
				IdleTimeout:   idleTimeout,
				OnIdleTimeout: func() { idleTimeoutCalls.Add(1) },
			})

			browser, downstream := net.Pipe()
			done := make(chan error, 1)
			go func() { done <- handler.Handle(context.Background(), downstream, downstream) }()

			select {
			case err := <-done:
				t.Fatalf("handler stopped with disabled idle timeout: %v", err)
			case <-time.After(75 * time.Millisecond):
			}
			if err := browser.Close(); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("handler error after browser close = %v", err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("handler did not stop after browser close")
			}
			if got := idleTimeoutCalls.Load(); got != 0 {
				t.Fatalf("idle timeout calls = %d, want 0", got)
			}
		})
	}
}

func newTestHandler(t *testing.T, config Config) *Handler {
	t.Helper()
	certPEM, keyPEM, err := mitmcert.Generate(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	identity, err := mitmcert.ParseIdentity(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	config.Identity = identity
	if config.Filter == nil {
		config.Filter = FilterFunc(func(s *State, _ *http.Request, _ Next) (*http.Response, *State, error) {
			return &http.Response{StatusCode: http.StatusOK}, s, nil
		})
	}
	if config.Dial == nil {
		config.Dial = func(context.Context, bool, string, string) (net.Conn, error) {
			return nil, fmt.Errorf("unused")
		}
	}
	if config.OnError == nil {
		config.OnError = func(*State, *http.Request, ErrorPhase, error) *http.Response { return nil }
	}

	handler, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}
