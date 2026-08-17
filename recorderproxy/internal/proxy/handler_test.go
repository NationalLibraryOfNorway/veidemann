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
