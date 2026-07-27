/*
 * Copyright 2018 National Library of Norway.
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
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	ooshandlerV1 "github.com/NationalLibraryOfNorway/veidemann/api/ooshandler/v1"
	"github.com/NationalLibraryOfNorway/veidemann/ooshandler/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

var (
	name    = "ooshandler"
	version = ""
	commit  = ""
	date    = ""
)

func main() {
	err := run()
	if err != nil {
		slog.Error("Bye!", "error", err)
		os.Exit(1)
	}
	slog.Info("Goodbye!")
}

func run() error {
	err := parseFlags()
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	opts := Options{}

	initLogger(os.Stdout, opts.LogLevel(), opts.LogMethod())

	slog.Info("Service version", "name", name, "version", version, "commit", commit, "date", date)

	dataDir, err := filepath.Abs(opts.DataDir())
	if err != nil {
		return fmt.Errorf("unable to resolve data directory: %w", err)
	}

	err = os.MkdirAll(dataDir, 0777)
	if err != nil {
		return fmt.Errorf("unable to create data directory: %w", err)
	}

	oosHandler, err := service.NewHandler(dataDir)
	if err != nil {
		return fmt.Errorf("unable to create OOS handler: %w", err)
	}

	oosService := service.NewOutOfScopeHandler(oosHandler)
	grpcServer := grpc.NewServer()
	ooshandlerV1.RegisterOosHandlerServer(grpcServer, oosService)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, groupCtx := errgroup.WithContext(ctx)

	mux := http.NewServeMux()
	mux.Handle(opts.MetricsPath(), promhttp.Handler())
	mux.Handle(opts.ReadyPath(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	telemetryServer := &http.Server{
		Addr:    opts.TelemetryAddr(),
		Handler: mux,
	}

	g.Go(func() error {
		err := telemetryServer.ListenAndServe()
		slog.Warn("Telemetry server stopped", "error", err)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	slog.Info("Telemetry server listening", "address", opts.TelemetryAddr())

	addr := opts.Address()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	g.Go(func() error { return grpcServer.Serve(listener) })

	slog.Info("gRPC server listening", "address", addr)

	<-groupCtx.Done()

	slog.Info("Shutting down gracefully")

	grpcServer.GracefulStop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = telemetryServer.Shutdown(shutdownCtx)

	return g.Wait()
}

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
