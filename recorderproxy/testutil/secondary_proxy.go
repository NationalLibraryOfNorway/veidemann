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

package testutil

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"

	proxy "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/proxycompat"
	"github.com/getlantern/mitm"
	"github.com/getlantern/proxy/v3/filters"
)

var acceptAllCerts = &tls.Config{InsecureSkipVerify: true}

func NewSecondaryProxy(t testing.TB, s *HttpServers) (net.Listener, string) {

	var downstreamConn net.Conn

	opts := &proxy.Opts{
		OnError: func(cs *filters.ConnectionState, req *http.Request, phase proxy.ErrorPhase, err error) (resp *http.Response) {
			t.Logf("Second Proxy: OnError: req: %v, phase: %v, err: %v", req, phase, err)

			var eofRegex = regexp.MustCompile("Unable to round-trip .*: EOF")

			switch errStr := err.Error(); {

			case strings.Contains(errStr, "tls: handshake failure"):
				resp, _, _ = filters.Fail(cs, req, http.StatusServiceUnavailable, errors.New("tls: handshake failure"))
				resp.Header.Set("X-Squid-Error", "ERR_CONNECT_FAIL 111")

			case strings.Contains(errStr, "connect: connection refused"):
				resp, _, _ = filters.Fail(cs, req, http.StatusServiceUnavailable, connectionRefusedCause(err))
				resp.Header.Set("X-Squid-Error", "ERR_CONNECT_FAIL 111")

			case strings.Contains(errStr, "first record does not look like a TLS handshake"):
				downstreamConn.Write([]byte("HTTP/"))

			case eofRegex.MatchString(errStr), strings.Contains(errStr, "tls: bad record MAC"):
				resp, _, _ = filters.Fail(cs, req, http.StatusBadGateway, errors.New("ERR_ZERO_SIZE_OBJECT 0"))
				resp.Header.Set("X-Squid-Error", "ERR_ZERO_SIZE_OBJECT 0")

			default:
				resp, _, _ = filters.Fail(cs, req, 555, err)
			}
			return
		},
		Filter: filters.FilterFunc(
			func(cs *filters.ConnectionState, req *http.Request, next filters.Next) (resp *http.Response, nextCS *filters.ConnectionState, err error) {
				resp, nextCS, err = next(cs, req)
				if err != nil && resp != nil && resp.StatusCode == 502 && strings.Contains(err.Error(), "connection refused") {
					resp, nextCS, err = filters.Fail(cs, req, http.StatusServiceUnavailable, err)
					resp.Header.Add("X-Squid-Error", "ERR_CONNECT_FAIL 111")
				}
				return
			}),
		OKWaitsForUpstream: true,
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
		Dial: func(context context.Context, isConnect bool, network, addr string) (conn net.Conn, err error) {
			timeout := 30 * time.Second
			deadline, hasDeadline := context.Deadline()
			if hasDeadline {
				timeout = time.Until(deadline)
			}
			conn, err = net.DialTimeout(network, addr, timeout)
			return conn, err
		},
	}
	p := proxy.New(opts)

	ln, err := net.Listen("tcp4", "localhost:0")
	if err != nil {
		panic(fmt.Sprintf("Secondary proxy: failed to listen on port %v: %v", 8080, err))
	}

	go func() {
		defer ln.Close()

		for {
			var err error
			downstreamConn, err = ln.Accept()
			if err != nil {
				if !errors.Is(err, net.ErrClosed) {
					t.Logf("Secondary proxy: unable to accept: %v", err)
				}
				return
			}
			downstreamConn = &badCertConn{Conn: downstreamConn, s: s, t: t}

			go func() {
				err := p.Handle(context.Background(), downstreamConn, downstreamConn)
				if err != nil {
					t.Logf("Secondary proxy: error handling request: %v", err)
				}
			}()
		}
	}()

	return ln, ln.Addr().String()
}

func connectionRefusedCause(err error) error {
	if errors.Is(err, syscall.ECONNREFUSED) {
		return syscall.ECONNREFUSED
	}

	var ge interface {
		RootCause() error
	}
	if errors.As(err, &ge) {
		rc := ge.RootCause()
		if errors.Is(rc, syscall.ECONNREFUSED) {
			return syscall.ECONNREFUSED
		}
		if rc != nil {
			return rc
		}
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Err != nil {
		return opErr.Err
	}

	return err
}

// badCertConn wraps a connection and uses invalid certificate negotiation if upstream server has bad certificate
type badCertConn struct {
	net.Conn
	s                   *HttpServers
	readCount           int
	shouldReturnBadCert bool
	t                   testing.TB
}

func (conn *badCertConn) Read(b []byte) (n int, err error) {
	conn.readCount++
	n, err = conn.Conn.Read(b)
	if conn.s != nil {
		u, _ := url.Parse(conn.s.SrvHttpsBrokenTLS.URL)
		if strings.Contains(string(b[:n]), u.Host) {
			conn.shouldReturnBadCert = true
		}
		if conn.shouldReturnBadCert && conn.readCount == 2 {
			conn.t.Log("Secondary Proxy: Sending bad certificate")
			conn.Write([]byte("HTTP/"))
		}
	}
	return
}
