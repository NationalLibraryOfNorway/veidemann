package session

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

const LevelTrace = slog.Level(-8)

func chromedpMessageLogLevel(message string) slog.Level {
	if strings.Contains(message, "unhandled node event *dom.EventAdoptedStyleSheetsModified") {
		return LevelTrace
	}
	return slog.LevelWarn
}

func chromedpErrorf(log *slog.Logger) func(string, ...any) {
	return func(format string, args ...any) {
		message := fmt.Sprintf(format, args...)
		log.Log(context.Background(), chromedpMessageLogLevel(message), message)
	}
}
