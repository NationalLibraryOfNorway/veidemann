package importutil

import (
	"testing"
	"time"
)

func TestUriChecker(t *testing.T) {

	uriChecker := &UriChecker{
		Client: NewHttpClient(5*time.Second, false),
	}

	tests := []struct {
		uri  string
		want string
	}{
		{"https://www.nb.no/", "https://www.nb.no/"},
		{"https://lokalhistoriewiki.no", "https://lokalhistoriewiki.no/wiki/Lokalhistoriewiki:Hovedside"},
	}

	for _, tt := range tests {
		got, err := uriChecker.Check(tt.uri)
		if err != nil {
			t.Error(err)
		}

		if got != tt.want {
			t.Errorf("Want %s, got %s", tt.want, got)
		}
	}
}
