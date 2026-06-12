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
	"testing"
	"time"

	proxy "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/proxycompat"
	gerr "github.com/getlantern/errors"
	"github.com/getlantern/mitm"
	"github.com/getlantern/proxy/v3/filters"
)

var acceptAllCerts = &tls.Config{InsecureSkipVerify: true}

func NewSecondaryProxy(t testing.TB, s *HttpServers) (net.Listener, string) {
	t.Helper()

	var ff filters.FilterFunc
	ff = func(cs *filters.ConnectionState, req *http.Request, next filters.Next) (r *http.Response, nextCS *filters.ConnectionState, e error) {
		r, nextCS, e = next(cs, req)
		if e != nil && r != nil && r.StatusCode == 502 && strings.Contains(e.Error(), "connection refused") {
			r, nextCS, e = filters.Fail(cs, req, http.StatusServiceUnavailable, e)
			r.Header.Add("X-Squid-Error", "ERR_CONNECT_FAIL 111")
		}
		return
	}
	var downstreamConn net.Conn

	opts := &proxy.Opts{
		OnError: func(cs *filters.ConnectionState, req *http.Request, read bool, err error) (r *http.Response) {
			var eofRegex = regexp.MustCompile("Unable to round-trip .*: EOF")
			t.Logf("Second Proxy: OnError: req: %v, read: %v, err: %v", req, read, err)
			switch s := err.Error(); {
			case strings.Contains(s, "tls: handshake failure"):
				r, _, _ = filters.Fail(cs, req, 503, errors.New("tls: handshake failure"))
				r.Header.Set("X-Squid-Error", "ERR_CONNECT_FAIL 111")
			case strings.Contains(s, "connect: connection refused"):
				r, _, _ = filters.Fail(cs, req, 503, err.(gerr.Error).RootCause().(*net.OpError).Err)
				r.Header.Set("X-Squid-Error", "ERR_CONNECT_FAIL 111")
			case strings.Contains(s, "first record does not look like a TLS handshake"):
				downstreamConn.Write([]byte("HTTP/"))
			case eofRegex.MatchString(s):
				r, _, _ = filters.Fail(cs, req, 502, errors.New("ERR_ZERO_SIZE_OBJECT 0"))
				r.Header.Set("X-Squid-Error", "ERR_ZERO_SIZE_OBJECT 0")
			default:
				r, _, _ = filters.Fail(cs, req, 555, err)
			}
			return
		},
		Filter:             ff,
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
	p2 := proxy.New(opts)
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(fmt.Sprintf("Secondary proxy: failed to listen on port %v: %v", 8080, err))
	}
	go func() {
		defer l.Close()

		for {
			var err error
			downstreamConn, err = l.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					t.Log("Secondary proxy: listener closed")
					return
				}
				t.Logf("Secondary proxy: unable to accept: %v", err)
				break
			}
			downstreamConn = &badCertConn{Conn: downstreamConn, s: s, t: t}
			go func() {
				err := p2.Handle(context.Background(), downstreamConn, downstreamConn)
				if err != nil {
					t.Logf("Secondary proxy: error handling request: %v", err)
				}
			}()
		}
	}()

	return l, l.Addr().String()
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
		u, _ := url.Parse(conn.s.SrvHttpsBadCert.URL)
		if strings.Contains(string(b[:n]), u.Host) {
			conn.shouldReturnBadCert = true
		}
		if conn.shouldReturnBadCert && conn.readCount == 2 {
			conn.t.Log("Second Proxy: Sending bad certificate")
			conn.Write([]byte("HTTP/"))
		}
	}
	return
}
