package recorderproxy

import (
	"context"
	"net"
	"net/http"

	context2 "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	"github.com/getlantern/netx"
	"github.com/getlantern/proxy/v3/filters"
)

type proxyContextCarrier interface {
	ProxyContext() context.Context
}

func proxyContextFromConn(conn net.Conn) context.Context {
	if conn == nil {
		return nil
	}

	var baseCtx context.Context
	netx.WalkWrapped(conn, func(candidate net.Conn) bool {
		carrier, ok := candidate.(proxyContextCarrier)
		if !ok {
			return true
		}
		baseCtx = carrier.ProxyContext()
		return false
	})
	return baseCtx
}

func filterContext(cs *filters.ConnectionState, req *http.Request) context.Context {
	if req != nil {
		requestCtx := req.Context()
		if context2.HasStateHandle(requestCtx) {
			return requestCtx
		}
		if cs != nil {
			if baseCtx := proxyContextFromConn(cs.Downstream()); baseCtx != nil {
				return context2.WithStateHandle(requestCtx, baseCtx)
			}
		}
		return context2.RecordProxyDataAware(requestCtx)
	}

	if cs != nil {
		if baseCtx := proxyContextFromConn(cs.Downstream()); baseCtx != nil {
			return baseCtx
		}
	}

	return context2.RecordProxyDataAware(context.Background())
}
