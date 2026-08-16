/*
 * Copyright 2019 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package recorderproxy

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	proxyengine "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/internal/proxy"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
)

func (proxy *RecorderProxy) Dial(ctx context.Context, isConnect bool, network, addr string) (net.Conn, error) {
	log := rpcontext.LogWithContext(ctx, "Dialer")
	log.Debugf("dial upstream %v, is connect request: %v", addr, isConnect)

	timeout := dialTimeout(ctx, 30*time.Second)

	dialAddr := addr
	if proxy.nextProxy != "" {
		dialAddr = proxy.nextProxy
	}

	conn, err := net.DialTimeout(network, dialAddr, timeout)
	if err != nil {
		if proxy.nextProxy != "" {
			log.Errorf("could not dial next proxy at %v: %v", proxy.nextProxy, err)
			return nil, err
		}

		log.Errorf("Failed to dial %v: %v", addr, err)

		return nil, err
	}

	if logger.IsLevelEnabled(logger.DebugLevel) {
		conn = WrapConn(conn, "up", true)
	}

	if isConnect && proxy.nextProxy != "" {
		if err := proxy.sendConnectToNextProxy(ctx, conn, log); err != nil {
			return conn, proxyengine.NewPhaseError(proxyengine.PhaseUpstreamProxyConnect, err)
		}
	}

	return conn, nil
}

func dialTimeout(ctx context.Context, fallback time.Duration) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return fallback
	}

	timeout := time.Until(deadline)
	if timeout <= 0 {
		return 0
	}

	return timeout
}

func (proxy *RecorderProxy) sendConnectToNextProxy(ctx context.Context, conn net.Conn, log *logger.Logger) error {
	wrappedCtx := rpcontext.WrapIfNecessary(ctx)
	uri := rpcontext.GetUri(wrappedCtx)

	req := NewConnectReq(uri.Host)
	log.Debugf("sending CONNECT for host %v to upstream proxy", req.URL)

	if err := req.Write(conn); err != nil {
		log.WithError(err).Warn("error while writing CONNECT request to upstream proxy")
		return err
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		log.WithError(err).Warn("error while reading CONNECT response from upstream proxy")
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	log.Debugf("response status from CONNECT request to upstream proxy was: %v", resp.Status)

	squidErr := resp.Header.Get("X-Squid-Error")
	if squidErr != "" {
		return handleSquidErrorString(squidErr)
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Error(
			errors.RuntimeException,
			fmt.Sprintf("could not connect to upstream proxy (%d)", resp.StatusCode),
			squidErr,
		)
	}

	return nil
}

func NewConnectReq(authority string) *http.Request {
	if authority == "" {
		panic("BUG: empty CONNECT authority")
	}

	return &http.Request{
		Method: http.MethodConnect,
		URL: &url.URL{
			Scheme: "http",
			Opaque: authority,
			Host:   authority,
		},
		Host:       authority,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
	}
}
