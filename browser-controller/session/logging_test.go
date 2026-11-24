package session

import (
	"log/slog"
	"testing"
)

func TestChromedpMessageLogLevel(t *testing.T) {
	if got := chromedpMessageLogLevel("ERROR: unhandled node event *dom.EventAdoptedStyleSheetsModified"); got != LevelTrace {
		t.Fatalf("known unsupported event level = %v, want TRACE", got)
	}
	if got := chromedpMessageLogLevel("unexpected browser failure"); got != slog.LevelWarn {
		t.Fatalf("unexpected error level = %v, want WARN", got)
	}
}
