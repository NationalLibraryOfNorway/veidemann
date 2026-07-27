package url

import "testing"

func TestIsBrowserLocal(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "data", url: "data:image/png;base64,AAAA", want: true},
		{name: "blob", url: "blob:https://example.com/id", want: true},
		{name: "about", url: "about:blank", want: true},
		{name: "javascript", url: "javascript:void(0)", want: true},
		{name: "chrome", url: "chrome://settings", want: true},
		{name: "devtools", url: "devtools://devtools/bundled/", want: true},
		{name: "mixed case", url: "DaTa:text/plain,hello", want: true},
		{name: "empty", url: "", want: false},
		{name: "https", url: "https://example.com/image.png", want: false},
		{name: "http", url: "http://example.com/", want: false},
		{name: "data in path", url: "https://example.com/data:image.png", want: false},
		{name: "data in query", url: "https://example.com/?value=data:text/plain", want: false},
		{name: "scheme prefix only", url: "database:value", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsBrowserLocal(tc.url); got != tc.want {
				t.Fatalf("IsBrowserLocal(%q) = %v, want %v", tc.url, got, tc.want)
			}
		})
	}
}
