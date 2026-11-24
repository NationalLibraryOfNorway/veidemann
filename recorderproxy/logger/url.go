package logger

import (
	"strings"
	"unicode/utf8"
)

// DiagnosticURL bounds log fields without changing the request or recorded URI.
func DiagnosticURL(raw string) string {
	if strings.EqualFold(raw[:min(5, len(raw))], "data:") {
		return "data:[omitted]"
	}
	if len(raw) <= 512 {
		return raw
	}
	end := 512 - len("…")
	for end > 0 && !utf8.RuneStart(raw[end]) {
		end--
	}
	return raw[:end] + "…"
}
