// Copyright 2026 National Library of Norway.
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
	p "gopkg.in/rethinkdb/rethinkdb-go.v6/ql2"
)

// These tests exercise the pinned driver over TCP without an external database.
// The peer uses the driver's supported V0_4 handshake to keep authentication out
// of the fixture; production continues to use the default handshake.
func startRethinkPeer(t *testing.T, handle func(net.Conn, uint64, p.Query_QueryType) error) (string, *atomic.Int32) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var peers []net.Conn
	var workers sync.WaitGroup
	var accepted atomic.Int32
	acceptDone := make(chan struct{})
	t.Cleanup(func() {
		_ = listener.Close()
		<-acceptDone
		mu.Lock()
		for _, peer := range peers {
			_ = peer.Close()
		}
		mu.Unlock()
		workers.Wait()
	})
	go func() {
		defer close(acceptDone)
		for {
			peer, err := listener.Accept()
			if err != nil {
				return
			}
			accepted.Add(1)
			mu.Lock()
			peers = append(peers, peer)
			mu.Unlock()
			workers.Add(1)
			go func() {
				defer workers.Done()
				defer peer.Close()
				// A deadline also bounds fixture failures during connection setup.
				_ = peer.SetDeadline(time.Now().Add(5 * time.Second))
				var header [12]byte
				if _, err := io.ReadFull(peer, header[:]); err != nil {
					return
				}
				if binary.LittleEndian.Uint32(header[4:8]) != 0 {
					t.Error("test peer expects an empty auth key")
					return
				}
				if _, err := io.WriteString(peer, "SUCCESS\x00"); err != nil {
					return
				}
				for {
					if _, err := io.ReadFull(peer, header[:]); err != nil {
						return
					}
					token := binary.LittleEndian.Uint64(header[:8])
					body := make([]byte, binary.LittleEndian.Uint32(header[8:]))
					if _, err := io.ReadFull(peer, body); err != nil {
						return
					}
					var query []json.RawMessage
					if err := json.Unmarshal(body, &query); err != nil || len(query) == 0 {
						t.Errorf("invalid query: %s", body)
						return
					}
					var kind p.Query_QueryType
					if err := json.Unmarshal(query[0], &kind); err != nil {
						t.Error(err)
						return
					}
					var err error
					if kind == p.Query_SERVER_INFO {
						err = writeRethinkResponse(peer, token, `{"t":5,"r":[{"id":"test-node","name":"test-node"}]}`)
					} else {
						err = handle(peer, token, kind)
					}
					if err != nil {
						return
					}
				}
			}()
		}
	}()
	return listener.Addr().String(), &accepted
}

func writeRethinkResponse(peer net.Conn, token uint64, body string) error {
	frame := make([]byte, 12+len(body))
	binary.LittleEndian.PutUint64(frame[:8], token)
	binary.LittleEndian.PutUint32(frame[8:12], uint32(len(body)))
	copy(frame[12:], body)
	_, err := peer.Write(frame)
	return err
}

func newDriverTestConnection(t *testing.T) *RethinkDbConnection {
	t.Helper()
	conn := NewRethinkDbConnection(Options{QueryTimeout: time.Second, MaxOpenConnections: 2, MaxRetries: 5})
	conn.connectOpts.HandshakeVersion = r.HandshakeV0_4
	// Registered before the peer cleanup: close peer sockets first so even a
	// regression leaving Query stuck cannot deadlock Session.Close in cleanup.
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestDriverRecoversDroppedConnectionWithSameSession(t *testing.T) {
	conn := newDriverTestConnection(t)
	var drop atomic.Bool
	drop.Store(true)
	address, _ := startRethinkPeer(t, func(peer net.Conn, token uint64, kind p.Query_QueryType) error {
		if drop.Swap(false) {
			_ = peer.Close()
			return io.EOF
		}
		return writeRethinkResponse(peer, token, `{"t":1,"r":["recovered"]}`)
	})
	conn.connectOpts.Address = address
	if err := conn.Connect(); err != nil {
		t.Fatal(err)
	}
	session := conn.session
	// More reads than pool slots exercise replacement of the dropped connection,
	// rather than only succeeding through another healthy slot.
	for i := 0; i < 6; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		term := r.Expr("probe")
		cursor, err := conn.execRead(ctx, "probe", &term)
		if err == nil {
			var result string
			err = cursor.One(&result)
			if err == nil && result != "recovered" {
				t.Errorf("query result = %q", result)
			}
		}
		cancel()
		// The first interrupted read may fail; subsequent reads must recover.
		if err != nil && i != 0 {
			t.Fatalf("read %d failed after disconnect: %v", i, err)
		}
		if conn.session != session {
			t.Fatal("connection recovery replaced the session")
		}
	}
}

func TestExecWithRetryRetainsSessionAfterClosedConnection(t *testing.T) {
	conn := newDriverTestConnection(t)
	address, accepted := startRethinkPeer(t, func(_ net.Conn, _ uint64, kind p.Query_QueryType) error {
		t.Errorf("unexpected query %v", kind)
		return io.EOF
	})
	conn.connectOpts.Address = address
	if err := conn.Connect(); err != nil {
		t.Fatal(err)
	}
	session := conn.session
	initialConnections := accepted.Load()

	// Inject the error at the retry boundary so the driver's own retries
	// cannot handle it before execWithRetry sees it.
	attempts := 0
	_, err := conn.execWithRetry(context.Background(), "test-query", func(ctx context.Context) (*r.Cursor, error) {
		attempts++
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("query context has no deadline")
		}
		if attempts == 1 {
			return nil, fmt.Errorf("wrapped: %w", r.ErrConnectionClosed)
		}
		return nil, nil
	})
	if err != nil || attempts != 2 {
		t.Fatalf("execWithRetry() = %v after %d attempts, want success after 2", err, attempts)
	}
	if extra := accepted.Load() - initialConnections; extra != 0 {
		t.Errorf("retry opened %d new connections, want 0", extra)
	}
	if conn.session != session {
		t.Error("retry replaced the database session")
	}
}

func TestFailedConnectDoesNotPublishSession(t *testing.T) {
	conn := newDriverTestConnection(t)
	// An invalid port fails before handshake without requiring a real server.
	conn.connectOpts.Address = "127.0.0.1:invalid-port"
	if err := conn.Connect(); err == nil {
		t.Fatal("Connect() succeeded with an invalid port")
	}
	if conn.session != nil {
		t.Fatal("failed Connect() published a session")
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("Close() after failed Connect() = %v", err)
	}
}
