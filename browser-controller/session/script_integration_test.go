//go:build integration

/*
 * Copyright 2020 National Library of Norway.
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

package session

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	testcontainersupport "github.com/NationalLibraryOfNorway/veidemann/browser-controller/internal/testcontainers"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type callScriptResult struct {
	WaitForData bool `json:"waitForData"`
	Data        struct {
		Answer int `json:"answer"`
	} `json:"data"`
}

func TestCallScriptAwaitPromise(t *testing.T) {
	ctx, cleanup := newScriptExecutionTestContext(t)
	defer cleanup()

	eci, err := getExecutionContextID(ctx)
	if err != nil {
		t.Fatalf("getExecutionContextID() error = %v", err)
	}

	arguments := json.RawMessage(`{"waitForData":true,"answer":42}`)
	tests := []struct {
		name                string
		functionDeclaration string
	}{
		{
			name: "sync return value",
			functionDeclaration: `function script(args) {
				return {waitForData: args.waitForData, data: {answer: args.answer}};
			}`,
		},
		{
			name: "promise return value",
			functionDeclaration: `function script(args) {
				return Promise.resolve({waitForData: args.waitForData, data: {answer: args.answer}});
			}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := callScript(ctx, eci, test.functionDeclaration, arguments)
			if err != nil {
				t.Fatalf("callScript() error = %v", err)
			}

			var got callScriptResult
			if err := json.Unmarshal(result, &got); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if !got.WaitForData {
				t.Fatalf("WaitForData = %v, want true", got.WaitForData)
			}
			if got.Data.Answer != 42 {
				t.Fatalf("Data.Answer = %d, want 42", got.Data.Answer)
			}
		})
	}
}

func TestEvaluateScriptAwaitPromise(t *testing.T) {
	ctx, cleanup := newScriptExecutionTestContext(t)
	defer cleanup()

	want := []string{"https://example.com/a", "https://example.com/b"}
	tests := []struct {
		name       string
		expression string
	}{
		{
			name: "sync expression",
			expression: `(() => {
				return ["https://example.com/a", "https://example.com/b"];
			})()`,
		},
		{
			name: "promise expression",
			expression: `(() => {
				return Promise.resolve(["https://example.com/a", "https://example.com/b"]);
			})()`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := evaluateScript(ctx, test.expression)
			if err != nil {
				t.Fatalf("evaluateScript() error = %v", err)
			}

			var got []string
			if err := json.Unmarshal(result, &got); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if len(got) != len(want) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
				}
			}
		})
	}
}

func newScriptExecutionTestContext(t *testing.T) (context.Context, func()) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	browserless, host, port := startBrowserlessContainer(t, ctx)

	t.Log("Started browserless container", "host", host, "port", port)

	sess := newDefaultSession(WithBrowserHost(host), WithBrowserPort(port))
	sess.Id = 0
	sess.logger = slog.Default().With("session", sess.Id)

	browserWsEndpoint, err := sess.compileBrowserWebsocketEndpoint()
	if err != nil {
		cancel()
		terminateBrowserlessContainer(browserless)
		t.Fatalf("failed to compile browser websocket endpoint: %v", err)
	}
	sess.browserWsEndpoint = browserWsEndpoint

	allocatorContext, allocatorCancel := chromedp.NewRemoteAllocator(ctx, sess.browserWsEndpoint, chromedp.NoModifyURL)
	cdpCtx, cdpCancel := chromedp.NewContext(allocatorContext)

	if err := chromedp.Run(cdpCtx,
		page.Enable(),
		runtime.Enable(),
		chromedp.Navigate("data:text/html,<html><body>script test</body></html>"),
	); err != nil {
		cdpCancel()
		allocatorCancel()
		terminateBrowserlessContainer(browserless)
		cancel()
		t.Fatalf("chromedp.Run() error = %v", err)
	}

	cleanup := func() {
		cdpCancel()
		allocatorCancel()
		terminateBrowserlessContainer(browserless)
		cancel()
	}

	return cdpCtx, cleanup
}

func startBrowserlessContainer(t *testing.T, ctx context.Context) (testcontainers.Container, string, int) {
	t.Helper()

	browserless, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Env: map[string]string{
				"DEBUG": "*",
			},
			Image:        testcontainersupport.BrowserlessChromium,
			ExposedPorts: []string{"3000/tcp"},
			WaitingFor:   wait.ForListeningPort("3000/tcp"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("starting browserless container: %v", err)
	}

	host, err := browserless.Host(ctx)
	if err != nil {
		terminateBrowserlessContainer(browserless)
		t.Fatalf("browserless.Host() error = %v", err)
	}
	if host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}

	port, err := browserless.MappedPort(ctx, "3000/tcp")
	if err != nil {
		terminateBrowserlessContainer(browserless)
		t.Fatalf("browserless.MappedPort() error = %v", err)
	}

	return browserless, host, int(port.Num())
}

func terminateBrowserlessContainer(container testcontainers.Container) {
	if container == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = container.Terminate(ctx)
}
