package connection

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// response is the response from the login server
type response struct {
	code  string
	state string
}

func handleLoginResponse(resp chan<- *response) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		fmt.Fprintf(w, "<html><body><h1>%s</h1></body></html>", "Window can safely be closed")
		resp <- &response{
			query.Get("code"),
			query.Get("state"),
		}
	}
}

// listenAndWaitForAuthorizationCode starts a http server and waits for the authorization code and state
func listenAndWaitForAuthorizationCode(uri string) (string, string, error) {
	var addr string
	if u, err := url.Parse(uri); err != nil {
		return "", "", err
	} else {
		addr = fmt.Sprintf("%s:%s", u.Hostname(), u.Port())
	}
	server := &http.Server{
		Addr: addr,
	}
	defer func() {
		go func() {
			_ = server.Shutdown(context.Background())
		}()
	}()

	response := make(chan *response)
	http.HandleFunc("/", handleLoginResponse(response))
	var err error

	go func() {
		err = server.ListenAndServe()
		if err != nil {
			close(response)
		}
	}()

	// wait for response
	resp := <-response
	if resp == nil {
		return "", "", err
	}

	return resp.code, resp.state, nil
}
