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
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	proxy "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/internal/proxy"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/mitmcert"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/serviceconnections"

	"net"
	"net/http"
	"time"
)

const (
	CRLF = "\r\n"
)

const defaultFinalizationTimeout = 30 * time.Second

type recorderProxyOptions struct {
	mitmIdentity        *mitmcert.Identity
	finalizationTimeout time.Duration
}

// Option configures a RecorderProxy.
type Option func(*recorderProxyOptions)

func WithMITMIdentity(identity *mitmcert.Identity) Option {
	return func(opts *recorderProxyOptions) { opts.mitmIdentity = identity }
}

func WithFinalizationTimeout(timeout time.Duration) Option {
	return func(opts *recorderProxyOptions) { opts.finalizationTimeout = timeout }
}

var (
	defaultIdentityOnce sync.Once
	defaultIdentity     *mitmcert.Identity
	defaultIdentityErr  error
)

func inMemoryMITMIdentity() (*mitmcert.Identity, error) {
	defaultIdentityOnce.Do(func() {
		certPEM, keyPEM, err := mitmcert.Generate(time.Now())
		if err != nil {
			defaultIdentityErr = err
			return
		}
		defaultIdentity, defaultIdentityErr = mitmcert.ParseIdentity(certPEM, keyPEM)
	})
	return defaultIdentity, defaultIdentityErr
}

type RecorderProxy struct {
	handler           *proxy.Handler
	id                int32
	conn              *serviceconnections.Connections
	ConnectionTimeout time.Duration
	nextProxy         string
	mu                sync.Mutex
	ln                net.Listener
	shuttingDown      bool
	lifecycle         *lifecycleTracker
}

func NewRecorderProxy(id int, conn *serviceconnections.Connections, nextProxyAddr string, options ...Option) *RecorderProxy {
	opts := recorderProxyOptions{finalizationTimeout: defaultFinalizationTimeout}
	for _, option := range options {
		option(&opts)
	}
	if opts.mitmIdentity == nil {
		var err error
		opts.mitmIdentity, err = inMemoryMITMIdentity()
		if err != nil {
			panic(fmt.Sprintf("failed to create default MITM identity: %v", err))
		}
	}
	if opts.finalizationTimeout <= 0 {
		opts.finalizationTimeout = defaultFinalizationTimeout
	}

	r := &RecorderProxy{
		id:        int32(id),
		conn:      conn,
		nextProxy: nextProxyAddr,
		lifecycle: newLifecycleTracker(),
	}

	filterChain := proxy.Join(
		&nonproxyFilter{},

		// Initializes request and connection metadata.
		&contextInitFilter{
			conn:                conn,
			proxyId:             int32(id),
			finalizationTimeout: opts.finalizationTimeout,
			lifecycle:           r.lifecycle,
		},

		// Must wrap DNS, recorder, and transport filters.
		&errorHandlerFilter{},

		&dnsLookupFilter{
			dnsResolverClient: conn.DnsResolverClient(),
		},

		&recorderFilter{},
	)

	if nextProxyAddr != "" {
		filterChain = filterChain.Append(&chainedProxyFilter{proxy: r})
	}

	handler, err := proxy.New(proxy.Config{
		Dial:     r.Dial,
		Filter:   filterChain,
		Identity: opts.mitmIdentity,
		OnError: func(cs *proxy.State, req *http.Request, phase proxy.ErrorPhase, err error) *http.Response {
			phasedErr := err
			if proxy.Phase(phasedErr) == "" {
				phasedErr = proxy.NewPhaseError(phase, err)
			}
			failure := classifyFailure(phasedErr, FailureScopeConnection)
			logger.LogWithComponent("PROXY").
				WithField("phase", failure.Phase).
				WithField("scope", failure.Scope).
				WithField("code", failure.Code).
				WithError(err).
				Error("Connection failure outside a recordable HTTP request")
			if phase != proxy.PhaseFilter && phase != proxy.PhaseReadRequest {
				return nil
			}
			res, _, _ := proxy.Fail(cs, req, 500, err)
			return res
		},
		WaitForUpstream: false,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to configure proxy engine: %v", err))
	}
	r.handler = handler

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
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || proxy.isShuttingDown() {
				return nil
			}
			return fmt.Errorf("failed to accept connection: %w", err)
		}

		connectionCtx, cancel := context.WithCancel(context.Background())
		ctx := rpcontext.RecordProxyDataAware(connectionCtx)

		wrappedConn := WrapConn(conn, "down", false)
		wrappedConn.baseCtx, wrappedConn.cancelFunc = ctx, cancel
		if !proxy.lifecycle.addConnection(wrappedConn) {
			_ = wrappedConn.Close()
			continue
		}
		activeConnections.Inc()

		go func() {
			defer activeConnections.Dec()
			defer proxy.lifecycle.removeConnection(wrappedConn)
			defer cancel()
			err := proxy.handler.Handle(ctx, wrappedConn, wrappedConn)
			if err != nil && !isExpectedDisconnect(err) {
				logger.LogWithComponent("PROXY").WithError(err).Error("Error handling request")
			}
		}()
	}
}

func (proxy *RecorderProxy) Shutdown(ctx context.Context) error {
	connections := proxy.lifecycle.closeAndSnapshotConnections()
	var shutdownErr error

	proxy.mu.Lock()

	proxy.shuttingDown = true
	ln := proxy.ln
	proxy.ln = nil

	proxy.mu.Unlock()

	if ln != nil {
		if err := ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			shutdownErr = err
		}
	}

	for _, conn := range connections {
		_ = conn.Close()
	}

	return errors.Join(shutdownErr, proxy.lifecycle.wait(ctx))
}

func (proxy *RecorderProxy) isShuttingDown() bool {
	proxy.mu.Lock()
	defer proxy.mu.Unlock()
	return proxy.shuttingDown
}

type wrappedConnection struct {
	net.Conn
	t          string
	closed     atomic.Int32
	baseCtx    context.Context
	cancelFunc func()
	dirOut     bool
}

func (conn *wrappedConnection) ProxyContext() context.Context {
	return conn.baseCtx
}

func (conn *wrappedConnection) Wrapped() net.Conn {
	return conn.Conn
}

func (conn *wrappedConnection) Close() (err error) {
	isClosed := !conn.closed.CompareAndSwap(0, 1)
	if isClosed {
		return nil
	}

	l := logger.LogWithComponent("CONN:" + conn.t)
	if conn.dirOut {
		l.Debugf("Close connection %v -> %v\n", conn.LocalAddr(), conn.RemoteAddr())
	} else {
		l.Debugf("Close connection %v -> %v\n", conn.RemoteAddr(), conn.LocalAddr())
	}

	if conn.cancelFunc != nil {
		conn.cancelFunc()
	}

	return conn.Conn.Close()
}

func (conn *wrappedConnection) Read(b []byte) (n int, err error) {
	n, err = conn.Conn.Read(b)
	if isExpectedDisconnect(err) && conn.cancelFunc != nil {
		conn.cancelFunc()
	}
	l := logger.LogWithComponent("CONN:" + conn.t)
	if err != nil {
		l = l.WithError(err)
	}
	if logger.IsLevelEnabled(logger.TraceLevel) {
		l.Tracef("read:\n%s\n", logger.FormatPayload(b, n, 10, 20))
	} else {
		l.WithField("bytes", n).Debug("Connection read")
	}
	return
}

func (conn *wrappedConnection) Write(b []byte) (n int, err error) {
	n, err = conn.Conn.Write(b)
	if isExpectedDisconnect(err) && conn.cancelFunc != nil {
		conn.cancelFunc()
	}
	l := logger.LogWithComponent("CONN:" + conn.t)
	if err != nil {
		l = l.WithError(err)
	}
	if logger.IsLevelEnabled(logger.TraceLevel) {
		l.Tracef("write:\n%s\n", logger.FormatPayload(b, n, 10, 20))
	} else {
		l.WithField("bytes", n).Debug("Connection write")
	}
	return
}

func isExpectedDisconnect(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) ||
		errors.Is(err, io.ErrClosedPipe) || errors.Is(err, context.Canceled) {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.HasSuffix(text, "eof") ||
		strings.Contains(text, "use of closed network connection") ||
		strings.Contains(text, "broken pipe") ||
		strings.Contains(text, "closed pipe") ||
		strings.Contains(text, "connection reset by peer")
}

func WrapConn(conn net.Conn, label string, dirOut bool) *wrappedConnection {
	l := logger.LogWithComponent("CONN:" + label)
	if dirOut {
		l.Debugf("New connection %v -> %v\n", conn.LocalAddr(), conn.RemoteAddr())
	} else {
		l.Debugf("New connection %v -> %v\n", conn.RemoteAddr(), conn.LocalAddr())
	}
	return &wrappedConnection{Conn: conn, t: label, dirOut: dirOut}
}
