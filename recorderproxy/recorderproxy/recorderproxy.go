/*
 * Copyright 2019 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package recorderproxy

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	proxy "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/proxycompat"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/serviceconnections"
	"github.com/getlantern/mitm"
	"github.com/getlantern/proxy/v3/filters"

	"net"
	"net/http"
	"time"
)

const (
	CRLF = "\r\n"
)

var acceptAllCerts = &tls.Config{InsecureSkipVerify: true}

type RecorderProxy struct {
	proxy.Proxy
	id                int32
	conn              *serviceconnections.Connections
	ConnectionTimeout time.Duration
	nextProxy         string
	mu                sync.Mutex
	ln                net.Listener
	shuttingDown      bool
	once              sync.Once
}

func NewRecorderProxy(id int, conn *serviceconnections.Connections, nextProxyAddr string) *RecorderProxy {
	r := &RecorderProxy{
		id:        int32(id),
		conn:      conn,
		nextProxy: nextProxyAddr,
	}

	filterChain := filters.Join(
		&NonproxyFilter{},
		&ContextInitFilter{
			conn:    conn,
			proxyId: int32(id),
		},
		&DnsLookupFilter{
			DnsResolverClient: conn.DnsResolverClient(),
		},
		&RecorderFilter{
			proxyId:           int32(id),
			DnsResolverClient: conn.DnsResolverClient(),
			hasNextProxy:      nextProxyAddr != "",
		},
		&ErrorHandlerFilter{
			hasNextProxy: nextProxyAddr != "",
		},
	)

	if nextProxyAddr != "" {
		filterChain = filterChain.Append(&ChainedProxyFilter{proxy: r})
	}

	proxyOpts := &proxy.Opts{
		Dial:   r.Dial,
		Filter: filterChain,
		ShouldMITM: func(req *http.Request, upstreamAddr string) bool {
			return true
		},
		MITMOpts: &mitm.Opts{
			Domains:         []string{"*"},
			ClientTLSConfig: acceptAllCerts,
			ServerTLSConfig: acceptAllCerts,
			Organization:    "Veidemann Recorder Proxy",
			CertFile:        "/tmp/rpcert.pem",
		},
		OnError: func(cs *filters.ConnectionState, req *http.Request, read bool, err error) *http.Response {
			logger.LogWithComponent("PROXY").WithError(err).Error("Probably bug. Error handled by OnError should have been handled elsewhere.")
			res, _, _ := filters.Fail(cs, req, 500, err)
			return res
		},
		OKWaitsForUpstream:  false,
		OKSendsServerTiming: false,
	}

	r.Proxy = proxy.New(proxyOpts)

	return r
}

func (proxy *RecorderProxy) Listen(host string, port int) (net.Listener, error) {
	portStr := strconv.Itoa(port + int(proxy.id))
	addr := net.JoinHostPort(host, portStr)
	return net.Listen("tcp", addr)
}

func (proxy *RecorderProxy) Serve(ln net.Listener) error {
	proxy.mu.Lock()
	if proxy.shuttingDown {
		proxy.mu.Unlock()
		_ = ln.Close()
		return net.ErrClosed
	}

	proxy.ln = ln
	proxy.mu.Unlock()

	for {
		co, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || proxy.isShuttingDown() {
				return nil
			}
			return fmt.Errorf("failed to accept connection: %w", err)
		}

		conn := WrapConn(co, "down", false)
		c, cancel := context.WithCancel(rpcontext.RecordProxyDataAware(context.Background()))

		conn.BaseContext = c
		conn.CancelFunc = cancel

		go func() {
			defer cancel()

			err := proxy.Handle(c, conn, conn)
			if err != nil {
				logger.LogWithComponent("PROXY").WithError(err).Error("Error handling request")
			}
		}()
	}
}

func (proxy *RecorderProxy) Shutdown(ctx context.Context) error {
	proxy.mu.Lock()

	proxy.shuttingDown = true
	ln := proxy.ln
	proxy.ln = nil

	proxy.mu.Unlock()

	if ln != nil {
		if err := ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			return err
		}
	}

	return proxy.waitOpenSessions(ctx)
}

func (proxy *RecorderProxy) isShuttingDown() bool {
	proxy.mu.Lock()
	defer proxy.mu.Unlock()
	return proxy.shuttingDown
}

func (proxy *RecorderProxy) waitOpenSessions(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var prev int64 = -1

	for {
		openSessions := rpcontext.OpenSessions()
		if openSessions == 0 {
			return nil
		}
		if openSessions != prev {
			logger.LogWithComponent("PROXY").Infof("Waiting for %d sessions to complete", openSessions)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

type wrappedConnection struct {
	net.Conn
	t           string
	closed      *int32
	BaseContext context.Context
	CancelFunc  func()
	dirOut      bool
}

func (conn *wrappedConnection) ProxyContext() context.Context {
	return conn.BaseContext
}

func (conn *wrappedConnection) Wrapped() net.Conn {
	return conn.Conn
}

func (conn *wrappedConnection) Close() (err error) {
	l := logger.LogWithComponent("CONN:" + conn.t)
	if atomic.CompareAndSwapInt32(conn.closed, 0, 1) {
		if conn.dirOut {
			l.Debugf("Close connection %v -> %v\n", conn.LocalAddr(), conn.RemoteAddr())
		} else {
			l.Debugf("Close connection %v -> %v\n", conn.RemoteAddr(), conn.LocalAddr())
		}
		if conn.CancelFunc != nil {
			conn.CancelFunc()
		}
	}
	return conn.Conn.Close()
}

func (conn *wrappedConnection) Read(b []byte) (n int, err error) {
	n, err = conn.Conn.Read(b)
	l := logger.LogWithComponent("CONN:" + conn.t)
	if err != nil {
		l = l.WithError(err)
	}
	if logger.IsLevelEnabled(logger.TraceLevel) {
		l.Tracef("read:\n%s\n", logger.FormatPayload(b, n, 10, 20))
	} else {
		l.Debugf("read: %s", logger.FormatPayload(b, n, 10, 20))
	}
	return
}

func (conn *wrappedConnection) Write(b []byte) (n int, err error) {
	n, err = conn.Conn.Write(b)
	l := logger.LogWithComponent("CONN:" + conn.t)
	if err != nil {
		l = l.WithError(err)
	}
	if logger.IsLevelEnabled(logger.TraceLevel) {
		l.Tracef("write:\n%s\n", logger.FormatPayload(b, n, 10, 20))
	} else {
		l.Debugf("write: %s", logger.FormatPayload(b, n, 10, 20))
	}
	return
}

func WrapConn(conn net.Conn, label string, dirOut bool) *wrappedConnection {
	l := logger.LogWithComponent("CONN:" + label)
	if dirOut {
		l.Debugf("New connection %v -> %v\n", conn.LocalAddr(), conn.RemoteAddr())
	} else {
		l.Debugf("New connection %v -> %v\n", conn.RemoteAddr(), conn.LocalAddr())
	}
	i := int32(0)
	return &wrappedConnection{Conn: conn, t: label, dirOut: dirOut, closed: &i}
}
