package proxy

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type wrappedConnection interface {
	Wrapped() net.Conn
}

func walkWrapped(conn net.Conn, visit func(net.Conn) bool) {
	seen := make(map[net.Conn]struct{})
	for conn != nil {
		if _, ok := seen[conn]; ok {
			return
		}
		seen[conn] = struct{}{}
		if !visit(conn) {
			return
		}
		wrapped, ok := conn.(wrappedConnection)
		if !ok {
			return
		}
		conn = wrapped.Wrapped()
	}
}

type readerConn struct {
	net.Conn
	reader io.Reader
}

func (c *readerConn) Read(p []byte) (int, error) { return c.reader.Read(p) }
func (c *readerConn) Wrapped() net.Conn          { return c.Conn }

func connWithBufferedReader(conn net.Conn, reader *bufio.Reader) net.Conn {
	return &readerConn{Conn: conn, reader: reader}
}

var errReplayOverflow = errors.New("recorded input exceeds replay limit")

type recordingConn struct {
	net.Conn
	limit     int
	recording bool
	recorded  []byte
}

func newRecordingConn(conn net.Conn, limit int) *recordingConn {
	return &recordingConn{Conn: conn, limit: limit, recording: true}
}

func (c *recordingConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if c.recording && n > 0 {
		if len(c.recorded)+n > c.limit {
			return 0, fmt.Errorf("%w: limit %d bytes", errReplayOverflow, c.limit)
		}
		c.recorded = append(c.recorded, p[:n]...)
	}
	return n, err
}

func (c *recordingConn) stop()             { c.recording = false; c.recorded = nil }
func (c *recordingConn) replay() io.Reader { return bytesReader(c.recorded) }
func (c *recordingConn) Wrapped() net.Conn { return c.Conn }

type sliceReader struct{ data []byte }

func bytesReader(data []byte) io.Reader { return &sliceReader{data: append([]byte(nil), data...)} }
func (r *sliceReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

type copyResult struct {
	direction string
	err       error
}

func bidirectionalCopy(upstream, downstream net.Conn) (writeErr, readErr error) {
	results := make(chan copyResult, 2)
	var once sync.Once
	copyOne := func(direction string, dst, src net.Conn) {
		_, err := io.Copy(dst, src)
		results <- copyResult{direction: direction, err: err}
	}
	go copyOne("upstream", upstream, downstream)
	go copyOne("downstream", downstream, upstream)

	for range 2 {
		result := <-results
		if result.direction == "upstream" {
			writeErr = result.err
		} else {
			readErr = result.err
		}
		once.Do(func() {
			if result.direction == "upstream" {
				closeWrite(upstream)
				closeRead(downstream)
			} else {
				closeWrite(downstream)
				closeRead(upstream)
			}
			now := time.Now()
			_ = upstream.SetDeadline(now)
			_ = downstream.SetDeadline(now)
		})
	}
	return writeErr, readErr
}

func closeWrite(conn net.Conn) {
	walkWrapped(conn, func(candidate net.Conn) bool {
		if half, ok := candidate.(interface{ CloseWrite() error }); ok {
			_ = half.CloseWrite()
			return false
		}
		return true
	})
}

func closeRead(conn net.Conn) {
	walkWrapped(conn, func(candidate net.Conn) bool {
		if half, ok := candidate.(interface{ CloseRead() error }); ok {
			_ = half.CloseRead()
			return false
		}
		return true
	})
}
