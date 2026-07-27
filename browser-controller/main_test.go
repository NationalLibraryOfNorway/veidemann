package main

import (
	"log/slog"
	"testing"
)

func TestToLogLevelIsCaseInsensitive(t *testing.T) {
	for input, want := range map[string]slog.Level{
		"DEBUG": slog.LevelDebug,
		"INFO":  slog.LevelInfo,
		"WARN":  slog.LevelWarn,
		"ERROR": slog.LevelError,
	} {
		if got := toLogLevel(input); got != want {
			t.Errorf("toLogLevel(%q) = %v, want %v", input, got, want)
		}
	}
}
