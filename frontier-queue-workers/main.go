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

package main

import (
	"context"
	"fmt"
	"io"
	stdlog "log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/frontier-queue-workers/app"
	"github.com/NationalLibraryOfNorway/veidemann/frontier-queue-workers/database"
	"github.com/opentracing/opentracing-go"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

func main() {
	pflag.String("db-host", "rethinkdb-proxy", "RethinkDB host")
	pflag.Int("db-port", 28015, "RethinkDB port")
	pflag.String("db-name", "veidemann", "RethinkDB database name")
	pflag.String("db-user", "", "RethinkDB username")
	pflag.String("db-password", "", "RethinkDB password")
	pflag.Duration("db-query-timeout", 10*time.Second, "RethinkDB query timeout")
	pflag.Int("db-max-retries", 3, "Max retries when query fails")
	pflag.Int("db-max-open-conn", 10, "Max open connections")
	pflag.Bool("db-use-opentracing", false, "Use opentracing for queries")

	pflag.String("redis-host", "redis-veidemann-frontier-master", "Redis host")
	pflag.Int("redis-port", 6379, "Redis port")
	pflag.String("redis-password", "", "Redis password")
	pflag.String("redis-sentinel-master-name", "", "Redis Sentinel master name")

	pflag.String("telemetry-address", ":9153", "Address for telemetry endpoint")

	pflag.String("log-level", "info", "log level, available levels are panic, fatal, error, warn, info, debug and trace")
	pflag.String("log-formatter", "logfmt", "log formatter, available values are logfmt and json")
	pflag.Bool("log-method", false, "log method names")

	pflag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err := run(ctx)
	if err != nil {
		slog.Error("Bye bye", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()

	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		return fmt.Errorf("failed to bind command line flags: %w", err)
	}

	initLog(viper.GetString("log-level"), viper.GetString("log-formatter"), viper.GetBool("log-method"))

	if tracer, closer := initTracer("frontier-queue-workers", newJaegerLogger()); tracer != nil {
		defer func() { _ = closer.Close() }()
		opentracing.SetGlobalTracer(tracer)
	}

	redisAddr := fmt.Sprintf("%s:%d", viper.GetString("redis-host"), viper.GetInt("redis-port"))
	redisOpts := &redis.UniversalOptions{
		Addrs:            []string{redisAddr},
		MasterName:       viper.GetString("redis-sentinel-master-name"),
		SentinelPassword: viper.GetString("redis-password"),
		Password:         viper.GetString("redis-password"),
		MaxRetries:       3,
	}

	rethinkdbOpts := database.RethinkDbOptions{
		Address:            fmt.Sprintf("%s:%d", viper.GetString("db-host"), viper.GetInt("db-port")),
		Username:           viper.GetString("db-user"),
		Password:           viper.GetString("db-password"),
		Database:           viper.GetString("db-name"),
		QueryTimeout:       viper.GetDuration("db-query-timeout"),
		MaxOpenConnections: viper.GetInt("db-max-open-conn"),
		MaxRetries:         viper.GetInt("db-max-retries"),
		UseOpenTracing:     viper.GetBool("db-use-opentracing"),
	}

	app := &app.App{
		DbOptions:     rethinkdbOpts,
		RedisOptions:  redisOpts,
		TelemetryAddr: viper.GetString("telemetry-address"),
	}

	return app.Run(ctx)
}

func initLog(level string, format string, logCaller bool) {
	handlerOptions := &slog.HandlerOptions{AddSource: logCaller, Level: parseLogLevel(level)}
	var handler slog.Handler
	if strings.EqualFold(format, "json") {
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
		return slog.LevelInfo
	}
}

// jaegerLogger implements the jaeger.Logger interface using slog.
type jaegerLogger struct {
	impl *slog.Logger
}

func newJaegerLogger() jaeger.Logger {
	return &jaegerLogger{
		impl: slog.Default().With("component", "jaeger"),
	}
}

func (j jaegerLogger) Error(msg string) {
	j.impl.Error(msg)
}

func (j *jaegerLogger) Infof(msg string, args ...interface{}) {
	j.impl.Info(fmt.Sprintf(msg, args...))
}

// InitTracer returns an instance of opentracing.Tracer and io.Closer.
func initTracer(service string, logger jaeger.Logger) (opentracing.Tracer, io.Closer) {
	cfg, err := config.FromEnv()
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = service
	}

	tracer, closer, err := cfg.NewTracer(config.Logger(logger))
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}
	return tracer, closer
}
