package recorderproxy

import (
	"context"
	"errors"
	"net"
	"strings"

	rperrors "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/proxycompat"
)

type FailureScope string

const (
	FailureScopeConnection FailureScope = "connection"
	FailureScopeResource   FailureScope = "resource"
)

// RecorderFailure is the canonical internal representation of an operational
// failure. Phase describes where it happened; Scope describes which terminal
// actions are legal.
type RecorderFailure struct {
	Code    rperrors.ErrorCode
	Message string
	Detail  string
	Phase   proxycompat.ErrorPhase
	Scope   FailureScope
	Cause   error
}

func (f RecorderFailure) Error() string {
	return f.asError().Error()
}

func (f RecorderFailure) asError() error {
	if f.Cause != nil {
		return rperrors.Wrap(f.Cause, f.Code, f.Message, f.Detail)
	}
	return rperrors.Error(f.Code, f.Message, f.Detail)
}

func classifyFailure(err error, scope FailureScope) RecorderFailure {
	phase := proxycompat.Phase(err)

	var proxyErr *rperrors.ProxyError
	if errors.As(err, &proxyErr) {
		return RecorderFailure{
			Code:    proxyErr.Code(),
			Message: proxyErr.Message(),
			Detail:  cleanFailureText(proxyErr.Detail()),
			Phase:   phase,
			Scope:   scope,
			Cause:   err,
		}
	}

	detail := cleanFailureDetail(err)
	code := rperrors.RuntimeException
	switch {
	case err == nil:
		detail = "unknown error"

	case errors.Is(err, context.DeadlineExceeded), isNetTimeout(err), containsAny(detail,
		"i/o timeout", "timeout awaiting response", "context deadline exceeded"):
		code = rperrors.HttpTimeout

	case containsAny(detail, "no such host", "server misbehaving", "temporary failure in name resolution"):
		code = rperrors.DomainLookupFailed

	case containsAny(detail, "empty reply from server", "server closed idle connection", "unexpected eof") ||
		(detail == "EOF" && phase == proxycompat.PhaseHTTPRoundTrip):
		code = rperrors.EmptyResponse
		detail = "Empty reply from server"

	case containsAny(detail, "connection reset by peer", "broken pipe"):
		code = rperrors.ConnectBroken

	case containsAny(detail, "connection refused", "network is unreachable", "no route to host"):
		code = rperrors.ConnectFailed

	case isConnectPhase(phase):
		code = rperrors.ConnectFailed
	}

	return RecorderFailure{
		Code:    code,
		Message: errorCodeMessage(code),
		Detail:  detail,
		Phase:   phase,
		Scope:   scope,
		Cause:   err,
	}
}

func isConnectPhase(phase proxycompat.ErrorPhase) bool {
	switch phase {
	case proxycompat.PhaseConnectDial,
		proxycompat.PhaseUpstreamProxyConnect,
		proxycompat.PhaseDownstreamTLS,
		proxycompat.PhaseUpstreamTLS,
		proxycompat.PhaseTunnel:
		return true
	default:
		return false
	}
}

func errorCodeMessage(code rperrors.ErrorCode) string {
	switch code {
	case rperrors.ConnectFailed:
		return "CONNECT_FAILED"
	case rperrors.ConnectBroken:
		return "CONNECT_BROKEN"
	case rperrors.HttpTimeout:
		return "HTTP_TIMEOUT"
	case rperrors.DomainLookupFailed:
		return "DOMAIN_LOOKUP_FAILED"
	case rperrors.EmptyResponse:
		return "EMPTY_RESPONSE"
	case rperrors.CanceledByBrowser:
		return "CANCELED_BY_BROWSER"
	case rperrors.PrecludedByRobots:
		return "PRECLUDED_BY_ROBOTS"
	default:
		return "UNKNOWN_ERROR"
	}
}

func cleanFailureDetail(err error) string {
	if err == nil {
		return ""
	}
	return cleanFailureText(err.Error())
}

func cleanFailureText(detail string) string {
	if idx := strings.IndexByte(detail, 0); idx >= 0 {
		detail = detail[:idx]
	}
	prefixes := []string{
		"unable to MITM connection: ",
		"Unable to round-trip http request to upstream: ",
		"unable to round-trip http request to upstream: ",
		"proxyconnect tcp: ",
	}
	for _, prefix := range prefixes {
		detail = strings.TrimPrefix(detail, prefix)
	}

	if idx := strings.Index(detail, ": dial tcp "); idx >= 0 {
		detail = strings.TrimPrefix(detail[idx+2:], "dial tcp ")
	}
	if strings.HasPrefix(detail, "Error round-tripping to ") {
		if idx := strings.Index(detail, ": "); idx >= 0 {
			detail = detail[idx+2:]
		}
	}
	if strings.Contains(detail, "connection refused") {
		return "connection refused"
	}
	if containsAny(
		detail,
		"tls: handshake failure",
		"certificate private key",
		"first record does not look like a tls handshake",
	) {
		return "tls: handshake failure"
	}
	if containsAny(detail, "tls: bad record MAC") {
		return "tls: bad record MAC"
	}
	return strings.TrimSpace(detail)
}

func containsAny(s string, needles ...string) bool {
	s = strings.ToLower(s)
	for _, needle := range needles {
		if strings.Contains(s, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func isNetTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
