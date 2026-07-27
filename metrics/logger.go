package main

import (
	"io"
	"log"
	"log/slog"
)

func initLogger(w io.Writer, level string, source bool) {
	levelVar := new(slog.LevelVar)
	levelVar.Set(toLogLevel(level))

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		AddSource: source,
		Level:     levelVar,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Redirect package-level log.Print/log.Printf/etc. to the same slog handler.
	log.SetOutput(slog.NewLogLogger(handler, slog.LevelInfo).Writer())
	log.SetFlags(0)
}

func toLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
