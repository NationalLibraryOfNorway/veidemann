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
	"crypto/tls"
	"net/http"
	"net/http/httptrace"

	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
)

func DecorateRequest(roundTripper http.RoundTripper, req *http.Request) (http.RoundTripper, *http.Request) {
	t := &transport{wrapped: roundTripper}
	t.log = rpcontext.LogWithContext(req.Context(), "http_client")

	trace := &httptrace.ClientTrace{
		DNSStart:          t.DNSStart,
		DNSDone:           t.DNSDone,
		GotConn:           t.GotConn,
		PutIdleConn:       t.PutIdleConn,
		ConnectStart:      t.ConnectStart,
		ConnectDone:       t.ConnectDone,
		TLSHandshakeStart: t.TLSHandshakeStart,
		TLSHandshakeDone:  t.TLSHandshakeDone,
		GetConn:           t.GetConn,
	}

	return t, req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
}

// transport is an http.RoundTripper that keeps track of the in-flight
// request and implements hooks to report HTTP tracing events.
type transport struct {
	wrapped http.RoundTripper
	current *http.Request
	log     *logger.Logger
}

// RoundTrip wraps http.DefaultTransport.RoundTrip to keep track
// of the current request.
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.current = req
	return t.wrapped.RoundTrip(req)
}

func (t *transport) GetConn(hostPort string) {
	t.log.WithField("destination", hostPort).Trace("HTTP connection requested")
}
func (t *transport) GotConn(info httptrace.GotConnInfo) {
	t.log.WithField("reused", info.Reused).Trace("HTTP connection acquired")
}
func (t *transport) DNSStart(info httptrace.DNSStartInfo) {
	t.log.WithField("host", info.Host).Trace("DNS lookup started")
}
func (t *transport) DNSDone(info httptrace.DNSDoneInfo) {
	t.log.WithFields(logger.Fields{"coalesced": info.Coalesced}).WithError(info.Err).Trace("DNS lookup finished")
}
func (t *transport) PutIdleConn(err error) {
	t.log.WithError(err).Trace("HTTP connection released")
}
func (t *transport) ConnectStart(network, addr string) {
	t.log.WithFields(logger.Fields{"network": network, "destination": addr}).Trace("TCP connection started")
}
func (t *transport) ConnectDone(network, addr string, err error) {
	t.log.WithFields(logger.Fields{"network": network, "destination": addr}).WithError(err).Trace("TCP connection finished")
}
func (t *transport) TLSHandshakeStart() {
	t.log.Trace("TLS handshake started")
}
func (t *transport) TLSHandshakeDone(state tls.ConnectionState, err error) {
	t.log.WithFields(logger.Fields{"serverName": state.ServerName, "handshakeComplete": state.HandshakeComplete}).WithError(err).Trace("TLS handshake finished")
}
