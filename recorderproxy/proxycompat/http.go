package proxycompat

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	gerrors "github.com/getlantern/errors"
	"github.com/getlantern/netx"
	"github.com/getlantern/preconn"
	"github.com/getlantern/proxy/v3/filters"
)

func (proxy *proxy) Handle(dialCtx context.Context, downstreamIn io.Reader, downstream net.Conn) (err error) {
	return proxy.handle(dialCtx, downstreamIn, downstream, nil, true, false, false)
}

func safeClose(conn net.Conn) {
	defer func() {
		if p := recover(); p != nil {
			log.Errorf("Panic on closing connection: %v", p)
		}
	}()
	_ = conn.Close()
}

func (proxy *proxy) logInitialReadError(downstream net.Conn, err error) error {
	rem := downstream.RemoteAddr()
	r := ""
	if rem != nil {
		r = rem.String()
	}
	txt := err.Error()
	if strings.HasPrefix(txt, "Client Hello has no cipher suites") {
		log.Debugf("No cipher suites in common -- old Lantern client")
		return err
	}
	if strings.Contains(txt, "oversized") {
		return log.Errorf("Oversized record on initial read: %v from %v", err, r)
	}
	if strings.Contains(txt, "first record does not") {
		return log.Errorf("Not a TLS client connection: %v from %v", err, r)
	}
	return log.Errorf("%v from %v - Initial ReadRequest Error", err, r)
}

func (proxy *proxy) handle(dialCtx context.Context, downstreamIn io.Reader, downstream net.Conn, upstream net.Conn, respondOK bool, mitming bool, tunneledHTTP bool) (err error) {
	defer func() {
		if p := recover(); p != nil {
			safeClose(downstream)
			err = gerrors.New("Recovered from panic handling connection: %v", p)
			log.Errorf("Recovered from panic handling connection: %v", p)
		}
	}()
	defer func() {
		if closeErr := downstream.Close(); closeErr != nil {
			log.Tracef("Error closing downstream connection: %s", closeErr)
		}
	}()

	downstreamBuffered := bufio.NewReader(downstreamIn)
	req, err := http.ReadRequest(downstreamBuffered)
	if err != nil {
		if isUnexpected(err) {
			cs := filters.NewConnectionState(nil, nil, downstream)
			errResp := proxy.OnError(cs, req, true, err)
			if errResp != nil {
				_ = proxy.writeResponse(downstream, req, errResp)
			}

			return proxy.logInitialReadError(downstream, err)
		}
		return nil
	}
	if remoteAddr := downstream.RemoteAddr(); remoteAddr != nil {
		req.RemoteAddr = remoteAddr.String()
	}

	cs := filters.NewConnectionState(req, nil, downstream)
	if upstream != nil && !tunneledHTTP {
		cs.SetUpstream(upstream)
	}
	cs.SetMITMing(mitming)

	var next filters.Next
	if req.Method == http.MethodConnect {
		next = proxy.nextCONNECT(dialCtx, respondOK)
	} else {
		var tr idleClosingTransport
		if upstream != nil {
			cs.SetRequestAwareUpstream(upstream)
			tr = &addressLoggingTransport{
				Transport: &http.Transport{
					DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
						return &noCloseConn{upstream}, nil
					},
					MaxIdleConnsPerHost: -1,
				},
				upstream: upstream,
			}
		} else {
			tr = &http.Transport{
				DialContext:         proxy.requestAwareDial(cs),
				IdleConnTimeout:     proxy.IdleTimeout,
				MaxIdleConnsPerHost: 1,
			}
		}

		defer tr.CloseIdleConnections()
		next = proxy.nextNonCONNECT(tr)
	}

	return proxy.processRequests(dialCtx, cs, req.RemoteAddr, req, downstream, downstreamBuffered, next, respondOK)
}

func (proxy *proxy) requestAwareDial(cs *filters.ConnectionState) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := proxy.Dial(ctx, false, network, addr)
		if err == nil {
			cs.SetRequestAwareUpstream(conn)
			handleRequestAware(cs)
		}
		return conn, err
	}
}

func (proxy *proxy) nextNonCONNECT(tr idleClosingTransport) filters.Next {
	return func(cs *filters.ConnectionState, modifiedReq *http.Request) (*http.Response, *filters.ConnectionState, error) {
		modifiedReq = prepareRequest(modifiedReq)

		cs.SetRequestAwareRequest(modifiedReq)
		handleRequestAware(cs)
		resp, err := tr.RoundTrip(modifiedReq)
		handleResponseAware(cs, modifiedReq, resp, err)
		if err != nil {
			err = gerrors.New("Unable to round-trip http request to upstream: %v", err)
		}
		return resp, cs, err
	}
}

func (proxy *proxy) processRequests(dialCtx context.Context, cs *filters.ConnectionState,
	remoteAddr string, req *http.Request, downstream net.Conn, downstreamBuffered *bufio.Reader,
	next filters.Next, respondOK bool) error {

	var readErr error
	var resp *http.Response
	var err error

	for {
		if req.URL.Scheme == "" {
			req.URL.Scheme = cs.OriginalURLScheme()
		}
		if req.URL.Host == "" {
			req.URL.Host = cs.OriginalURLHost()
		}
		if req.Host == "" {
			req.Host = cs.OriginalHost()
		}
		resp, cs, err = proxy.Filter.Apply(cs, req, next)
		if err != nil && resp == nil {
			resp = proxy.OnError(cs, req, false, err)
			if resp != nil {
				log.Debugf("Closing client connection on error: %v", err)
				resp.Close = true
			}
		}

		if resp != nil {
			writeErr := proxy.writeResponse(downstream, req, resp)
			if writeErr != nil {
				if isUnexpected(writeErr) {
					return log.Errorf("Unable to write response to downstream: %v", writeErr)
				}
				return err
			}
		}

		if err != nil {
			return err
		}

		upstream := cs.Upstream()
		upstreamAddr := cs.UpstreamAddr()
		isConnect := upstream != nil || upstreamAddr != ""

		buffered := downstreamBuffered.Buffered()
		if buffered > 0 {
			b, _ := downstreamBuffered.Peek(buffered)
			downstream = preconn.Wrap(downstream, b)
		}

		if isConnect {
			return proxy.proceedWithConnect(dialCtx, req, upstreamAddr, upstream, downstream, respondOK)
		}

		if req.Close {
			return err
		}

		if resp != nil && resp.Close {
			return err
		}

		req, readErr = http.ReadRequest(downstreamBuffered)
		if readErr != nil {
			if isUnexpected(readErr) {
				errResp := proxy.OnError(cs, req, true, readErr)
				if errResp != nil {
					_ = proxy.writeResponse(downstream, req, errResp)
				}
				return log.Errorf("Unable to read next request from downstream: %v", readErr)
			}
			return err
		}

		cs.IncrementRequestNumber()
		req.RemoteAddr = remoteAddr
	}
}

func handleRequestAware(cs *filters.ConnectionState) {
	upstream := cs.RequestAwareUpstream()
	if upstream == nil {
		return
	}

	netx.WalkWrapped(upstream, func(wrapped net.Conn) bool {
		switch t := wrapped.(type) {
		case RequestAware:
			t.OnRequest(cs.RequestAwareRequest())
		}
		return true
	})
}

func handleResponseAware(cs *filters.ConnectionState, req *http.Request, resp *http.Response, err error) {
	upstream := cs.RequestAwareUpstream()
	if upstream == nil {
		return
	}

	netx.WalkWrapped(upstream, func(wrapped net.Conn) bool {
		switch t := wrapped.(type) {
		case ResponseAware:
			t.OnResponse(req, resp, err)
		}
		return true
	})
}

func (proxy *proxy) writeResponse(downstream io.Writer, req *http.Request, resp *http.Response) error {
	if resp.Request == nil {
		resp.Request = req
	}
	out := downstream
	if resp.ProtoMajor == 0 {
		resp.ProtoMajor = 1
		resp.ProtoMinor = 1
	}
	belowHTTP11 := !resp.ProtoAtLeast(1, 1)
	if belowHTTP11 && resp.StatusCode < 200 {
		out = io.Discard
	} else {
		resp = prepareResponse(resp, belowHTTP11)
		proxy.addIdleKeepAlive(resp.Header)
	}

	err := resp.Write(out)
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

	newHeader := make(http.Header)
	copyHeadersForForwarding(newHeader, req.Header)
	newHeader.Set("Host", req.Host)
	req.Header = newHeader

	req.URL = cloneURL(req.URL)
	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}
	req.URL.Host = req.Host

	userAgent := req.UserAgent()
	if userAgent == "" {
		req.Header.Del("User-Agent")
	} else {
		req.Header.Set("User-Agent", userAgent)
	}

	return req
}

func prepareResponse(resp *http.Response, belowHTTP11 bool) *http.Response {
	origHeader := resp.Header
	resp.Header = make(http.Header)
	copyHeadersForForwarding(resp.Header, origHeader)
	if resp.Header.Get("Date") == "" {
		resp.Header.Set("Date", time.Now().Format(time.RFC850))
	}
	if belowHTTP11 {
		resp.TransferEncoding = nil
	}
	return resp
}

func cloneURL(input *url.URL) *url.URL {
	out := *input
	return &out
}

func copyHeadersForForwarding(dst, src http.Header) {
	var extraHopByHopHeaders []string
	for k, vv := range src {
		switch k {
		case "Connection":
			extraHopByHopHeaders = vv
		case "Keep-Alive":
		case "Proxy-Authenticate":
		case "Proxy-Authorization":
		case "TE":
		case "Trailers":
		case "Transfer-Encoding":
		case "Upgrade":
		default:
			if !contains(k, extraHopByHopHeaders) {
				for _, v := range vv {
					dst.Add(k, v)
				}
			}
		}
	}

	pc := src.Get("Proxy-Connection")
	if pc != "" {
		dst.Set("Connection", pc)
	}
	dst.Del("Proxy-Connection")
}

func contains(k string, items []string) bool {
	for _, item := range items {
		if k == item {
			return true
		}
	}
	return false
}

func isUnexpected(err error) bool {
	if err == nil {
		return false
	}
	if err == io.EOF {
		return false
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return false
	}

	text := err.Error()
	return !strings.HasSuffix(text, "EOF") &&
		!strings.Contains(text, "i/o timeout") &&
		!strings.Contains(text, "Use of idled network connection") &&
		!strings.Contains(text, "use of closed network connection") &&
		!strings.Contains(text, "broken pipe") &&
		!strings.Contains(text, "closed pipe") &&
		!strings.Contains(text, "connection reset by peer")
}

func defaultFilter(cs *filters.ConnectionState, req *http.Request, next filters.Next) (*http.Response, *filters.ConnectionState, error) {
	return next(cs, req)
}

func defaultOnError(cs *filters.ConnectionState, req *http.Request, read bool, err error) *http.Response {
	return nil
}

type idleClosingTransport interface {
	RoundTrip(req *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

type addressLoggingTransport struct {
	*http.Transport
	upstream net.Conn
}

func (alt *addressLoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := alt.Transport.RoundTrip(req)
	if err != nil {
		err = gerrors.New("Error round-tripping to %v: %v", alt.upstream.RemoteAddr(), err)
	}
	return resp, err
}
