package proxy

import (
	"io"
	"net"
	"testing"
	"time"
)

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
