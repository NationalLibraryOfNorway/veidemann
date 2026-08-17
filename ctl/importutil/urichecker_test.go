package importutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUriChecker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("method = %q, want %q", r.Method, http.MethodHead)
		}

		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(http.StatusOK)
		case "/moved":
			w.Header().Set("Location", "/destination")
			w.WriteHeader(http.StatusMovedPermanently)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	uriChecker := &UriChecker{
		Client: NewHttpClient(5*time.Second, false),
	}

	tests := []struct {
		name string
		uri  string
		want string
	}{
		{
			name: "reachable URI",
			uri:  server.URL + "/ok",
			want: server.URL + "/ok",
		},
		{
			name: "permanent redirect",
			uri:  server.URL + "/moved",
			want: server.URL + "/destination",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := uriChecker.Check(tt.uri)
			if err != nil {
				t.Fatal(err)
			}

			if got != tt.want {
				t.Errorf("Check(%q) = %q, want %q", tt.uri, got, tt.want)
			}
		})
	}
}
