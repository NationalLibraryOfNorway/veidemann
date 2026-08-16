package session

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/requests"
)

func resourceFailureLogLevel(resourceType string) slog.Level {
	req := requests.Request{GotNew: true, ResourceType: resourceType}
	if req.BlocksPageCompletion() {
		return slog.LevelWarn
	}
	return slog.LevelDebug
}

func chromedpMessageLogLevel(message string) slog.Level {
	if strings.Contains(message, "unhandled node event *dom.EventAdoptedStyleSheetsModified") {
		return slog.LevelDebug
	}
	return slog.LevelWarn
}

func chromedpErrorf(log *slog.Logger) func(string, ...any) {
	return func(format string, args ...any) {
		message := fmt.Sprintf(format, args...)
		log.Log(context.Background(), chromedpMessageLogLevel(message), message)
	}
}
