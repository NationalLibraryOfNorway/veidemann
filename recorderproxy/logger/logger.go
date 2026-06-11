/*
 * Copyright 2019 National Library of Norway.
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
	"context"
	"fmt"
	stdLog "log"
	"log/slog"
	"os"
	"strings"
)

const (
	FORMATTER_TEXT   = "text"
	FORMATTER_JSON   = "json"
	FORMATTER_LOGFMT = "logfmt"
)

type Fields map[string]any

type Level = slog.Level

const (
	TraceLevel Level = slog.Level(-8)
	DebugLevel Level = slog.LevelDebug
	InfoLevel  Level = slog.LevelInfo
	WarnLevel  Level = slog.LevelWarn
	ErrorLevel Level = slog.LevelError
)

var (
	currentLevel slog.LevelVar
	Log          *Logger
)

func init() {
	currentLevel.Set(InfoLevel)
	handler, err := newHandler(FORMATTER_TEXT, false)
	if err != nil {
		panic(err)
	}
	setDefaultLogger(handler)
}

func StandardLogger() *Logger {
	return Log
}

func InitLog(level, formatter string, logMethod bool) error {
	logLevel, err := ParseLevel(level)
	if err != nil {
		return fmt.Errorf("failed to parse log level: %q", level)
	}
	currentLevel.Set(logLevel)
	handler, err := newHandler(formatter, logMethod)
	if err != nil {
		return err
	}
	setDefaultLogger(handler)
	return nil
}

func ParseLevel(level string) (Level, error) {
	switch strings.ToLower(level) {
	case "trace":
		return TraceLevel, nil
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "error", "fatal", "panic":
		return ErrorLevel, nil
	default:
		return InfoLevel, fmt.Errorf("unknown level %q", level)
	}
}

func SetLevel(level Level) {
	currentLevel.Set(level)
}

func IsLevelEnabled(level Level) bool {
	return level >= currentLevel.Level()
}

func newHandler(formatter string, addSource bool) (slog.Handler, error) {
	options := &slog.HandlerOptions{
		AddSource: addSource,
		Level:     &currentLevel,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.LevelKey {
				if level, ok := attr.Value.Any().(slog.Level); ok {
					attr.Value = slog.StringValue(levelName(level))
				}
			}
			return attr
		},
	}

	switch strings.ToLower(formatter) {
	case FORMATTER_TEXT:
	case FORMATTER_LOGFMT:
	case FORMATTER_JSON:
		return slog.NewJSONHandler(os.Stderr, options), nil
	default:
		return nil, fmt.Errorf("unknown formatter type: %q", formatter)
	}
	return slog.NewTextHandler(os.Stderr, options), nil
}

func levelName(level slog.Level) string {
	switch {
	case level <= TraceLevel:
		return "TRACE"
	case level <= DebugLevel:
		return "DEBUG"
	case level <= InfoLevel:
		return "INFO"
	case level <= WarnLevel:
		return "WARN"
	default:
		return "ERROR"
	}
}

func setDefaultLogger(handler slog.Handler) {
	logger := slog.New(handler)
	Log = &Logger{logger: logger}
	slog.SetDefault(logger)

	stdLogger := slog.NewLogLogger(handler, InfoLevel)
	stdLog.SetFlags(0)
	stdLog.SetOutput(stdLogger.Writer())
}

type Logger struct {
	logger *slog.Logger
}

func (l *Logger) base() *slog.Logger {
	if l == nil || l.logger == nil {
		return slog.Default()
	}
	return l.logger
}

func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{logger: l.base().With(key, value)}
}

func (l *Logger) WithFields(fields Fields) *Logger {
	args := make([]any, 0, len(fields)*2)
	for key, value := range fields {
		args = append(args, key, value)
	}
	return &Logger{logger: l.base().With(args...)}
}

func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return l.WithField("error", err)
}

func (l *Logger) WithComponent(comp string) *Logger {
	return l.WithField("component", comp)
}

func (l *Logger) log(level Level, msg string) {
	l.base().Log(context.Background(), level, msg)
}

func (l *Logger) Trace(args ...interface{}) {
	l.log(TraceLevel, fmt.Sprint(args...))
}

func (l *Logger) Traceln(args ...interface{}) {
	l.log(TraceLevel, strings.TrimSuffix(fmt.Sprintln(args...), "\n"))
}

func (l *Logger) Tracef(format string, args ...interface{}) {
	l.log(TraceLevel, fmt.Sprintf(format, args...))
}

func (l *Logger) Debug(args ...interface{}) {
	l.log(DebugLevel, fmt.Sprint(args...))
}

func (l *Logger) Debugln(args ...interface{}) {
	l.log(DebugLevel, strings.TrimSuffix(fmt.Sprintln(args...), "\n"))
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DebugLevel, fmt.Sprintf(format, args...))
}

func (l *Logger) Info(args ...interface{}) {
	l.log(InfoLevel, fmt.Sprint(args...))
}

func (l *Logger) Infoln(args ...interface{}) {
	l.log(InfoLevel, strings.TrimSuffix(fmt.Sprintln(args...), "\n"))
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(InfoLevel, fmt.Sprintf(format, args...))
}

func (l *Logger) Warn(args ...interface{}) {
	l.log(WarnLevel, fmt.Sprint(args...))
}

func (l *Logger) Warnln(args ...interface{}) {
	l.log(WarnLevel, strings.TrimSuffix(fmt.Sprintln(args...), "\n"))
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WarnLevel, fmt.Sprintf(format, args...))
}

func (l *Logger) Error(args ...interface{}) {
	l.log(ErrorLevel, fmt.Sprint(args...))
}

func (l *Logger) Errorln(args ...interface{}) {
	l.log(ErrorLevel, strings.TrimSuffix(fmt.Sprintln(args...), "\n"))
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ErrorLevel, fmt.Sprintf(format, args...))
}

func (l *Logger) Print(args ...interface{}) {
	l.Info(args...)
}

func (l *Logger) Println(args ...interface{}) {
	l.Infoln(args...)
}

func (l *Logger) Printf(format string, args ...interface{}) {
	l.Infof(format, args...)
}

func (l *Logger) Fatal(args ...interface{}) {
	msg := fmt.Sprint(args...)
	l.log(ErrorLevel, msg)
	os.Exit(1)
}

func (l *Logger) Fatalln(args ...interface{}) {
	msg := strings.TrimSuffix(fmt.Sprintln(args...), "\n")
	l.log(ErrorLevel, msg)
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(ErrorLevel, fmt.Sprintf(format, args...))
	os.Exit(1)
}

func (l *Logger) Panic(args ...interface{}) {
	msg := fmt.Sprint(args...)
	l.log(ErrorLevel, msg)
	panic(msg)
}

func (l *Logger) Panicln(args ...interface{}) {
	msg := strings.TrimSuffix(fmt.Sprintln(args...), "\n")
	l.log(ErrorLevel, msg)
	panic(msg)
}

func (l *Logger) Panicf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.log(ErrorLevel, msg)
	panic(msg)
}

func LogWithComponent(comp string) *Logger {
	return Log.WithField("component", comp)
}
