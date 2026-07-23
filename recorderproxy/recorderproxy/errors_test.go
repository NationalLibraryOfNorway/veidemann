package recorderproxy

import (
	"context"
	stderrors "errors"
	"fmt"
	"net"
	"testing"

	rperrors "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/proxycompat"
)

func TestClassifyFailure(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		code    rperrors.ErrorCode
		message string
		detail  string
		phase   proxycompat.ErrorPhase
	}{
		{
			name:    "wrapped dial refusal",
			err:     proxycompat.NewPhaseError(proxycompat.PhaseConnectDial, fmt.Errorf("dial tcp: %w", stderrors.New("connection refused"))),
			code:    rperrors.ConnectFailed,
			message: "CONNECT_FAILED",
			detail:  "connection refused",
			phase:   proxycompat.PhaseConnectDial,
		},
		{
			name:    "deadline",
			err:     proxycompat.NewPhaseError(proxycompat.PhaseHTTPRoundTrip, context.DeadlineExceeded),
			code:    rperrors.HttpTimeout,
			message: "HTTP_TIMEOUT",
			detail:  "context deadline exceeded",
			phase:   proxycompat.PhaseHTTPRoundTrip,
		},
		{
			name:    "dns",
			err:     proxycompat.NewPhaseError(proxycompat.PhaseConnectDial, &net.DNSError{Err: "no such host", Name: "missing.example"}),
			code:    rperrors.DomainLookupFailed,
			message: "DOMAIN_LOOKUP_FAILED",
			detail:  "lookup missing.example: no such host",
			phase:   proxycompat.PhaseConnectDial,
		},
		{
			name:    "round trip EOF",
			err:     proxycompat.NewPhaseError(proxycompat.PhaseHTTPRoundTrip, stderrors.New("EOF")),
			code:    rperrors.EmptyResponse,
			message: "EMPTY_RESPONSE",
			detail:  "Empty reply from server",
			phase:   proxycompat.PhaseHTTPRoundTrip,
		},
		{
			name:    "upstream TLS",
			err:     proxycompat.NewPhaseError(proxycompat.PhaseUpstreamTLS, stderrors.New("tls: handshake failure")),
			code:    rperrors.ConnectFailed,
			message: "CONNECT_FAILED",
			detail:  "tls: handshake failure",
			phase:   proxycompat.PhaseUpstreamTLS,
		},
		{
			name:    "upstream TLS invalid record",
			err:     proxycompat.NewPhaseError(proxycompat.PhaseUpstreamTLS, stderrors.New("tls: first record does not look like a TLS handshake")),
			code:    rperrors.ConnectFailed,
			message: "CONNECT_FAILED",
			detail:  "tls: handshake failure",
			phase:   proxycompat.PhaseUpstreamTLS,
		},
		{
			name: "typed squid refusal",
			err: proxycompat.NewPhaseError(
				proxycompat.PhaseUpstreamProxyConnect,
				rperrors.Error(rperrors.ConnectFailed, "CONNECT_FAILED", "connect: connection refused"),
			),
			code:    rperrors.ConnectFailed,
			message: "CONNECT_FAILED",
			detail:  "connection refused",
			phase:   proxycompat.PhaseUpstreamProxyConnect,
		},
		{
			name:    "untyped squid header",
			err:     proxycompat.NewPhaseError(proxycompat.PhaseUpstreamProxyConnect, stderrors.New("X-Squid-Error: ERR_CONNECT_FAIL 111")),
			code:    rperrors.ConnectFailed,
			message: "CONNECT_FAILED",
			detail:  "X-Squid-Error: ERR_CONNECT_FAIL 111",
			phase:   proxycompat.PhaseUpstreamProxyConnect,
		},
		{
			name:    "browser cancellation",
			err:     rperrors.Error(rperrors.CanceledByBrowser, "CANCELED_BY_BROWSER", "client disconnected"),
			code:    rperrors.CanceledByBrowser,
			message: "CANCELED_BY_BROWSER",
			detail:  "client disconnected",
		},
		{
			name:    "unknown",
			err:     proxycompat.NewPhaseError(proxycompat.PhaseHTTPRoundTrip, stderrors.New("tls: bad record MAC")),
			code:    rperrors.RuntimeException,
			message: "UNKNOWN_ERROR",
			detail:  "tls: bad record MAC",
			phase:   proxycompat.PhaseHTTPRoundTrip,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyFailure(tc.err, FailureScopeResource)
			if got.Code != tc.code || got.Message != tc.message || got.Detail != tc.detail || got.Phase != tc.phase {
				t.Fatalf("classifyFailure() = %#v", got)
			}
			if got.Scope != FailureScopeResource {
				t.Fatalf("scope = %q, want %q", got.Scope, FailureScopeResource)
			}
			if got.Cause == nil {
				t.Fatal("original cause was not retained")
			}
		})
	}
}
