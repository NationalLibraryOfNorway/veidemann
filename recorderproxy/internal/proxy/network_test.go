package proxy

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestConnectPrefetchReplaysBufferedInput(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })
	t.Cleanup(func() { _ = server.Close() })

	ctx, cancel := context.WithCancelCause(context.Background())
	prefetch := startConnectPrefetch(server, maxTLSReplaySize, cancel)
	written := make(chan error, 1)
	go func() {
		_, err := client.Write([]byte("client hello"))
		written <- err
	}()
	if err := <-written; err != nil {
		t.Fatal(err)
	}

	replayed, err := prefetch.stop()
	if err != nil {
		t.Fatalf("stop() error = %v", err)
	}
	got := make([]byte, len("client hello"))
	if _, err := io.ReadFull(replayed, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != "client hello" {
		t.Fatalf("replayed input = %q", got)
	}
	if cause := context.Cause(ctx); cause != nil {
		t.Fatalf("prefetch cancellation cause = %v", cause)
	}
}

func TestConnectPrefetchPeerCloseCancelsSetup(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { _ = server.Close() })

	ctx, cancel := context.WithCancelCause(context.Background())
	prefetch := startConnectPrefetch(server, maxTLSReplaySize, cancel)
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("prefetch did not cancel setup")
	}
	_, err := prefetch.stop()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("stop() error = %v, want EOF", err)
	}
}

func TestConnectPrefetchRejectsReplayOverflow(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })
	t.Cleanup(func() { _ = server.Close() })

	ctx, cancel := context.WithCancelCause(context.Background())
	prefetch := startConnectPrefetch(server, 4, cancel)
	written := make(chan error, 1)
	go func() {
		_, err := client.Write([]byte("12345"))
		written <- err
	}()
	if err := <-written; err != nil {
		t.Fatal(err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("overflow did not cancel setup")
	}
	_, err := prefetch.stop()
	if !errors.Is(err, errReplayOverflow) {
		t.Fatalf("stop() error = %v, want replay overflow", err)
	}
}

func TestBidirectionalCopyDrainsResponseAfterClientHalfClose(t *testing.T) {
	downstreamClient, downstreamProxy := tcpPair(t)
	upstreamProxy, upstreamServer := tcpPair(t)

	deadline := time.Now().Add(3 * time.Second)
	if err := downstreamClient.SetDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	if err := upstreamServer.SetDeadline(deadline); err != nil {
		t.Fatal(err)
	}

	copyDone := make(chan struct {
		writeErr error
		readErr  error
	}, 1)
	go func() {
		writeErr, readErr := bidirectionalCopy(upstreamProxy, downstreamProxy)
		copyDone <- struct {
			writeErr error
			readErr  error
		}{writeErr: writeErr, readErr: readErr}
	}()

	request := []byte("request")
	if _, err := downstreamClient.Write(request); err != nil {
		t.Fatal(err)
	}
	if err := downstreamClient.CloseWrite(); err != nil {
		t.Fatal(err)
	}

	gotRequest, err := io.ReadAll(upstreamServer)
	if err != nil {
		t.Fatalf("read request: %v", err)
	}
	if string(gotRequest) != string(request) {
		t.Fatalf("request = %q, want %q", gotRequest, request)
	}

	response := []byte("remaining response")
	if _, err := upstreamServer.Write(response); err != nil {
		t.Fatal(err)
	}
	if err := upstreamServer.CloseWrite(); err != nil {
		t.Fatal(err)
	}

	gotResponse := make([]byte, len(response))
	_, err = io.ReadFull(downstreamClient, gotResponse)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if string(gotResponse) != string(response) {
		t.Fatalf("response = %q, want %q", gotResponse, response)
	}

	result := <-copyDone
	if result.writeErr != nil || result.readErr != nil {
		t.Fatalf("bidirectionalCopy() errors = (%v, %v), want nil", result.writeErr, result.readErr)
	}
}

func tcpPair(t *testing.T) (*net.TCPConn, *net.TCPConn) {
	t.Helper()

	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	accepted := make(chan *net.TCPConn, 1)
	acceptErr := make(chan error, 1)
	go func() {
		conn, err := listener.AcceptTCP()
		if err != nil {
			acceptErr <- err
			return
		}
		accepted <- conn
	}()

	client, err := net.DialTCP("tcp4", nil, listener.Addr().(*net.TCPAddr))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })

	select {
	case server := <-accepted:
		t.Cleanup(func() { _ = server.Close() })
		return client, server
	case err := <-acceptErr:
		t.Fatal(err)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out accepting TCP connection")
	}
	return nil, nil
}
