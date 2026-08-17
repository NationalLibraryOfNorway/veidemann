package proxy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
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

type prefetchResult struct {
	data []byte
	err  error
}

// connectPrefetch owns downstream reads while upstream CONNECT setup is in
// progress. It must be stopped and joined before another reader takes over.
type connectPrefetch struct {
	conn     net.Conn
	limit    int
	cancel   context.CancelCauseFunc
	done     chan prefetchResult
	stopping atomic.Bool
}

func startConnectPrefetch(conn net.Conn, limit int, cancel context.CancelCauseFunc) *connectPrefetch {
	p := &connectPrefetch{
		conn:   conn,
		limit:  limit,
		cancel: cancel,
		done:   make(chan prefetchResult, 1),
	}
	go p.read()
	return p
}

func (p *connectPrefetch) read() {
	data := make([]byte, 0, min(p.limit, 32<<10))
	buf := make([]byte, 32<<10)
	for {
		n, err := p.conn.Read(buf)
		if n > 0 {
			if len(data)+n > p.limit {
				err = fmt.Errorf("%w: limit %d bytes", errReplayOverflow, p.limit)
				p.cancel(err)
				p.done <- prefetchResult{data: data, err: err}
				return
			}
			data = append(data, buf[:n]...)
		}
		if err != nil {
			var netErr net.Error
			if p.stopping.Load() && errors.As(err, &netErr) && netErr.Timeout() {
				err = nil
			} else {
				p.cancel(err)
			}
			p.done <- prefetchResult{data: data, err: err}
			return
		}
	}
}

func (p *connectPrefetch) stop() (net.Conn, error) {
	p.stopping.Store(true)
	if err := p.conn.SetReadDeadline(time.Now()); err != nil {
		_ = p.conn.Close()
		result := <-p.done
		return nil, errors.Join(fmt.Errorf("unable to stop CONNECT prefetch: %w", err), result.err)
	}

	result := <-p.done
	if err := p.conn.SetReadDeadline(time.Time{}); err != nil {
		_ = p.conn.Close()
		return nil, errors.Join(fmt.Errorf("unable to clear CONNECT prefetch deadline: %w", err), result.err)
	}

	replayed := &readerConn{
		Conn:   p.conn,
		reader: io.MultiReader(bytesReader(result.data), p.conn),
	}
	return replayed, result.err
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

const tunnelDrainTimeout = time.Second

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
			deadline := time.Now().Add(tunnelDrainTimeout)
			_ = upstream.SetDeadline(deadline)
			_ = downstream.SetDeadline(deadline)
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
