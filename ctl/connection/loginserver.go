package connection

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type callbackResult struct {
	code string
	err  error
}

type loopbackServer struct {
	listener      net.Listener
	server        *http.Server
	result        chan callbackResult
	serveErr      chan error
	expectedState string
	start         sync.Once
	close         sync.Once
}

func newLoopbackServer() (*loopbackServer, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       5 * time.Second,
	}
	loopback := &loopbackServer{
		listener: listener,
		server:   server,
		result:   make(chan callbackResult, 1),
		serveErr: make(chan error, 1),
	}
	mux.HandleFunc("/callback", loopback.handleCallback)
	return loopback, nil
}

func (s *loopbackServer) RedirectURI() string {
	port := s.listener.Addr().(*net.TCPAddr).Port
	return "http://localhost:" + strconv.Itoa(port) + "/callback"
}

// Wait starts serving before returning a function that waits for the callback.
// This lets callers open the browser only after the loopback port is bound and
// the HTTP server is running.
func (s *loopbackServer) Wait(ctx context.Context, expectedState string) func() (string, error) {
	s.start.Do(func() {
		s.expectedState = expectedState
		go func() {
			err := s.server.Serve(s.listener)
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.serveErr <- err
			}
		}()
	})

	return func() (string, error) {
		for {
			select {
			case result := <-s.result:
				if result.err != nil {
					return "", result.err
				}
				return result.code, nil
			case err := <-s.serveErr:
				return "", fmt.Errorf("login callback server failed: %w", err)
			case <-ctx.Done():
				return "", fmt.Errorf("timed out waiting for login callback: %w", ctx.Err())
			}
		}
	}
}

func (s *loopbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/callback" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	state := query.Get("state")
	if state != s.expectedState {
		http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}

	if oauthError := query.Get("error"); oauthError != "" {
		http.Error(w, "Login was not completed", http.StatusBadRequest)
		s.send(callbackResult{err: oauthCallbackError(query)})
		return
	}
	code := query.Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		s.send(callbackResult{err: errors.New("identity provider callback did not contain an authorization code")})
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, "<html><body><h1>Window can safely be closed</h1></body></html>")
	s.send(callbackResult{code: code})
}

func oauthCallbackError(query url.Values) error {
	if description := query.Get("error_description"); description != "" {
		return fmt.Errorf("identity provider returned %s: %s", query.Get("error"), description)
	}
	return fmt.Errorf("identity provider returned %s", query.Get("error"))
}

func (s *loopbackServer) send(result callbackResult) {
	select {
	case s.result <- result:
	default:
	}
}

func (s *loopbackServer) Close() {
	s.close.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.server.Shutdown(ctx)
		_ = s.listener.Close()
	})
}
