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
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	dnsresolverV1 "github.com/NationalLibraryOfNorway/veidemann/api/dnsresolver/v1"
	robotsV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator/v1"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	proxy "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/internal/proxy"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/mitmcert"
)

const (
	CRLF = "\r\n"
)

const (
	defaultFinalizationTimeout = 30 * time.Second
	defaultIdleTimeout         = 2 * time.Minute
)

type recorderProxyOptions struct {
	robots              robotsV1.RobotsEvaluatorClient
	mitmIdentity        *mitmcert.Identity
	finalizationTimeout time.Duration
	idleTimeout         time.Duration
}

// Option configures a RecorderProxy.
type Option func(*recorderProxyOptions)

func WithRobotsEvaluator(client robotsV1.RobotsEvaluatorClient) Option {
	return func(o *recorderProxyOptions) { o.robots = client }
}

func WithMITMIdentity(identity *mitmcert.Identity) Option {
	return func(opts *recorderProxyOptions) { opts.mitmIdentity = identity }
}

func WithFinalizationTimeout(timeout time.Duration) Option {
	return func(opts *recorderProxyOptions) { opts.finalizationTimeout = timeout }
}

// WithIdleTimeout configures how long the proxy waits for downstream request
// headers. Non-positive values disable the timeout.
func WithIdleTimeout(timeout time.Duration) Option {
	return func(opts *recorderProxyOptions) { opts.idleTimeout = timeout }
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
	attempt             *harvestAttempt
	nextToken           string
	finalizationTimeout time.Duration
	handler             *proxy.Handler
	id                  int32
	// ConnectionTimeout is retained for source compatibility.
	//
	// Deprecated: this field has no effect. Use WithIdleTimeout instead.
	ConnectionTimeout time.Duration
	nextProxy         string
	mu                sync.Mutex
	ln                net.Listener
	shuttingDown      bool
	lifecycle         *lifecycleTracker
}

// NewRecorderProxy uses service clients whose connections are owned by the caller.
func NewRecorderProxy(id int, contentWriter contentwriterV1.ContentWriterClient, dnsResolver dnsresolverV1.DnsResolverClient, nextProxyAddr string, options ...Option) *RecorderProxy {
	opts := recorderProxyOptions{
		finalizationTimeout: defaultFinalizationTimeout,
		idleTimeout:         defaultIdleTimeout,
	}
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
		id:                  int32(id),
		finalizationTimeout: opts.finalizationTimeout,
		nextProxy:           nextProxyAddr,
		lifecycle:           newLifecycleTracker(),
	}

	filterChain := proxy.Join(
		&nonproxyFilter{},

		// Initializes request and connection metadata.
		&contextInitFilter{
			contentWriterClient: contentWriter,
			robots:              opts.robots,
			proxyId:             int32(id),
			finalizationTimeout: opts.finalizationTimeout,
			lifecycle:           r.lifecycle,
		},

		// Must wrap DNS, recorder, and transport filters.
		&errorHandlerFilter{},

		&dnsLookupFilter{
			dnsResolverClient: dnsResolver,
		},

		&recorderFilter{},
	)

	if nextProxyAddr != "" {
		filterChain = filterChain.Append(&chainedProxyFilter{proxy: r})
	}

	handler, err := proxy.New(proxy.Config{
		Dial:          r.Dial,
		Filter:        filterChain,
		Identity:      opts.mitmIdentity,
		IdleTimeout:   opts.idleTimeout,
		OnIdleTimeout: idleConnectionTimeouts.Inc,
		OnError: func(cs *proxy.State, req *http.Request, phase proxy.ErrorPhase, err error) *http.Response {
			var completed *completedResourceError
			if phase == proxy.PhaseResponseWrite && errors.As(err, &completed) {
				return nil
			}
			phasedErr := err
			log := logger.Log
			if proxy.Phase(phasedErr) == "" {
				phasedErr = proxy.NewPhaseError(phase, err)
			}
			if cs != nil {
				if ctx := proxyContextFromConn(cs.Downstream()); ctx != nil {
					log = rpcontext.HarvestLogger(ctx)
					if binding, _ := ctx.Value(attemptKey{}).(*attemptConnection); binding != nil && binding.verified.Load() {
						binding.attempt.connectionFailure(phasedErr)
					}
				}
			}
			failure := classifyFailure(phasedErr, FailureScopeConnection)
			log.
				WithField("phase", failure.Phase).
				WithField("scope", failure.Scope).
				WithField("errorCode", failure.Code).
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
	portStr := strconv.Itoa(port + int(proxy.id) - 1)
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
		proxy.mu.Lock()
		a := proxy.attempt
		if a == nil {
			proxy.mu.Unlock()
			cancel()
			_ = conn.Close()
			continue
		}
		connectionCtx = context.WithValue(connectionCtx, attemptKey{}, &attemptConnection{attempt: a})
		connectionCtx = rpcontext.WithHarvestLogger(connectionCtx, harvestLog(a.spec))
		ctx := rpcontext.RecordProxyDataAware(connectionCtx)

		wrappedConn := wrapConn(conn, "browser_connection", harvestLog(a.spec))
		wrappedConn.baseCtx, wrappedConn.cancelFunc = ctx, cancel
		if !a.lifecycle.addConnection(wrappedConn) {
			proxy.mu.Unlock()
			_ = wrappedConn.Close()
			continue
		}
		proxy.mu.Unlock()
		if !proxy.lifecycle.addConnection(wrappedConn) {
			a.lifecycle.removeConnection(wrappedConn)
			_ = wrappedConn.Close()
			continue
		}
		activeConnections.Inc()

		go func() {
			defer activeConnections.Dec()
			defer proxy.lifecycle.removeConnection(wrappedConn)
			if a != nil {
				defer a.lifecycle.removeConnection(wrappedConn)
			}
			defer cancel()
			err := proxy.handler.Handle(ctx, wrappedConn, wrappedConn)
			if err != nil && !isExpectedDisconnect(err) {
				harvestLog(a.spec).WithField("phase", "connection").WithError(err).Error("Connection failed")
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
	closed     atomic.Int32
	baseCtx    context.Context
	cancelFunc func()
	log        *logger.Logger
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

	conn.log.Trace("Connection closed")

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
	return
}

func (conn *wrappedConnection) Write(b []byte) (n int, err error) {
	n, err = conn.Conn.Write(b)
	if isExpectedDisconnect(err) && conn.cancelFunc != nil {
		conn.cancelFunc()
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

func WrapConn(conn net.Conn, label string, _ bool) *wrappedConnection {
	return wrapConn(conn, label, logger.Log)
}

func wrapConn(conn net.Conn, phase string, log *logger.Logger) *wrappedConnection {
	l := log.WithFields(logger.Fields{"phase": phase, "localAddress": conn.LocalAddr().String(), "remoteAddress": conn.RemoteAddr().String()})
	l.Trace("Connection opened")
	return &wrappedConnection{Conn: conn, log: l}
}
