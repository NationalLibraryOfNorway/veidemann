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
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/metrics/internal/frontier"
	"github.com/NationalLibraryOfNorway/veidemann/metrics/internal/metrics"
	"github.com/NationalLibraryOfNorway/veidemann/metrics/internal/rethinkdb"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	name    = "metrics"
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

	db := rethinkdb.NewConnection(
		opts.DBHost(),
		opts.DBPort(),
		opts.DBUsername(),
		opts.DBPassword(),
		opts.DBName(),
		1*time.Minute)
	err = db.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to RethinkDB: %w", err)
	}
	defer func() { _ = db.Close() }()

	slog.Info("Connected to RethinkDB", "host", opts.DBHost(), "port", opts.DBPort())

	if err := db.Verify(); err != nil {
		_ = db.Close()
		return fmt.Errorf("database is not initialized: %w", err)
	}

	frontierAddress := fmt.Sprintf("%s:%d", opts.FrontierHost(), opts.FrontierPort())
	frontierConn, err := grpc.NewClient(frontierAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to create frontier client: %w", err)
	}
	defer func() { _ = frontierConn.Close() }()

	slog.Info("Frontier client created", "address", frontierAddress)

	mux := http.NewServeMux()
	mux.Handle(opts.MetricsPath(), promhttp.Handler())
	mux.Handle(opts.ReadyPath(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	server := &http.Server{
		Addr:    opts.Address(),
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	exporter := metrics.New(db, frontier.New(frontierConn), 30*time.Second)
	exporter.Start(ctx)
	defer exporter.Stop()

	go func() {
		err := server.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Metrics server stopped", "error", err)
		}
	}()

	slog.Info("Metrics server listening", "address", opts.Address())

	<-ctx.Done()

	slog.Info("Shutting down gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
