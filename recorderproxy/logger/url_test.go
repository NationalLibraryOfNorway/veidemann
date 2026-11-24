package logger

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestDiagnosticURLBoundsAndOmitsData(t *testing.T) {
	for _, raw := range []string{"", "https://example.com/path?query=value", "https://example.com/" + strings.Repeat("æ", 1000), "data:image/png;base64,PRIVATE_PAYLOAD"} {
		got := DiagnosticURL(raw)
		if len(got) > 512 || !utf8.ValidString(got) || strings.Contains(got, "PRIVATE_PAYLOAD") {
			t.Fatal(got)
		}
		if len(raw) <= 512 && !strings.HasPrefix(raw, "data:") && got != raw {
			t.Fatal("changed short diagnostic URL")
		}
	}
}
