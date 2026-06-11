package proxycompat

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/getlantern/netx"
	"github.com/getlantern/proxy/v3/filters"
	"github.com/getlantern/reconn"
)

const connectRequest = "CONNECT %v HTTP/1.1\r\nHost: %v\r\n\r\n"

func (proxy *proxy) nextCONNECT(dialCtx context.Context, respondOK bool) filters.Next {
	return func(cs *filters.ConnectionState, modifiedReq *http.Request) (*http.Response, *filters.ConnectionState, error) {
		var resp *http.Response
		upstreamAddr := modifiedReq.URL.Host
		nextCS := cs.Clone()
		nextCS.SetUpstreamAddr(upstreamAddr)

		if !proxy.OKWaitsForUpstream {
			if respondOK {
				resp, nextCS = doRespondOK(nextCS, modifiedReq)
			}
			if proxy.OKSendsServerTiming {
				addDialUpstreamHeader(resp, 0)
			}
			return resp, nextCS, nil
		}

		var start time.Time
		if proxy.OKSendsServerTiming {
			start = time.Now()
		}

		dialCtx, cancelDial := addDialDeadlineIfNecessary(dialCtx, modifiedReq)
		upstream, err := proxy.Dial(dialCtx, true, "tcp", upstreamAddr)
		cancelDial()
		if err != nil {
			if proxy.OKWaitsForUpstream {
				return badGateway(cs, modifiedReq, err)
			}
			return nil, cs, err
		}

		if respondOK {
			resp, nextCS = doRespondOK(nextCS, modifiedReq)
		}
		if proxy.OKSendsServerTiming {
			addDialUpstreamHeader(resp, time.Since(start))
		}

		nextCS.SetUpstream(upstream)
		return resp, nextCS, nil
	}
}

func addDialUpstreamHeader(resp *http.Response, duration time.Duration) {
	if resp == nil {
		return
	}
	resp.Header.Add(serverTimingHeader, fmt.Sprintf("dialupstream;dur=%d", duration/time.Millisecond))
}

func addDialDeadlineIfNecessary(ctx context.Context, req *http.Request) (context.Context, context.CancelFunc) {
	timeoutString := req.Header.Get(DialTimeoutHeader)
	if timeoutString == "" {
		return ctx, noopCancel
	}

	timeoutInt, err := strconv.ParseInt(timeoutString, 10, 64)
	if err != nil {
		log.Errorf("Invalid %v, expected integer, got '%v'", DialTimeoutHeader, timeoutString)
		return ctx, noopCancel
	}

	newDeadline := time.Now().Add(time.Duration(timeoutInt) * time.Millisecond)
	existingDeadline, contextHasDeadline := ctx.Deadline()
	if contextHasDeadline && existingDeadline.Before(newDeadline) {
		return ctx, noopCancel
	}

	return context.WithDeadline(ctx, newDeadline)
}

func noopCancel() {}

func (proxy *proxy) Connect(dialCtx context.Context, in io.Reader, conn net.Conn, origin string) error {
	pin := io.MultiReader(strings.NewReader(fmt.Sprintf(connectRequest, origin, origin)), in)
	return proxy.handle(dialCtx, pin, conn, nil, false, false, false)
}

func (proxy *proxy) proceedWithConnect(
	dialCtx context.Context, req *http.Request,
	upstreamAddr string, upstream net.Conn, downstream net.Conn, respondOK bool) error {

	if upstream == nil {
		var dialErr error
		upstream, dialErr = proxy.Dial(dialCtx, true, "tcp", upstreamAddr)
		if dialErr != nil {
			return dialErr
		}
	}
	defer func() {
		if closeErr := upstream.Close(); closeErr != nil {
			log.Tracef("Error closing upstream connection: %s", closeErr)
		}
	}()

	var rr io.Reader
	if proxy.ShouldMITM != nil && proxy.ShouldMITM(req, upstreamAddr) {
		downstreamMITM, upstreamMITM, mitming, err := proxy.mitmIC.MITM(downstream, upstream)
		if err != nil {
			return fmt.Errorf("unable to MITM connection: %w", err)
		}
		downstream = wrapConn(downstreamMITM, downstream)
		upstream = wrapConn(upstreamMITM, upstream)
		if mitming {
			downstreamRR := reconn.Wrap(downstream, maxHTTPSize)
			_, peekReqErr := http.ReadRequest(bufio.NewReader(downstreamRR))
			var rrErr error
			rr, rrErr = downstreamRR.Rereader()
			if rrErr != nil {
				return fmt.Errorf("unable to re-read data: %w", rrErr)
			}
			if peekReqErr == nil {
				fullDownstream := io.MultiReader(rr, downstream)
				return proxy.handle(dialCtx, fullDownstream, downstream, upstream, respondOK, true, true)
			}
		}
	}

	bufOut := proxy.BufferSource.Get()
	bufIn := proxy.BufferSource.Get()
	defer proxy.BufferSource.Put(bufOut)
	defer proxy.BufferSource.Put(bufIn)

	if rr != nil {
		_, copyErr := io.CopyBuffer(upstream, rr, *bufOut)
		if copyErr != nil {
			return fmt.Errorf("error copying initial data to upstream: %w", copyErr)
		}
	}

	writeErr, readErr := netx.BidiCopy(upstream, downstream, *bufOut, *bufIn)
	if isUnexpected(readErr) {
		return fmt.Errorf("error piping data to downstream: %w", readErr)
	}
	if isUnexpected(writeErr) {
		return fmt.Errorf("error piping data to upstream at %v: %w", upstream.RemoteAddr(), writeErr)
	}
	return nil
}

func badGateway(cs *filters.ConnectionState, req *http.Request, err error) (*http.Response, *filters.ConnectionState, error) {
	log.Debugf("Responding BadGateway: %v", err)
	return filters.Fail(cs, req, http.StatusBadGateway, err)
}

func doRespondOK(cs *filters.ConnectionState, req *http.Request) (*http.Response, *filters.ConnectionState) {
	resp, nextCS, _ := filters.ShortCircuit(cs, req, &http.Response{StatusCode: http.StatusOK})
	return resp, nextCS
}
