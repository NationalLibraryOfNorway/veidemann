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
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
)

func TestNewConnectReqWrite(t *testing.T) {
	req := NewConnectReq("example.com:443")

	var b bytes.Buffer
	if err := req.Write(&b); err != nil {
		t.Fatal(err)
	}

	got := b.String()

	if !strings.HasPrefix(got, "CONNECT example.com:443 HTTP/1.1\r\n") {
		t.Fatalf("bad request line:\n%s", got)
	}

	if !strings.Contains(got, "Host: example.com:443\r\n") {
		t.Fatalf("missing Host header:\n%s", got)
	}
}

func TestSendConnectToNextProxyCancellationClosesConnection(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })
	t.Cleanup(func() { _ = server.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	ctx = rpcontext.RecordProxyDataAware(ctx)
	rpcontext.SetUri(ctx, &url.URL{Scheme: "https", Host: "example.com"})

	requestRead := make(chan error, 1)
	go func() {
		_, err := http.ReadRequest(bufio.NewReader(server))
		requestRead <- err
	}()

	done := make(chan error, 1)
	go func() {
		done <- (&RecorderProxy{}).sendConnectToNextProxy(ctx, client, logger.LogWithComponent("TEST"))
	}()
	if err := <-requestRead; err != nil {
		t.Fatal(err)
	}
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("sendConnectToNextProxy() error = %v, want context canceled", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("CONNECT response read was not canceled")
	}
}
