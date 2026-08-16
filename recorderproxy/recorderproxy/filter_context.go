package recorderproxy

import (
	"context"
	"net"
	"net/http"

	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	proxy "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/internal/proxy"
)

type proxyContextCarrier interface {
	ProxyContext() context.Context
}

func proxyContextFromConn(conn net.Conn) context.Context {
	if conn == nil {
		return nil
	}

	var baseCtx context.Context
	for candidate := conn; candidate != nil; {
		carrier, ok := candidate.(proxyContextCarrier)
		if ok {
			baseCtx = carrier.ProxyContext()
			break
		}
		wrapped, ok := candidate.(interface{ Wrapped() net.Conn })
		if !ok || wrapped.Wrapped() == candidate {
			break
		}
		candidate = wrapped.Wrapped()
	}
	return baseCtx
}

func filterContext(cs *proxy.State, req *http.Request) context.Context {
	if req != nil {
		requestCtx := req.Context()
		if rpcontext.HasStateHandle(requestCtx) {
			return requestCtx
		}
		if cs != nil {
			if baseCtx := proxyContextFromConn(cs.Downstream()); baseCtx != nil {
				return rpcontext.WithStateHandle(requestCtx, baseCtx)
			}
		}
		return rpcontext.RecordProxyDataAware(requestCtx)
	}

	if cs != nil {
		if baseCtx := proxyContextFromConn(cs.Downstream()); baseCtx != nil {
			return baseCtx
		}
	}

	return rpcontext.RecordProxyDataAware(context.Background())
}
