package main

import (
	"bytes"
	"encoding/json"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"log/slog"
	"testing"
)

func TestToLogLevelIsCaseInsensitive(t *testing.T) {
	for input, want := range map[string]slog.Level{
		"TRACE": session.LevelTrace,
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

func TestTraceLogsHaveReadableLevelAndServiceContext(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
	var buf bytes.Buffer
	initLogger(&buf, "TRACE", false)
	serviceLogger().Log(t.Context(), session.LevelTrace, "Listener ready", "address", "[::]:9301")
	var row map[string]any
	if err := json.Unmarshal(buf.Bytes(), &row); err != nil {
		t.Fatal(err)
	}
	if row["level"] != "TRACE" || row["component"] != "browser-controller" {
		t.Fatal(row)
	}
}
