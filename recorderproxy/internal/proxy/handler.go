package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/mitmcert"
)

const (
	maxTLSReplaySize = 128 << 10
	recordTypeAlert  = 21
)

type DialFunc func(context.Context, bool, string, string) (net.Conn, error)
type ErrorFunc func(*State, *http.Request, ErrorPhase, error) *http.Response

type Config struct {
	Filter          Filter
	Dial            DialFunc
	OnError         ErrorFunc
	Identity        *mitmcert.Identity
	WaitForUpstream bool
}

type Handler struct {
	config          Config
	serverTLSConfig *tls.Config
	clientTLSConfig *tls.Config
}

func New(config Config) (*Handler, error) {
	if config.Filter == nil {
		return nil, errors.New("proxy request filter is required")
	}
	if config.Dial == nil {
		return nil, errors.New("proxy dial function is required")
	}
	if config.OnError == nil {
		return nil, errors.New("proxy error callback is required")
	}
	certificate := config.Identity.TLSCertificate()
	if certificate == nil {
		return nil, errors.New("MITM identity is required")
	}

	h := &Handler{config: config}
	h.serverTLSConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			return certificate, nil
		},
	}
	h.clientTLSConfig = &tls.Config{InsecureSkipVerify: true}
	return h, nil
}

func (h *Handler) Handle(ctx context.Context, input io.Reader, downstream net.Conn) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("recovered from panic handling connection: %v", recovered)
			logger.LogWithComponent("PROXY").WithError(err).Error("Recovered from panic handling connection")
		}
		_ = downstream.Close()
	}()
	return h.handle(ctx, input, downstream, nil, nil, false)
}

func (h *Handler) handle(ctx context.Context, input io.Reader, downstream, upstream net.Conn, upstreamFailure error, mitming bool) error {
	reader := bufio.NewReader(input)
	req, err := http.ReadRequest(reader)
	if err != nil {
		phase := PhaseReadRequest
		if mitming {
			phase = PhaseInnerHTTPRequest
		}
		if isUnexpected(err) {
			state := newState(nil, nil, downstream)
			if resp := h.config.OnError(state, nil, phase, err); resp != nil {
				_ = h.writeResponse(downstream, nil, resp)
			}
			return NewPhaseError(phase, fmt.Errorf("initial ReadRequest error from %v: %w", downstream.RemoteAddr(), err))
		}
		return nil
	}
	req = req.WithContext(ctx)
	if remote := downstream.RemoteAddr(); remote != nil {
		req.RemoteAddr = remote.String()
	}

	stateUpstream := upstream
	if mitming {
		// A tunneled HTTP request uses the established connection only through
		// its transport. State.Upstream is reserved for pending CONNECT setup.
		stateUpstream = nil
	}
	state := newState(req, stateUpstream, downstream)
	state.SetMITMing(mitming)
	var next Next
	if req.Method == http.MethodConnect {
		next = h.nextConnect(ctx)
	} else if upstreamFailure != nil {
		next = func(s *State, req *http.Request) (*http.Response, *State, error) {
			return nil, s, upstreamFailure
		}
	} else {
		transport := h.transport(upstream)
		defer transport.CloseIdleConnections()
		next = h.nextHTTP(transport)
	}

	return h.processRequests(ctx, state, req.RemoteAddr, req, downstream, reader, next)
}

func (h *Handler) transport(upstream net.Conn) *http.Transport {
	if upstream != nil {
		return &http.Transport{
			DialContext: func(context.Context, string, string) (net.Conn, error) {
				return &noCloseConn{Conn: upstream}, nil
			},
			MaxIdleConnsPerHost: -1,
		}
	}
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return h.config.Dial(ctx, false, network, addr)
		},
		MaxIdleConnsPerHost: 1,
	}
}

type noCloseConn struct{ net.Conn }

func (c *noCloseConn) Close() error      { return nil }
func (c *noCloseConn) Wrapped() net.Conn { return c.Conn }

func (h *Handler) nextHTTP(transport *http.Transport) Next {
	return func(state *State, req *http.Request) (*http.Response, *State, error) {
		req = prepareRequest(req)
		resp, err := transport.RoundTrip(req)
		if err != nil {
			return resp, state, NewPhaseError(PhaseHTTPRoundTrip, fmt.Errorf("Unable to round-trip http request to upstream: %w", err))
		}
		return resp, state, nil
	}
}

func (h *Handler) nextConnect(ctx context.Context) Next {
	return func(state *State, req *http.Request) (*http.Response, *State, error) {
		nextState := state.Clone()
		upstreamAddr := connectAddress(req)
		nextState.SetUpstreamAddr(upstreamAddr)
		if h.config.WaitForUpstream {
			upstream, err := h.config.Dial(ctx, true, "tcp", upstreamAddr)
			if err != nil {
				return Fail(state, req, http.StatusBadGateway, err)
			}
			nextState.SetUpstream(upstream)
		}
		return ShortCircuit(nextState, req, &http.Response{StatusCode: http.StatusOK})
	}
}

func connectAddress(req *http.Request) string {
	if req == nil {
		return ""
	}
	if req.URL != nil {
		if req.URL.Host != "" {
			return req.URL.Host
		}
		if req.URL.Opaque != "" {
			return req.URL.Opaque
		}
	}
	return req.Host

}

func (h *Handler) processRequests(ctx context.Context, state *State, remote string, req *http.Request, downstream net.Conn, reader *bufio.Reader, next Next) error {
	for {
		if req.URL.Scheme == "" {
			req.URL.Scheme = state.OriginalURLScheme()
		}
		if req.URL.Host == "" {
			req.URL.Host = state.OriginalURLHost()
		}
		if req.Host == "" {
			req.Host = state.OriginalHost()
		}

		resp, nextState, err := h.config.Filter.Apply(state, req, next)
		if nextState != nil {
			state = nextState
		}
		if err != nil && resp == nil {
			resp = h.config.OnError(state, req, PhaseFilter, err)
			if resp != nil {
				resp.Close = true
			}
		}
		if resp != nil {
			if writeErr := h.writeResponse(downstream, req, resp); writeErr != nil {
				if isExpectedDisconnect(writeErr) {
					return nil
				}
				h.config.OnError(state, req, PhaseResponseWrite, writeErr)
				return NewPhaseError(PhaseResponseWrite, fmt.Errorf("unable to write response to downstream: %w", writeErr))
			}
		}
		if err != nil {
			return err
		}

		if state.Upstream() != nil || state.UpstreamAddr() != "" {
			return h.proceedWithConnect(ctx, state.UpstreamAddr(), state.Upstream(), connWithBufferedReader(downstream, reader))
		}
		if req.Close || (resp != nil && resp.Close) {
			return nil
		}

		req, err = http.ReadRequest(reader)
		if err != nil {
			if isUnexpected(err) {
				if errResp := h.config.OnError(state, nil, PhaseReadRequest, err); errResp != nil {
					_ = h.writeResponse(downstream, nil, errResp)
				}
				return NewPhaseError(PhaseReadRequest, fmt.Errorf("unable to read next request from downstream: %w", err))
			}
			return nil
		}
		req = req.WithContext(ctx)
		req.RemoteAddr = remote
	}
}

func (h *Handler) proceedWithConnect(ctx context.Context, upstreamAddr string, upstream, downstream net.Conn) error {
	var upstreamFailure error
	if upstream == nil {
		setupCtx, cancelSetup := context.WithCancelCause(ctx)
		prefetch := startConnectPrefetch(downstream, maxTLSReplaySize, cancelSetup)

		var err error
		upstream, err = h.config.Dial(setupCtx, true, "tcp", upstreamAddr)
		var prefetchErr error
		downstream, prefetchErr = prefetch.stop()
		setupCause := context.Cause(setupCtx)
		cancelSetup(nil)

		if prefetchErr != nil {
			if isExpectedDisconnect(prefetchErr) || errors.Is(prefetchErr, context.Canceled) {
				if upstream != nil {
					_ = upstream.Close()
				}
				return nil
			}
			failure := NewPhaseError(PhaseDownstreamTLS, fmt.Errorf("CONNECT prefetch failed: %w", prefetchErr))
			h.config.OnError(newState(nil, nil, downstream), nil, PhaseDownstreamTLS, failure)
			if upstream != nil {
				_ = upstream.Close()
			}
			return failure
		}
		if err != nil && setupCause != nil && isExpectedDisconnect(setupCause) {
			return nil
		}
		if err != nil {
			phase := Phase(err)
			if phase == "" {
				phase = PhaseConnectDial
			}
			upstreamFailure = NewPhaseError(phase, err)
		}
	}
	if upstream != nil {
		defer upstream.Close()
	}

	recordedDownstream := newRecordingConn(downstream, maxTLSReplaySize)
	alertDownstream := &alertDetectingConn{Conn: recordedDownstream}
	tlsDown := tls.Server(alertDownstream, h.serverTLSConfig)
	if err := tlsDown.HandshakeContext(ctx); err != nil {
		if alertDownstream.sawAlert() || errors.As(err, new(tls.RecordHeaderError)) {
			if upstreamFailure != nil {
				return upstreamFailure
			}
			if _, copyErr := io.Copy(upstream, recordedDownstream.replay()); copyErr != nil {
				return NewPhaseError(PhaseTunnel, fmt.Errorf("unable to replay TLS input upstream: %w", copyErr))
			}
			return h.tunnel(upstream, downstream)
		}
		if isExpectedDisconnect(err) || errors.Is(err, context.Canceled) {
			return nil
		}
		failure := NewPhaseError(PhaseDownstreamTLS, fmt.Errorf("unable to MITM connection: %w", err))
		h.config.OnError(newState(nil, nil, downstream), nil, PhaseDownstreamTLS, failure)
		return failure
	}
	recordedDownstream.stop()
	downstreamTLS := net.Conn(&wrappedConn{Conn: tlsDown, wrapped: downstream})

	if upstreamFailure != nil {
		return h.handle(ctx, downstreamTLS, downstreamTLS, nil, upstreamFailure, true)
	}

	if skipsMITM(upstream) {
		return h.handle(ctx, downstreamTLS, downstreamTLS, upstream, nil, true)
	}
	clientConfig := h.clientTLSConfig.Clone()
	clientConfig.ServerName = tlsDown.ConnectionState().ServerName
	tlsUpstream := tls.Client(upstream, clientConfig)
	if err := tlsUpstream.HandshakeContext(ctx); err != nil {
		failure := NewPhaseError(PhaseUpstreamTLS, fmt.Errorf("unable to MITM connection: %w", err))
		return h.handle(ctx, downstreamTLS, downstreamTLS, nil, failure, true)
	}
	upstreamTLS := &wrappedConn{Conn: tlsUpstream, wrapped: upstream}
	return h.handle(ctx, downstreamTLS, downstreamTLS, upstreamTLS, nil, true)
}

func skipsMITM(conn net.Conn) bool {
	skip := false
	walkWrapped(conn, func(candidate net.Conn) bool {
		_, skip = candidate.(interface{ MITMSkipEncryption() })
		return !skip
	})
	return skip
}

func (h *Handler) tunnel(upstream, downstream net.Conn) error {
	writeErr, readErr := bidirectionalCopy(upstream, downstream)
	if isUnexpected(readErr) {
		return NewPhaseError(PhaseTunnel, fmt.Errorf("error piping data to downstream: %w", readErr))
	}
	if isUnexpected(writeErr) {
		return NewPhaseError(PhaseTunnel, fmt.Errorf("error piping data to upstream: %w", writeErr))
	}
	return nil
}

type wrappedConn struct {
	net.Conn
	wrapped net.Conn
}

func (c *wrappedConn) Wrapped() net.Conn { return c.wrapped }

type alertDetectingConn struct {
	net.Conn
	initialized atomic.Bool
	alerted     atomic.Bool
}

func (c *alertDetectingConn) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if c.initialized.CompareAndSwap(false, true) && p[0] == recordTypeAlert {
		c.alerted.Store(true)
	}
	if c.sawAlert() {
		return len(p), nil
	}
	return c.Conn.Write(p)
}
func (c *alertDetectingConn) sawAlert() bool { return c.alerted.Load() }

func (h *Handler) writeResponse(downstream io.Writer, req *http.Request, resp *http.Response) error {
	if resp.Request == nil {
		resp.Request = req
	}
	if resp.ProtoMajor == 0 {
		resp.Proto = "HTTP/1.1"
		resp.ProtoMajor = 1
		resp.ProtoMinor = 1
	}
	belowHTTP11 := !resp.ProtoAtLeast(1, 1)
	if belowHTTP11 && resp.StatusCode < 200 {
		downstream = io.Discard
	} else {
		resp = prepareResponse(resp, belowHTTP11)
	}
	err := resp.Write(downstream)
	if err != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	return err
}

func prepareRequest(req *http.Request) *http.Request {
	req.Proto = "HTTP/1.1"
	req.ProtoMajor = 1
	req.ProtoMinor = 1
	req.Close = false
	header := make(http.Header)
	copyHeadersForForwarding(header, req.Header)
	header.Set("Host", req.Host)
	req.Header = header
	req.URL = cloneURL(req.URL)
	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}
	req.URL.Host = req.Host
	if req.UserAgent() == "" {
		req.Header.Del("User-Agent")
	}
	return req
}

func prepareResponse(resp *http.Response, belowHTTP11 bool) *http.Response {
	original := resp.Header
	resp.Header = make(http.Header)
	copyHeadersForForwarding(resp.Header, original)
	if resp.Header.Get("Date") == "" {
		resp.Header.Set("Date", time.Now().Format(time.RFC850))
	}
	if belowHTTP11 {
		resp.TransferEncoding = nil
	}
	return resp
}

func cloneURL(input *url.URL) *url.URL { output := *input; return &output }

func copyHeadersForForwarding(dst, src http.Header) {
	var extraHopHeaders []string
	for key, values := range src {
		switch key {
		case "Connection":
			extraHopHeaders = values
		case "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "TE", "Trailers", "Transfer-Encoding", "Upgrade":
		default:
			if !slices.Contains(extraHopHeaders, key) {
				for _, value := range values {
					dst.Add(key, value)
				}
			}
		}
	}
	if proxyConnection := src.Get("Proxy-Connection"); proxyConnection != "" {
		dst.Set("Connection", proxyConnection)
	}
	dst.Del("Proxy-Connection")
}

func isExpectedDisconnect(err error) bool {
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) ||
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

func isUnexpected(err error) bool {
	if isExpectedDisconnect(err) {
		return false
	}
	var netErr net.Error
	return !(errors.As(err, &netErr) && netErr.Timeout())
}
