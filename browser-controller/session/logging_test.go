package session

import (
	"log/slog"
	"testing"

	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/requests"
)

func TestResourceFailureLogLevel(t *testing.T) {
	tests := []struct {
		name         string
		req          *requests.Request
		resourceType string
		want         slog.Level
	}{
		{name: "document before recorder registration", req: &requests.Request{ResourceType: "Document"}, want: slog.LevelWarn},
		{name: "blocking image", req: &requests.Request{GotNew: true, ResourceType: "Image"}, want: slog.LevelWarn},
		{name: "non-blocking fetch", req: &requests.Request{GotNew: true, ResourceType: "Fetch"}, want: slog.LevelDebug},
		{name: "unknown non-blocking request", resourceType: "Other", want: slog.LevelDebug},
		{name: "unknown document request", resourceType: "Document", want: slog.LevelWarn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resourceFailureLogLevel(tt.req, tt.resourceType); got != tt.want {
				t.Fatalf("resourceFailureLogLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChromedpMessageLogLevel(t *testing.T) {
	if got := chromedpMessageLogLevel("ERROR: unhandled node event *dom.EventAdoptedStyleSheetsModified"); got != slog.LevelDebug {
		t.Fatalf("known unsupported event level = %v, want DEBUG", got)
	}
	if got := chromedpMessageLogLevel("unexpected browser failure"); got != slog.LevelWarn {
		t.Fatalf("unexpected error level = %v, want WARN", got)
	}
}
