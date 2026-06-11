package proxycompat

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/mitm"
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
	OnError      func(cs *filters.ConnectionState, req *http.Request, read bool, err error) *http.Response

	OKWaitsForUpstream  bool
	OKSendsServerTiming bool

	Dial       DialFunc
	ShouldMITM func(req *http.Request, upstreamAddr string) bool
	MITMOpts   *mitm.Opts
	InitMITM   func() (MITMInterceptor, error)
}

type MITMInterceptor interface {
	MITM(downstream net.Conn, upstream net.Conn) (newDown net.Conn, newUp net.Conn, success bool, err error)
}

type proxy struct {
	*Opts
	mitmIC      MITMInterceptor
	mitmDomains []*regexp.Regexp
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

	p := &proxy{
		Opts:        opts,
		mitmDomains: make([]*regexp.Regexp, 0),
	}
	p.applyCONNECTDefaults()

	if opts.InitMITM != nil {
		mitmIC, err := opts.InitMITM()
		if err != nil {
			log.Errorf("Unable to configure MITM: %v", err)
		} else {
			p.mitmIC = mitmIC
			if opts.MITMOpts != nil {
				for _, domain := range opts.MITMOpts.Domains {
					re, err := domainToRegex(domain)
					if err != nil {
						log.Errorf("Unable to convert domain %v to regex: %v", domain, err)
						continue
					}
					p.mitmDomains = append(p.mitmDomains, re)
				}
			}
		}
	}

	return p
}

func (proxy *proxy) applyCONNECTDefaults() {
	if proxy.InitMITM == nil && proxy.MITMOpts != nil {
		proxy.InitMITM = proxy.defaultInitMITM
	}
	if proxy.ShouldMITM == nil {
		proxy.ShouldMITM = proxy.defaultShouldMITM
	} else {
		orig := proxy.ShouldMITM
		proxy.ShouldMITM = func(req *http.Request, upstreamAddr string) bool {
			if !orig(req, upstreamAddr) {
				return false
			}
			return proxy.defaultShouldMITM(req, upstreamAddr)
		}
	}
}

func (proxy *proxy) defaultShouldMITM(req *http.Request, upstreamAddr string) bool {
	if proxy.mitmIC == nil {
		return false
	}
	host, _, err := net.SplitHostPort(upstreamAddr)
	if err != nil {
		return false
	}
	for _, mitmDomain := range proxy.mitmDomains {
		if mitmDomain.MatchString(host) {
			return true
		}
	}
	return false
}

func (proxy *proxy) defaultInitMITM() (MITMInterceptor, error) {
	i, err := mitm.Configure(proxy.MITMOpts)
	return &defaultMITMInterceptor{Interceptor: i}, err
}

type defaultMITMInterceptor struct {
	*mitm.Interceptor
}

func (i *defaultMITMInterceptor) MITM(downstream net.Conn, upstream net.Conn) (newDown net.Conn, newUp net.Conn, success bool, err error) {
	return i.Interceptor.MITM(downstream, upstream)
}

func (opts *Opts) addIdleKeepAlive(header http.Header) {
	if opts.IdleTimeout > 0 {
		header.Set("Keep-Alive", fmt.Sprintf("timeout=%d", int(opts.IdleTimeout.Seconds())-2))
	}
}
