package proxy

import "errors"

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
	var phased *PhaseError
	if errors.As(err, &phased) {
		return phased.Phase
	}
	return ""
}
