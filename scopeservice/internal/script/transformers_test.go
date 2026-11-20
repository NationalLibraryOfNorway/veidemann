package script

import (
	"testing"

	"github.com/nationallibraryofnorway/whatwg-url/url"
	"go.starlark.net/starlark"
)

func TestRemoveQueryPreservesSerializedTerms(t *testing.T) {
	tests := []struct {
		name, query, remove, want string
	}{
		{"plus spaces and escapes", "?drop=1&q=a+b%2Bc&space=%20&tilde=~&slash=%2f", "drop", "?q=a+b%2Bc&space=%20&tilde=~&slash=%2f"},
		{"empty values and duplicates", "?drop=1&flag=&flag&x=", "drop", "?flag&flag&x"},
		{"missing parameter still normalizes empty values", "?flag=&x=0", "missing", "?flag&x=0"},
		{"nonempty values retain equals", "?drop=1&a=0&b=%20&c=%3D&=value", "drop", "?a=0&b=%20&c=%3D&=value"},
		{"malformed UTF-8 values", "?drop=1&q=%FC%FF", "drop", "?q=%FC%FF"},
		{"malformed escapes", "?drop=1&q=%GG%2", "drop", "?q=%GG%2"},
		{"decoded names and duplicates", "?%64rop=1&keep=2&drop=3", "drop", "?keep=2"},
		{"encoded surviving names", "?drop=1&%6beep=%2f&keep=2", "drop", "?%6beep=%2f&keep=2"},
		{"space in names", "?a+b=1&a%20b=2&a%2Bb=3", "a b", "?a%2Bb=3"},
		{"literal plus in names", "?a+b=1&a%2Bb=2", "a+b", "?a+b=1"},
		{"names are decoded once", "?%2564rop=1&%64rop=2", "drop", "?%2564rop=1"},
		{"malformed UTF-8 name matching", "?%FC%FF=1&%EF%BF%BD%EF%BF%BD=2&keep=3", "\uFFFD\uFFFD", "?keep=3"},
		{"BOM in names", "?%EF%BB%BFdrop=1&drop=2", "drop", "?%EF%BB%BFdrop=1"},
		{"empty separator terms", "?&drop=1&&keep=2&", "drop", "?&&keep=2&"},
		{"empty name excludes separator terms", "?&=value&&=", "", "?&"},
		{"remaining empty term preserves query", "?drop=1&", "drop", "?"},
		{"last parameter removes query marker", "?drop=1", "drop", ""},
		{"absent query", "", "drop", ""},
		{"present empty query", "?", "drop", "?"},
		{"empty name in empty query", "?", "", "?"},
		{"fragment preserved", "?drop=1&keep=%20#part", "drop", "?keep=%20#part"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := url.Parse("https://example.com/" + tt.query)
			if err != nil {
				t.Fatal(err)
			}
			thread := &starlark.Thread{Name: t.Name()}
			thread.SetLocal(urlKey, &UrlValue{parsedUri: parsed})
			_, err = starlark.Call(thread, starlark.Universe["removeQuery"], starlark.Tuple{starlark.String(tt.remove)}, nil)
			if err != nil {
				t.Fatal(err)
			}
			want := "https://example.com/" + tt.want
			if parsed.String() != want {
				t.Fatalf("removeQuery(%q) = %q, want %q", tt.remove, parsed.String(), want)
			}
		})
	}
}
