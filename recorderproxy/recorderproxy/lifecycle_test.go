package recorderproxy

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
)

func TestLifecycleWaitIncludesRecordRegisteredDuringShutdown(t *testing.T) {
	lifecycle := newLifecycleTracker()
	lifecycle.closeAndSnapshotConnections()

	rc := &rpcontext.RecordContext{}
	lifecycle.addRecord(rc)
	done := make(chan error, 1)
	go func() { done <- lifecycle.wait(context.Background()) }()

	select {
	case err := <-done:
		t.Fatalf("wait returned before record completed: %v", err)
	default:
	}
	rc.CloseFunc()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("wait did not observe completed record")
	}
}

func TestWrappedConnectionPeerCloseCancelsContext(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { _ = server.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	conn := WrapConn(server, "test", false)
	conn.baseCtx = ctx
	conn.cancelFunc = cancel

	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	_, err := conn.Read(make([]byte, 1))
	if err != io.EOF {
		t.Fatalf("Read() error = %v, want EOF", err)
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatal("peer close did not cancel connection context")
	}
}
