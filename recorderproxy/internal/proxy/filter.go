package proxy

import (
	"io"
	"net"
	"net/http"
	"strings"
)

// State contains the protocol state shared by filters for one connection.
type State struct {
	originalURLScheme string
	originalURLHost   string
	originalHost      string
	upstream          net.Conn
	upstreamAddr      string
	downstream        net.Conn
	mitming           bool
}

func newState(initialReq *http.Request, upstream, downstream net.Conn) *State {
	s := &State{upstream: upstream, downstream: downstream}
	if initialReq != nil {
		s.originalURLScheme = initialReq.URL.Scheme
		s.originalURLHost = initialReq.URL.Host
		s.originalHost = initialReq.Host
	}
	if upstream != nil && upstream.RemoteAddr() != nil {
		s.upstreamAddr = upstream.RemoteAddr().String()
	}
	return s
}

func (s *State) Clone() *State                       { clone := *s; return &clone }
func (s *State) Downstream() net.Conn                { return s.downstream }
func (s *State) Upstream() net.Conn                  { return s.upstream }
func (s *State) UpstreamAddr() string                { return s.upstreamAddr }
func (s *State) OriginalURLScheme() string           { return s.originalURLScheme }
func (s *State) OriginalURLHost() string             { return s.originalURLHost }
func (s *State) OriginalHost() string                { return s.originalHost }
func (s *State) IsMITMing() bool                     { return s.mitming }
func (s *State) SetMITMing(mitming bool)             { s.mitming = mitming }
func (s *State) SetUpstreamAddr(upstreamAddr string) { s.upstreamAddr = upstreamAddr }
func (s *State) SetUpstream(upstream net.Conn) {
	s.upstream = upstream
	if upstream != nil && upstream.RemoteAddr() != nil {
		s.upstreamAddr = upstream.RemoteAddr().String()
	}
}

// Filter intercepts a request around the next filter or transport.
type Filter interface {
	Apply(*State, *http.Request, Next) (*http.Response, *State, error)
}

type FilterFunc func(*State, *http.Request, Next) (*http.Response, *State, error)

func (f FilterFunc) Apply(s *State, req *http.Request, next Next) (*http.Response, *State, error) {
	return f(s, req, next)
}

type Next func(*State, *http.Request) (*http.Response, *State, error)

// Chain applies filters in declaration order.
type Chain []Filter

func Join(filters ...Filter) Chain             { return Chain(filters) }
func (c Chain) Append(filters ...Filter) Chain { return append(c, filters...) }

func (c Chain) Apply(s *State, req *http.Request, next Next) (*http.Response, *State, error) {
	var apply func(int, *State, *http.Request) (*http.Response, *State, error)
	apply = func(index int, current *State, currentReq *http.Request) (*http.Response, *State, error) {
		if index == len(c) {
			return next(current, currentReq)
		}
		return c[index].Apply(current, currentReq, func(nextState *State, nextReq *http.Request) (*http.Response, *State, error) {
			return apply(index+1, nextState, nextReq)
		})
	}
	return apply(0, s, req)
}

func ShortCircuit(s *State, req *http.Request, resp *http.Response) (*http.Response, *State, error) {
	if resp.Header == nil {
		resp.Header = make(http.Header)
	}
	if req != nil {
		resp.Proto = req.Proto
		resp.ProtoMajor = req.ProtoMajor
		resp.ProtoMinor = req.ProtoMinor
	}
	if resp.Body != nil && resp.ContentLength <= 0 && len(resp.TransferEncoding) == 0 {
		resp.ContentLength = -1
		resp.TransferEncoding = []string{"chunked"}
	}
	return resp, s, nil
}

func Fail(s *State, req *http.Request, statusCode int, err error) (*http.Response, *State, error) {
	message := ""
	if err != nil {
		message = err.Error()
	}
	resp := &http.Response{
		StatusCode:    statusCode,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader(message)),
		ContentLength: int64(len(message)),
		Close:         true,
	}
	if req != nil {
		resp.Proto = req.Proto
		resp.ProtoMajor = req.ProtoMajor
		resp.ProtoMinor = req.ProtoMinor
	}
	return resp, s, err
}
