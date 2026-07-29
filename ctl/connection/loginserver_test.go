package connection

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestLoopbackServerAcceptsValidCallbackAfterWrongState(t *testing.T) {
	server, err := newLoopbackServer()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	wait := server.Wait(ctx, "correct-state")

	response := callbackRequest(t, server.RedirectURI(), url.Values{
		"code":  {"wrong-code"},
		"state": {"wrong-state"},
	})
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("wrong-state status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}
	_ = response.Body.Close()

	response = callbackRequest(t, server.RedirectURI(), url.Values{
		"code":  {"valid-code"},
		"state": {"correct-state"},
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("valid callback status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	_ = response.Body.Close()

	code, err := wait()
	if err != nil {
		t.Fatal(err)
	}
	if code != "valid-code" {
		t.Fatalf("code = %q, want %q", code, "valid-code")
	}
}

func TestLoopbackServerReturnsOAuthError(t *testing.T) {
	server, err := newLoopbackServer()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	wait := server.Wait(ctx, "state")
	response := callbackRequest(t, server.RedirectURI(), url.Values{
		"error":             {"access_denied"},
		"error_description": {"user cancelled"},
		"state":             {"state"},
	})
	_ = response.Body.Close()

	if _, err := wait(); err == nil {
		t.Fatal("expected OAuth callback error")
	}
}

func TestLoopbackServerTimesOut(t *testing.T) {
	server, err := newLoopbackServer()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	wait := server.Wait(ctx, "state")
	cancel()
	if _, err := wait(); err == nil {
		t.Fatal("expected timeout error")
	}
}

func callbackRequest(t *testing.T, redirectURI string, values url.Values) *http.Response {
	t.Helper()
	callbackURL, err := url.Parse(redirectURI)
	if err != nil {
		t.Fatal(err)
	}
	callbackURL.RawQuery = values.Encode()
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, callbackURL.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}
