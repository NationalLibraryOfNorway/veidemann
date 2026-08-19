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
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/log-service/internal/logservice"
	"github.com/NationalLibraryOfNorway/veidemann/log-service/internal/parquet"
	"github.com/NationalLibraryOfNorway/veidemann/log-service/internal/recentforward"
	"github.com/NationalLibraryOfNorway/veidemann/log-service/internal/recentlog"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	otgrpc "github.com/opentracing-contrib/go-grpc"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/uber/jaeger-client-go/config"
	jaegerLog "github.com/uber/jaeger-client-go/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	name    = "log-service"
	version = ""
	commit  = ""
	date    = ""
)

const (
	modeCombined = "combined"
	modeWriter   = "writer"
	modeRecent   = "recent"
)

type Options struct {
}

func parseFlags() (Options, error) {
	flags := pflag.CommandLine
	flags.String("host", "", "Interface the log service API is listening to. No value means all interfaces.")
	flags.Int("port", 8090, "Port the log service api is listening to")
	flags.String("mode", modeCombined, "Runtime mode: combined, writer, or recent")

	flags.String("parquet-dir", "./data/parquet", "Directory where parquet files are written")
	flags.Int64("max-lines-per-file", 1000000, "Rotate parquet file when this many rows are written")
	flags.String("recent-log-db-path", "./data/recent-logs.db", "Path to the SQLite database used for recent log reads")
	flags.Int64("recent-crawl-log-max-entries", 1000000, "Maximum number of crawl logs retained in the recent read store")
	flags.Int64("recent-page-log-max-entries", 250000, "Maximum number of page logs retained in the recent read store")
	flags.String("recent-log-service-address", "", "Optional host:port of the recent log service used by writer mode")
	flags.Int("recent-forward-queue-size", 1024, "Maximum number of archived logs waiting for recent-service forwarding")
	flags.Int("recent-forward-workers", 2, "Number of asynchronous recent-service forwarding workers")
	flags.Duration("recent-forward-timeout", 5*time.Second, "Timeout for one asynchronous recent-service forward")
	flags.Duration("recent-forward-shutdown-timeout", 30*time.Second, "Maximum time to drain recent-service forwards during shutdown")
	flags.String("s3-endpoint", "", "S3-compatible endpoint for parquet handoff. If empty, parquet files remain on local disk")
	flags.String("s3-bucket", "", "S3-compatible bucket for parquet handoff")
	flags.String("s3-access-key", "", "Access key for S3-compatible parquet handoff")
	flags.String("s3-secret-key", "", "Secret key for S3-compatible parquet handoff")
	flags.String("s3-key-prefix", "", "Optional S3 object key prefix for parquet handoff")
	flags.Bool("s3-insecure", false, "Use HTTP instead of HTTPS for S3-compatible parquet handoff when the endpoint has no scheme")
	flags.Duration("s3-upload-delay", 0, "Delay before uploading finalized parquet files to S3. Example: 72h for 3 days. Zero uploads on close")
	flags.Duration("s3-scan-interval", time.Minute, "Interval for scanning finalized parquet files for S3 upload eligibility")

	flags.String("log-level", "info", "Log level, available levels are: panic, fatal, error, warn, info, debug and trace")
	flags.String("log-formatter", "logfmt", "Log formatter, available values are: logfmt and json")
	flags.Bool("log-method", false, "Log file:line of method caller")

	pflag.String("metrics-address", ":9153", "address to expose metrics on")

	pflag.Parse()

	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	//  viper.SetEnvPrefix("CONTENTWRITER")
	viper.AutomaticEnv()

	return Options{}, viper.BindPFlags(flags)
}

func (o Options) LogLevel() string {
	return viper.GetString("log-level")
}

func (o Options) Mode() string {
	return strings.ToLower(strings.TrimSpace(viper.GetString("mode")))
}

func (o Options) LogFormatter() string {
	return viper.GetString("log-formatter")
}

func (o Options) LogMethod() bool {
	return viper.GetBool("log-method")
}

func (o Options) ParquetDir() string {
	return viper.GetString("parquet-dir")
}

func (o Options) MaxLinesPerFile() int64 {
	return viper.GetInt64("max-lines-per-file")
}

func (o Options) RecentLogDBPath() string {
	return viper.GetString("recent-log-db-path")
}

func (o Options) RecentCrawlLogMaxEntries() int64 {
	return viper.GetInt64("recent-crawl-log-max-entries")
}

func (o Options) RecentPageLogMaxEntries() int64 {
	return viper.GetInt64("recent-page-log-max-entries")
}

func (o Options) RecentLogServiceAddress() string {
	return strings.TrimSpace(viper.GetString("recent-log-service-address"))
}

func (o Options) RecentForwardQueueSize() int {
	return viper.GetInt("recent-forward-queue-size")
}

func (o Options) RecentForwardWorkers() int {
	return viper.GetInt("recent-forward-workers")
}

func (o Options) RecentForwardTimeout() time.Duration {
	return viper.GetDuration("recent-forward-timeout")
}

func (o Options) RecentForwardShutdownTimeout() time.Duration {
	return viper.GetDuration("recent-forward-shutdown-timeout")
}

func (o Options) Host() string {
	return viper.GetString("host")
}

func (o Options) Port() int {
	return viper.GetInt("port")
}

func (o Options) TelemetryAddr() string {
	return viper.GetString("metrics-address")
}

func (o Options) S3Endpoint() string {
	return viper.GetString("s3-endpoint")
}

func (o Options) S3Bucket() string {
	return viper.GetString("s3-bucket")
}

func (o Options) S3AccessKey() string {
	return viper.GetString("s3-access-key")
}

func (o Options) S3SecretKey() string {
	return viper.GetString("s3-secret-key")
}

func (o Options) S3KeyPrefix() string {
	return viper.GetString("s3-key-prefix")
}

func (o Options) S3Insecure() bool {
	return viper.GetBool("s3-insecure")
}

func (o Options) S3UploadDelay() time.Duration {
	return viper.GetDuration("s3-upload-delay")
}

func (o Options) S3ScanInterval() time.Duration {
	return viper.GetDuration("s3-scan-interval")
}

func (o Options) Validate() error {
	switch o.Mode() {
	case modeCombined, modeWriter, modeRecent:
	default:
		return fmt.Errorf("mode must be one of %q, %q, or %q", modeCombined, modeWriter, modeRecent)
	}
	if o.Mode() != modeWriter && o.RecentLogServiceAddress() != "" {
		return fmt.Errorf("recent-log-service-address is only valid in writer mode")
	}
	if o.Mode() == modeWriter {
		if strings.TrimSpace(o.ParquetDir()) == "" {
			return fmt.Errorf("parquet-dir must not be empty in writer mode")
		}
		if o.RecentForwardQueueSize() <= 0 {
			return fmt.Errorf("recent-forward-queue-size must be > 0")
		}
		if o.RecentForwardWorkers() <= 0 {
			return fmt.Errorf("recent-forward-workers must be > 0")
		}
		if o.RecentForwardTimeout() <= 0 {
			return fmt.Errorf("recent-forward-timeout must be > 0")
		}
		if o.RecentForwardShutdownTimeout() <= 0 {
			return fmt.Errorf("recent-forward-shutdown-timeout must be > 0")
		}
	}
	if o.Mode() != modeWriter && strings.TrimSpace(o.RecentLogDBPath()) == "" {
		return fmt.Errorf("recent-log-db-path must not be empty in %s mode", o.Mode())
	}
	return nil
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
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	opts, err := parseFlags()
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	if err := opts.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	initLog(opts.LogLevel(), opts.LogFormatter(), opts.LogMethod())

	slog.Info("Service version", "name", name, "version", version, "commit", commit, "date", date, "mode", opts.Mode())

	closer := initTracer(name)
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	tracer := opentracing.GlobalTracer()
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)),
		grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(tracer)),
	)

	var storage *parquet.Storage
	if opts.Mode() == modeCombined || opts.Mode() == modeWriter {
		storageOpts := make([]parquet.Option, 0, 1)
		s3Handoff, err := newParquetS3Handoff(opts)
		if err != nil {
			return err
		}
		if s3Handoff != nil {
			storageOpts = append(storageOpts, parquet.WithPostCloseHandoff(s3Handoff))
			logArgs := []any{
				"endpoint", opts.S3Endpoint(),
				"bucket", opts.S3Bucket(),
				"keyPrefix", opts.S3KeyPrefix(),
				"uploadDelay", opts.S3UploadDelay(),
				"scanInterval", opts.S3ScanInterval(),
			}
			if opts.S3UploadDelay() > 0 {
				slog.Info("Enabled parquet S3 archival with delayed upload", logArgs...)
			} else {
				slog.Info("Enabled parquet S3 handoff on close", logArgs...)
			}
		} else {
			slog.Info("Parquet S3 handoff disabled; finalized files remain on local disk")
		}

		storage, err = parquet.New(opts.ParquetDir(), opts.MaxLinesPerFile(), storageOpts...)
		if err != nil {
			if s3Handoff != nil {
				_ = s3Handoff.Close()
			}
			return fmt.Errorf("failed to initialize parquet storage: %w", err)
		}
		defer func() {
			slog.Info("Closing parquet storage")
			if err := storage.Close(); err != nil {
				slog.Error("Failed to close parquet storage", "error", err)
			}
			if s3Handoff != nil {
				slog.Info("Closing parquet S3 handoff")
				if err := s3Handoff.Close(); err != nil {
					slog.Error("Failed to close parquet S3 handoff", "error", err)
				}
			}
		}()
		slog.Info("Initialized parquet storage backend", "dir", opts.ParquetDir(), "maxLinesPerFile", opts.MaxLinesPerFile())
	}

	var recentStore *recentlog.Store
	if opts.Mode() == modeCombined || opts.Mode() == modeRecent {
		recentLogDBPath, err := filepath.Abs(opts.RecentLogDBPath())
		if err != nil {
			return fmt.Errorf("resolve recent log database path: %w", err)
		}
		recentMetrics := recentlog.NewMetrics(prometheus.DefaultRegisterer, recentLogDBPath)
		recentStore, err = recentlog.New(ctx, recentlog.Config{
			Path:            recentLogDBPath,
			CrawlMaxEntries: opts.RecentCrawlLogMaxEntries(),
			PageMaxEntries:  opts.RecentPageLogMaxEntries(),
			Metrics:         recentMetrics,
		})
		if err != nil {
			return fmt.Errorf("failed to initialize recent log read store: %w", err)
		}
		defer func() {
			slog.Info("Closing recent log read store")
			if err := recentStore.Close(); err != nil {
				slog.Error("Failed to close recent log read store", "error", err)
			}
		}()
		slog.Info("Initialized recent log read store",
			"path", recentLogDBPath,
			"crawlMaxEntries", opts.RecentCrawlLogMaxEntries(),
			"pageMaxEntries", opts.RecentPageLogMaxEntries(),
		)
	}

	switch opts.Mode() {
	case modeCombined:
		logV1.RegisterLogServer(grpcServer, logservice.New(storage, recentStore))
	case modeRecent:
		logV1.RegisterLogServer(grpcServer, logservice.NewRecent(recentStore))
	case modeWriter:
		var forwarder logservice.RecentForwarder
		if address := opts.RecentLogServiceAddress(); address != "" {
			conn, err := grpc.NewClient(address,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithStreamInterceptor(otgrpc.OpenTracingStreamClientInterceptor(tracer)),
			)
			if err != nil {
				return fmt.Errorf("create recent log service client: %w", err)
			}
			defer func() {
				if err := conn.Close(); err != nil {
					slog.Error("Failed to close recent log service connection", "error", err)
				}
			}()
			metrics := recentforward.NewMetrics(prometheus.DefaultRegisterer)
			asyncForwarder, err := recentforward.New(
				recentforward.NewLogWriter(logV1.NewLogClient(conn)),
				recentforward.Config{
					QueueSize: opts.RecentForwardQueueSize(),
					Workers:   opts.RecentForwardWorkers(),
					Timeout:   opts.RecentForwardTimeout(),
					Metrics:   metrics,
				},
			)
			if err != nil {
				return fmt.Errorf("initialize recent log forwarder: %w", err)
			}
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), opts.RecentForwardShutdownTimeout())
				defer cancel()
				if err := asyncForwarder.Close(shutdownCtx); err != nil {
					slog.Warn("Recent log forwarder did not drain before shutdown deadline", "error", err)
				}
			}()
			forwarder = asyncForwarder
			slog.Info("Enabled asynchronous recent-log forwarding",
				"address", address,
				"queueSize", opts.RecentForwardQueueSize(),
				"workers", opts.RecentForwardWorkers(),
				"timeout", opts.RecentForwardTimeout(),
			)
		} else {
			slog.Info("Recent-log forwarding disabled; writes are archived only")
		}
		logV1.RegisterLogServer(grpcServer, logservice.NewWriter(storage, forwarder))
	}

	g, groupCtx := errgroup.WithContext(ctx)

	const readyPath = "/readyz"
	const metricsPath = "/metrics"

	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.Handler())
	mux.Handle(readyPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	telemetry := &http.Server{
		Addr:    opts.TelemetryAddr(),
		Handler: mux,
	}

	g.Go(func() error {
		err := telemetry.ListenAndServe()
		slog.Warn("Telemetry server stopped", "error", err)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	slog.Info("Telemetry server listening", "address", opts.TelemetryAddr())

	addr := fmt.Sprintf("%s:%d", opts.Host(), opts.Port())

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	slog.Info("gRPC server listening", "address", addr)

	g.Go(func() error { return grpcServer.Serve(listener) })

	<-groupCtx.Done()

	slog.Info("Shutting down gracefully")

	grpcServer.GracefulStop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = telemetry.Shutdown(shutdownCtx)

	return g.Wait()
}

func newParquetS3Handoff(opts Options) (*parquet.AsyncS3Handoff, error) {
	endpoint := strings.TrimSpace(opts.S3Endpoint())
	if endpoint == "" {
		return nil, nil
	}
	if strings.TrimSpace(opts.S3Bucket()) == "" {
		return nil, fmt.Errorf("s3-bucket must be set when s3-endpoint is provided")
	}
	if strings.TrimSpace(opts.S3AccessKey()) == "" || strings.TrimSpace(opts.S3SecretKey()) == "" {
		return nil, fmt.Errorf("s3-access-key and s3-secret-key must be set when s3-endpoint is provided")
	}

	parsedEndpoint, secure, err := parseS3Endpoint(endpoint, !opts.S3Insecure())
	if err != nil {
		return nil, fmt.Errorf("invalid s3 endpoint: %w", err)
	}

	client, err := minio.New(parsedEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(opts.S3AccessKey(), opts.S3SecretKey(), ""),
		Secure: secure,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create s3 client: %w", err)
	}

	handoff, err := parquet.NewAsyncS3Handoff(client, parquet.AsyncS3HandoffConfig{
		BaseDir:      opts.ParquetDir(),
		Bucket:       opts.S3Bucket(),
		KeyPrefix:    opts.S3KeyPrefix(),
		ScanInterval: opts.S3ScanInterval(),
		UploadDelay:  opts.S3UploadDelay(),
		OnError: func(file parquet.FinalizedParquetFile, err error) {
			slog.Error("Parquet S3 handoff failed", "error", err, "path", file.Path, "table", file.Table, "collection", file.Collection)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize parquet s3 handoff: %w", err)
	}
	return handoff, nil
}

func parseS3Endpoint(raw string, defaultSecure bool) (string, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, fmt.Errorf("endpoint must not be empty")
	}
	if !strings.Contains(raw, "://") {
		return raw, defaultSecure, nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false, err
	}
	if parsed.Host == "" {
		return "", false, fmt.Errorf("endpoint host must not be empty")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", false, fmt.Errorf("endpoint must not include a path")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http":
		return parsed.Host, false, nil
	case "https":
		return parsed.Host, true, nil
	default:
		return "", false, fmt.Errorf("unsupported endpoint scheme %q", parsed.Scheme)
	}
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

// initTracer initializes the global OpenTracing tracer using Jaeger configuration from environment variables.
func initTracer(service string) io.Closer {
	cfg, err := config.FromEnv()
	if err != nil {
		return nil
	}

	if cfg.ServiceName == "" {
		cfg.ServiceName = service
	}

	tracer, closer, err := cfg.NewTracer(config.Logger(jaegerLog.StdLogger))
	if err == nil {
		opentracing.SetGlobalTracer(tracer)
	}

	return closer
}
