/*
 * Copyright 2021 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package logger

import (
	stdlog "log"
	"log/slog"
	"os"
	"strings"
)

// Log formats
const (
	formatJson   = "json"
	formatPretty = "pretty"
)

// InitLogger initializes the logger with the given level and format.
// If logCaller is true, the caller is logged.
func InitLogger(level string, format string, logCaller bool) {
	handlerOptions := &slog.HandlerOptions{AddSource: logCaller, Level: parseLogLevel(level)}
	var handler slog.Handler
	if strings.ToLower(format) == formatJson {
		handler = slog.NewJSONHandler(os.Stderr, handlerOptions)
	} else {
		handler = slog.NewTextHandler(os.Stderr, handlerOptions)
	}
	slog.SetDefault(slog.New(handler))

	stdlog.SetFlags(0)

	slog.Info("Setting log level", "level", strings.ToLower(level))
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "panic", "fatal", "error":
		return slog.LevelError
	case "warn":
		return slog.LevelWarn
	case "info", "debug":
		return slog.LevelInfo
	case "trace":
		return slog.LevelDebug
	default:
		return slog.Level(100)
	}
}
