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
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	robotsevaluatorV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator/v1"
	"github.com/NationalLibraryOfNorway/veidemann/robots-evaluator/cache"
	"github.com/NationalLibraryOfNorway/veidemann/robots-evaluator/robots"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

const name = "robots-evaluator"

var (
	version = ""
	commit  = ""
	date    = ""
)

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

func main() {
	pflag.String("log-level", "info", "error, warn, info or debug")
	pflag.String("addr", ":8090", "Address for the gRPC server")
	pflag.String("telemetry-addr", ":9153", "Address for the telemetry server")
	pflag.StringSlice("olric-address", []string{"localhost:3320"}, "Olric address")
	pflag.String("olric-dmap", "robots-evaluator", "Olric DMap name")
	pflag.Parse()

	_ = viper.BindPFlags(pflag.CommandLine)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	level := viper.GetString("log-level")
	addr := viper.GetString("addr")
	telemetryAddr := viper.GetString("telemetry-addr")
	olricAddress := viper.GetStringSlice("olric-address")
	olricDmap := viper.GetString("olric-dmap")

	initLogger(os.Stderr, level)

	app := &App{
		Addr:          addr,
		TelemetryAddr: telemetryAddr,
		OlricAddr:     olricAddress,
		OlricDmap:     olricDmap,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	slog.Info(name, "version", version, "commit", commit, "date", date)

	if err := app.Run(ctx); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

type App struct {
	Addr          string
	TelemetryAddr string
	OlricAddr     []string
	OlricDmap     string

	ready   atomic.Bool
	cachier cache.Cachier
}

func (app *App) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	mux := http.NewServeMux()
	mux.Handle(readyPath, app)

	telemetry := &http.Server{
		Addr:    app.TelemetryAddr,
		Handler: mux,
	}

	g.Go(func() error {
		err := telemetry.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	err := app.init(ctx)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()

	impl := &robots.EvaluatorServer{
		Evaluator: &robots.Evaluator{
			Cache: app.cachier,
			Client: &http.Client{
				Timeout: 10 * time.Second,
				Transport: &http.Transport{
					Proxy: http.ProxyFromEnvironment,
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			},
		},
	}
	robotsevaluatorV1.RegisterRobotsEvaluatorServer(grpcServer, impl)

	listener, err := net.Listen("tcp", app.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", app.Addr, err)
	}

	slog.Info("gRPC server listening", "address", app.Addr)

	g.Go(func() error { return grpcServer.Serve(listener) })

	<-ctx.Done()

	grpcServer.GracefulStop()
	app.ready.Store(false)

	_ = listener.Close()
	_ = app.cachier.Close(context.Background())
	_ = telemetry.Shutdown(context.Background())

	return g.Wait()
}

const readyPath = "/readyz"

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !app.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready\n"))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (app *App) init(ctx context.Context) error {
	init := new(errgroup.Group)

	init.Go(func() error {
		cachier, err := app.newOlricCache(ctx)
		if err != nil {
			return err
		}
		app.cachier = cachier
		app.ready.Store(true)

		return nil
	})

	return init.Wait()
}

func (app *App) newOlricCache(ctx context.Context) (cache.Cachier, error) {
	backoff := time.Second
	timer := time.NewTimer(backoff)
	const maxBackoff = 30 * time.Second

	for {
		c, err := cache.NewOlricCache(app.OlricAddr, app.OlricDmap)
		if err == nil {
			slog.Info("Connected to olric", "address", app.OlricAddr)
			return c, nil
		}

		slog.Warn("Connection failed, retrying...", "error", err, "backoff", backoff.String())

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
		if backoff < maxBackoff {
			backoff *= 2
		}
		timer.Reset(backoff)
	}
}
