package url

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"IDN", "https://BÜCHER.example/", "https://xn--bcher-kva.example/"},
		{"default HTTP port", "http://Example.COM:80", "http://example.com/"},
		{"default HTTPS port", "https://Example.COM:443/a", "https://example.com/a"},
		{"nondefault port", "https://example.com:8443/a", "https://example.com:8443/a"},
		{"escaped path", "https://example.com/a%2Fb/%2520?q=a+b%2Bc", "https://example.com/a%2Fb/%2520?q=a+b%2Bc"},
		{"dot segments", "https://example.com/a/../b", "https://example.com/b"},
		{"invalid host fallback", "http://[invalid]/", "http://[invalid]/"},
		{"invalid port fallback", "https://example.com:99999/", "https://example.com:99999/"},
		{"relative fallback", "/relative/path", "/relative/path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Normalize(tt.input); got != tt.want {
				t.Fatalf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

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
