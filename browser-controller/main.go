/*
 * Copyright 2020 National Library of Norway.
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
	"sync/atomic"
	"syscall"
	"time"

	browserControllerV2 "github.com/NationalLibraryOfNorway/veidemann/api/browsercontroller/v2"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/controller"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/database"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/frontier"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/logwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/robotsevaluator"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/screenshotwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/server"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/serviceconnections"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/uber/jaeger-client-go/config"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

var (
	name    = "browser-controller"
	version = ""
	commit  = ""
	date    = ""
)

const readyPath = "/readyz"

func main() {
	err := run()
	if err != nil {
		slog.Error("Bye", "error", err)
		os.Exit(1)
	}
	slog.Info("Goodbye!")
}

func run() error {
	opts, err := parseFlags()
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if opts.Help() {
		pflag.Usage()
		return nil
	}

	// init logger
	initLogger(
		os.Stdout,
		opts.LogLevel(),
		opts.LogMethod(),
	)

	slog.Info(name, "version", version, "commit", commit, "date", date)

	closer := initTracer(name)
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	screenshotWriter := screenshotwriter.New(
		serviceconnections.WithHost(opts.ContentWriterHost()),
		serviceconnections.WithPort(opts.ContentWriterPort()),
	)
	err = screenshotWriter.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to content writer: %w", err)
	}
	defer func() { _ = screenshotWriter.Close() }()

	tracer := opentracing.GlobalTracer()
	frontier := frontier.New(
		serviceconnections.WithHost(opts.FrontierHost()),
		serviceconnections.WithPort(opts.FrontierPort()),
		serviceconnections.WithDialOptions(
			grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(tracer)),
		),
	)
	err = frontier.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to frontier: %w", err)
	}
	defer func() { _ = frontier.Close() }()

	robotsEvaluator := robotsevaluator.New(
		serviceconnections.WithHost(opts.RobotsEvaluatorHost()),
		serviceconnections.WithPort(opts.RobotsEvaluatorPort()),
	)
	err = robotsEvaluator.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to robots evaluator: %w", err)
	}
	defer func() { _ = robotsEvaluator.Close() }()

	logWriter := logwriter.New(
		serviceconnections.WithHost(opts.LogServiceHost()),
		serviceconnections.WithPort(opts.LogServicePort()),
	)
	err = logWriter.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to log service: %w", err)
	}
	defer func() { _ = logWriter.Close() }()

	db := database.NewRethinkDbConnection(
		database.Options{
			Address:            fmt.Sprintf("%s:%d", opts.DBHost(), opts.DBPort()),
			Username:           opts.DBUser(),
			Password:           opts.DBPassword(),
			Database:           opts.DBName(),
			QueryTimeout:       opts.DBQueryTimeout(),
			MaxOpenConnections: opts.DBMaxOpenConn(),
			MaxRetries:         opts.DBMaxRetries(),
			UseOpenTracing:     opts.DBUseOpentracing(),
		},
	)
	err = db.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() { _ = db.Close() }()

	configAdapter := database.NewConfigAdapter(db)

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, ctx := errgroup.WithContext(rootCtx)

	var ready atomic.Bool

	mux := http.NewServeMux()
	mux.HandleFunc(readyPath, func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle(opts.MetricsPath(), promhttp.Handler())

	sessions := session.NewRegistry(
		opts.ProxyCount(),
		session.WithLogWriter(logWriter),
		session.WithScreenshotWriter(screenshotWriter),
		session.WithBrowserHost(opts.BrowserHost()),
		session.WithBrowserPort(opts.BrowserPort()),
		session.WithProxyHost(opts.ProxyHost()),
		session.WithProxyPort(opts.ProxyPort()),
		session.WithConfigAdapter(configAdapter),
	)
	defer sessions.Close()

	telemetryAddr := fmt.Sprintf("%s:%d", opts.MetricsInterface(), opts.MetricsPort())
	telemetryListener, err := net.Listen("tcp", telemetryAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on telemetry address %s: %w", telemetryAddr, err)
	}
	defer func() { _ = telemetryListener.Close() }()

	slog.Info("Telemetry server listening", "address", telemetryAddr)

	telemetryServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", opts.MetricsInterface(), opts.MetricsPort()),
		Handler: mux,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	// Start telemetry server
	g.Go(func() error {
		err := telemetryServer.Serve(telemetryListener)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	grpcAddr := fmt.Sprintf("%s:%d", opts.Interface(), opts.Port())
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", grpcAddr, err)
	}
	defer func() { _ = listener.Close() }()

	slog.Info("gRPC server listening", "address", grpcAddr)

	grpcServer := grpc.NewServer()
	serverImpl := server.NewApiServer(sessions, robotsEvaluator, logWriter)
	browserControllerV2.RegisterBrowserControllerServer(grpcServer, serverImpl)

	// Start gRPC server
	g.Go(func() error {
		err := grpcServer.Serve(listener)
		if errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return err
	})

	mainLoop := controller.New(sessions, frontier, opts.FetchTimeout(), opts.ReportTimeout())

	// Browser controller main loop
	g.Go(func() error {
		err := mainLoop.Run(ctx)

		if errors.Is(err, context.Canceled) {
			return nil
		}

		return err
	})

	ready.Store(true)
	slog.Info("Server ready")

	<-ctx.Done()

	ready.Store(false)
	slog.Info("Server shutting down")

	grpcServer.GracefulStop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

// initTracer initializes the global OpenTracing tracer using Jaeger configuration from environment variables.
func initTracer(service string) io.Closer {
	cfg, err := config.FromEnv()
	if err != nil {
		return nil
	}

	if cfg.ServiceName == "" {
		cfg.ServiceName = service
	}

	tracer, closer, err := cfg.NewTracer()
	if err == nil {
		opentracing.SetGlobalTracer(tracer)
	}

	return closer
}
