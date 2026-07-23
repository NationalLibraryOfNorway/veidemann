package proxycompat

import (
	"net"
	"time"
)

// ErrorPhase identifies the protocol operation that failed. Callers use the
// phase to decide whether an HTTP response is still legal and how to classify
// the failure.
type ErrorPhase string

const (
	PhaseReadRequest          ErrorPhase = "read_request"
	PhaseFilter               ErrorPhase = "filter"
	PhaseConnectDial          ErrorPhase = "connect_dial"
	PhaseUpstreamProxyConnect ErrorPhase = "upstream_proxy_connect"
	PhaseDownstreamTLS        ErrorPhase = "downstream_tls"
	PhaseUpstreamTLS          ErrorPhase = "upstream_tls"
	PhaseInnerHTTPRequest     ErrorPhase = "inner_http_request"
	PhaseHTTPRoundTrip        ErrorPhase = "http_round_trip"
	PhaseResponseWrite        ErrorPhase = "response_write"
	PhaseTunnel               ErrorPhase = "tunnel"
)

// PhaseError preserves the operation that produced an error while retaining
// the original error for errors.Is/errors.As classification.
type PhaseError struct {
	Phase ErrorPhase
	Err   error
}

func (e *PhaseError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *PhaseError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewPhaseError(phase ErrorPhase, err error) error {
	if err == nil {
		return nil
	}
	return &PhaseError{Phase: phase, Err: err}
}

func Phase(err error) ErrorPhase {
	for err != nil {
		if phased, ok := err.(*PhaseError); ok {
			return phased.Phase
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			break
		}
		err = u.Unwrap()
	}
	return ""
}

// unavailableUpstreamConn adapts an upstream failure to the eager getlantern
// MITM API. It is not a simulated successful connection: every I/O operation
// returns the original failure and it owns no network resources.
type unavailableUpstreamConn struct {
	err error
}

func newUnavailableUpstreamConn(err error) net.Conn {
	return &unavailableUpstreamConn{err: err}
}

func (c *unavailableUpstreamConn) Read([]byte) (int, error)         { return 0, c.err }
func (c *unavailableUpstreamConn) Write([]byte) (int, error)        { return 0, c.err }
func (c *unavailableUpstreamConn) Close() error                     { return nil }
func (c *unavailableUpstreamConn) LocalAddr() net.Addr              { return unavailableAddr("local") }
func (c *unavailableUpstreamConn) RemoteAddr() net.Addr             { return unavailableAddr("upstream") }
func (c *unavailableUpstreamConn) SetDeadline(time.Time) error      { return nil }
func (c *unavailableUpstreamConn) SetReadDeadline(time.Time) error  { return nil }
func (c *unavailableUpstreamConn) SetWriteDeadline(time.Time) error { return nil }

type unavailableAddr string

func (a unavailableAddr) Network() string { return "unavailable" }
func (a unavailableAddr) String() string  { return string(a) }
