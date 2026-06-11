package proxycompat

import (
	"net"
	"regexp"
	"strings"
	"sync"
)

const defaultBufferSize = 2 << 11

type BufferSource interface {
	Get() *[]byte
	Put(buf *[]byte)
}

type defaultBufferSource struct{ sync.Pool }

func (dbs *defaultBufferSource) Get() *[]byte {
	return dbs.Pool.Get().(*[]byte)
}

func (dbs *defaultBufferSource) Put(buf *[]byte) {
	dbs.Pool.Put(buf)
}

func newBufferSource() BufferSource {
	return &defaultBufferSource{
		Pool: sync.Pool{
			New: func() any {
				b := make([]byte, defaultBufferSize)
				return &b
			},
		},
	}
}

type noCloseConn struct {
	net.Conn
}

func (conn *noCloseConn) Close() error {
	return nil
}

func (conn *noCloseConn) Wrapped() net.Conn {
	return conn.Conn
}

type wrappedConn struct {
	net.Conn
	wrapped net.Conn
}

func (conn *wrappedConn) Wrapped() net.Conn {
	return conn.wrapped
}

func wrapConn(conn net.Conn, wrapped net.Conn) net.Conn {
	if conn == nil || wrapped == nil {
		return conn
	}
	return &wrappedConn{Conn: conn, wrapped: wrapped}
}

func domainToRegex(domain string) (*regexp.Regexp, error) {
	parts := strings.Split(domain, ".")
	for i, part := range parts {
		if part == "*" {
			parts[i] = `[^.]+`
		}
	}
	return regexp.Compile("^" + strings.Join(parts, `\.`))
}
