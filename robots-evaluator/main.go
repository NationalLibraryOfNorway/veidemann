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
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	writerV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	dnsV1 "github.com/NationalLibraryOfNorway/veidemann/api/dnsresolver/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	robotsevaluatorV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator/v1"
	"github.com/NationalLibraryOfNorway/veidemann/robots-evaluator/cache"
	"github.com/NationalLibraryOfNorway/veidemann/robots-evaluator/robots"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	name    = "robots-evaluator"
	version = ""
	commit  = ""
	date    = ""
)

const (
	defaultRobotsCacheFreshness           = 24 * time.Hour
	defaultRobotsUnreachableRetryInterval = time.Hour
)

func initLogger(w io.Writer, level string) {
	levelVar := new(slog.LevelVar)
	levelVar.Set(toLogLevel(level))

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: levelVar,
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

func parseFlags() error {
	flags := pflag.CommandLine

	flags.String("dns-resolver-address", "dns-resolver:8053", "DNS resolver gRPC endpoint")
	flags.String("content-writer-address", "contentwriter:8082", "ContentWriter gRPC endpoint")
	flags.String("log-service-address", "log-service-writer:8080", "Log-service gRPC endpoint")
	flags.String("cache-address", "cache-balancer:3128", "Explicit Squid endpoint")
	flags.String("log-level", "info", "error, warn, info or debug")
	flags.String("address", ":8090", "Address for the gRPC server")
	flags.String("telemetry-address", ":9153", "Address for the telemetry server")
	flags.StringSlice("olric-address", []string{"localhost:3320"}, "Olric address")
	flags.String("olric-dmap", "robots-evaluator", "Olric DMap name")
	flags.Duration("recording-timeout", 30*time.Second, "Maximum time for all recording writes in one refresh")
	flags.Duration("robots-cache-freshness", defaultRobotsCacheFreshness,
		"Maximum normal age of cached robots.txt rules")
	flags.Duration("robots-unreachable-retry-interval", defaultRobotsUnreachableRetryInterval,
		"Delay before retrying an unreachable robots.txt")

	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	return viper.BindPFlags(pflag.CommandLine)
}

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

	level := viper.GetString("log-level")
	addr := viper.GetString("address")
	telemetryAddr := viper.GetString("telemetry-address")
	olricAddress := viper.GetStringSlice("olric-address")
	olricDmap := viper.GetString("olric-dmap")
	robotsCacheFreshness, robotsUnreachableRetryInterval, err := robotsCacheSettingsFromViper()
	if err != nil {
		return err
	}

	recordingTimeout := viper.GetDuration("recording-timeout")
	if recordingTimeout <= 0 {
		return fmt.Errorf("recording-timeout must be positive")
	}
	initLogger(os.Stderr, level)

	slog.Info(name, "version", version, "commit", commit, "date", date)

	app := &App{
		RecordingTimeout:               recordingTimeout,
		Addr:                           addr,
		DNSAddress:                     viper.GetString("dns-resolver-address"),
		WriterAddress:                  viper.GetString("content-writer-address"),
		LogAddress:                     viper.GetString("log-service-address"),
		CacheAddress:                   viper.GetString("cache-address"),
		TelemetryAddr:                  telemetryAddr,
		OlricAddr:                      olricAddress,
		OlricDmap:                      olricDmap,
		RobotsCacheFreshness:           robotsCacheFreshness,
		RobotsUnreachableRetryInterval: robotsUnreachableRetryInterval,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	return app.Run(ctx)
}

type App struct {
	RecordingTimeout               time.Duration
	DNSAddress                     string
	WriterAddress                  string
	LogAddress                     string
	CacheAddress                   string
	Addr                           string
	TelemetryAddr                  string
	OlricAddr                      []string
	OlricDmap                      string
	RobotsCacheFreshness           time.Duration
	RobotsUnreachableRetryInterval time.Duration

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

	defer func() { _ = telemetry.Close() }()
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
	defer func() { _ = app.cachier.Close(context.Background()) }()

	grpcServer := grpc.NewServer()

	connect := func(address string) (*grpc.ClientConn, error) {
		if address == "" {
			return nil, fmt.Errorf("service endpoint is required")
		}
		return grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	dnsConn, err := connect(app.DNSAddress)
	if err != nil {
		return err
	}
	defer func() { _ = dnsConn.Close() }()
	writerConn, err := connect(app.WriterAddress)
	if err != nil {
		return err
	}
	defer func() { _ = writerConn.Close() }()
	logConn, err := connect(app.LogAddress)
	if err != nil {
		return err
	}
	defer func() { _ = logConn.Close() }()
	transport, err := robots.NewTransport(dnsV1.NewDnsResolverClient(dnsConn), app.CacheAddress)
	if err != nil {
		return err
	}
	evaluator := robots.NewEvaluator(
		app.cachier,
		&http.Client{
			Timeout:       10 * time.Second,
			Transport:     transport,
			CheckRedirect: stopAfterFiveRedirects,
		},
		app.RobotsCacheFreshness,
		app.RobotsUnreachableRetryInterval,
	)
	evaluator.Recorder = robots.NewRecorder(
		writerV1.NewContentWriterClient(writerConn),
		logV1.NewLogClient(logConn), app.RecordingTimeout,
	)
	impl := &robots.EvaluatorServer{Evaluator: evaluator}

	robotsevaluatorV1.RegisterRobotsEvaluatorServer(grpcServer, impl)

	listener, err := net.Listen("tcp", app.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", app.Addr, err)
	}

	app.ready.Store(true)
	slog.Info("gRPC server listening", "address", app.Addr)

	g.Go(func() error { return grpcServer.Serve(listener) })

	<-ctx.Done()

	app.ready.Store(false)
	grpcServer.GracefulStop()
	_ = telemetry.Shutdown(context.Background())

	return g.Wait()
}

func validateRobotsCacheSettings(freshness, retryInterval time.Duration) error {
	if freshness <= 0 {
		return fmt.Errorf("robots-cache-freshness must be greater than zero")
	}
	if freshness > 24*time.Hour {
		return fmt.Errorf("robots-cache-freshness must not exceed 24h")
	}
	if retryInterval <= 0 {
		return fmt.Errorf("robots-unreachable-retry-interval must be greater than zero")
	}
	return nil
}

func robotsCacheSettingsFromViper() (time.Duration, time.Duration, error) {
	freshness := viper.GetDuration("robots-cache-freshness")
	retryInterval := viper.GetDuration("robots-unreachable-retry-interval")
	if err := validateRobotsCacheSettings(freshness, retryInterval); err != nil {
		return 0, 0, err
	}
	return freshness, retryInterval, nil
}

func stopAfterFiveRedirects(_ *http.Request, via []*http.Request) error {
	if len(via) > 5 {
		return http.ErrUseLastResponse
	}
	return nil
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
	cachier, err := app.newOlricCache(ctx)
	if err != nil {
		return err
	}
	app.cachier = cachier
	return nil
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
