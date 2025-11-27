/*
 * Copyright 2025 National Library of Norway.
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
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	robotsevaluatorV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator/v1"
	"github.com/NationalLibraryOfNorway/veidemann/robots-evaluator/cache"
	"github.com/NationalLibraryOfNorway/veidemann/robots-evaluator/robots"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

const name = "robots-evaluator"

var (
	version = ""
	commit  = ""
	date    = ""
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err := run(ctx)
	slog.Error("Stopped "+name, "error", err)

	os.Exit(1)
}

type Config struct {
	Host         string
	Port         int
	MetricsPort  int
	LogLevel     string
	LogFile      string
	OlricAddress []string
}

func run(ctx context.Context) error {
	pflag.String("log-level", "info", "Log level, available levels are: error, warn, info and debug")
	pflag.String("addr", ":8090", "Address for the gRPC server")
	pflag.String("metrics-addr", ":9153", "Address for the metrics server")
	pflag.StringSlice("olric-address", []string{"localhost:3320"}, "Olric address")
	pflag.String("olric-dmap", "robots-evaluator", "Olric DMap name")
	pflag.Parse()

	_ = viper.BindPFlags(pflag.CommandLine)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	level := viper.GetString("log-level")
	addr := viper.GetString("addr")
	metricsAddr := viper.GetString("metrics-addr")
	metricsPath := "/metrics"
	olricAddress := viper.GetStringSlice("olric-address")
	olricDmap := viper.GetString("olric-dmap")

	initLogger(os.Stderr, level)

	slog.Info("Starting "+name, "version", version, "commit", commit, "date", date)

	cache, err := cache.NewOlricCache(olricAddress, olricDmap)
	if err != nil {
		return fmt.Errorf("failed to create Olric cache: %w", err)
	}

	defer func() { _ = cache.Close(context.Background()) }()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	s := &robots.EvaluatorServer{
		Evaluator: &robots.Evaluator{
			Cache:  cache,
			Client: client,
		},
	}

	server := grpc.NewServer()
	robotsevaluatorV1.RegisterRobotsEvaluatorServer(server, s)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		slog.Info("Starting metrics server", "address", metricsAddr, "path", metricsPath)
		err := runMetricsServer(ctx, metricsAddr, metricsPath)
		slog.Error("Metrics server stopped", "error", err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		slog.Info("Starting gRPC server", "address", addr)
		err := server.Serve(listener)
		slog.Error("gRPC server stopped", "error", err)
	}()

	<-ctx.Done()          // Wait for context cancellation
	server.GracefulStop() // Stop the gRPC server gracefully
	wg.Wait()             // Wait for all goroutines to finish

	return nil
}

func runMetricsServer(ctx context.Context, addr string, path string) error {
	http.Handle(path, promhttp.Handler())

	server := &http.Server{Addr: addr}

	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}

func initLogger(w io.Writer, level string) {
	levelVar := new(slog.LevelVar)
	levelVar.Set(toLogLevel(level))
	opts := &slog.HandlerOptions{Level: levelVar}
	handler := slog.NewJSONHandler(w, opts)
	slog.SetDefault(slog.New(handler))
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
