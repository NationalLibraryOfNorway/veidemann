package connection

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestLoginServer(t *testing.T) {
	done := make(chan struct{})

	timeout := time.NewTimer(time.Second)
	const testCode = "abcd"
	const testState = "1234"

	go func() {
		defer close(done)
		code, state, err := listenAndWaitForAuthorizationCode(autoRedirectURI)
		if err != nil {
			t.Error(err)
		}
		if code != testCode {
			t.Errorf("want %s, got %s", testCode, code)
		}
		if state != testState {
			t.Errorf("want %s, get %s", testState, state)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, autoRedirectURI, nil)
	if err != nil {
		t.Error(err)
	}

	q := req.URL.Query()
	q.Add("code", testCode)
	q.Add("state", testState)
	req.URL.RawQuery = q.Encode()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	select {
	case <-done:
	case <-timeout.C:
		t.Error("timed out waiting for code")
	}
}
