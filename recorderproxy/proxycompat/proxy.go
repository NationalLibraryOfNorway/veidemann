package proxycompat

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/proxy/v3/filters"
)

const (
	defaultDialTimeout = 30 * time.Second

	serverTimingHeader = "Server-Timing"

	MetricDialUpstream = "dialupstream"

	DialTimeoutHeader = "X-Lantern-Dial-Timeout"
	maxHTTPSize       = 2 << 15
)

var log = golog.LoggerFor("proxycompat")

type DialFunc func(ctx context.Context, isCONNECT bool, network, addr string) (conn net.Conn, err error)

type OnErrorFunc func(cs *filters.ConnectionState, req *http.Request, phase ErrorPhase, err error) *http.Response

type Proxy interface {
	Handle(dialCtx context.Context, in io.Reader, conn net.Conn) error
	Connect(dialCtx context.Context, in io.Reader, conn net.Conn, origin string) error
	Serve(l net.Listener) error
}

type RequestAware interface {
	OnRequest(req *http.Request)
}

type ResponseAware interface {
	OnResponse(req *http.Request, resp *http.Response, err error)
}

type Opts struct {
	IdleTimeout time.Duration

	BufferSource BufferSource
	Filter       filters.Filter
	OnError      OnErrorFunc

	OKWaitsForUpstream  bool
	OKSendsServerTiming bool

	Dial       DialFunc
	ShouldMITM func(req *http.Request, upstreamAddr string) bool
	InitMITM   func() (MITMInterceptor, error)
}

type MITMInterceptor interface {
	MITM(downstream net.Conn, upstream net.Conn) (newDown net.Conn, newUp net.Conn, success bool, err error)
}

type proxy struct {
	*Opts
	mitmIC MITMInterceptor
}

func New(opts *Opts) Proxy {
	if opts == nil {
		opts = &Opts{}
	}
	if opts.Dial == nil {
		opts.Dial = func(ctx context.Context, isCONNECT bool, network, addr string) (conn net.Conn, err error) {
			ctx, cancel := context.WithTimeout(ctx, defaultDialTimeout)
			defer cancel()
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		}
	}
	if opts.Filter == nil {
		opts.Filter = filters.FilterFunc(defaultFilter)
	}
	if opts.OnError == nil {
		opts.OnError = defaultOnError
	}
	if opts.BufferSource == nil {
		opts.BufferSource = newBufferSource()
	}

	p := &proxy{Opts: opts}
	p.applyCONNECTDefaults()

	if opts.InitMITM != nil {
		mitmIC, err := opts.InitMITM()
		if err != nil {
			log.Errorf("Unable to configure MITM: %v", err)
		} else {
			p.mitmIC = mitmIC
		}
	}

	return p
}

func (proxy *proxy) applyCONNECTDefaults() {
	if proxy.ShouldMITM == nil {
		proxy.ShouldMITM = func(*http.Request, string) bool { return false }
	}
}

func (opts *Opts) addIdleKeepAlive(header http.Header) {
	if opts.IdleTimeout > 0 {
		header.Set("Keep-Alive", fmt.Sprintf("timeout=%d", int(opts.IdleTimeout.Seconds())-2))
	}
}

func defaultOnError(
	cs *filters.ConnectionState,
	req *http.Request,
	phase ErrorPhase,
	err error,
) *http.Response {
	return nil
}
