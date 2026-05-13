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
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/log-service/internal/logservice"
	"github.com/NationalLibraryOfNorway/veidemann/log-service/internal/parquet"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	otgrpc "github.com/opentracing-contrib/go-grpc"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/uber/jaeger-client-go/config"
	jaegerLog "github.com/uber/jaeger-client-go/log"
	"google.golang.org/grpc"
)

var (
	name    = "log-service"
	version = ""
	commit  = ""
	date    = ""
)

type Options struct {
}

func parseFlags() (Options, error) {
	flags := pflag.CommandLine
	flags.String("host", "", "Interface the log service API is listening to. No value means all interfaces.")
	flags.Int("port", 8090, "Port the log service api is listening to")

	flags.String("parquet-dir", "./data/parquet", "Directory where parquet files are written")
	flags.Int64("max-lines-per-file", 100000, "Rotate parquet file when this many rows are written")
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

func main() {
	err := run()
	if err != nil {
		log.Error().Err(err).Msg("Bye!")
		os.Exit(1)
	}
	log.Info().Msg("Goodbye!")
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	opts, err := parseFlags()
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	initLog(opts.LogLevel(), opts.LogFormatter(), opts.LogMethod())

	log.Info().Msgf("%s version %s, commit %s, date %s", name, version, commit, date)

	closer := initTracer(name)
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	tracer := opentracing.GlobalTracer()
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)),
		grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(tracer)),
	)

	storageOpts := make([]parquet.Option, 0, 1)
	s3Handoff, err := newParquetS3Handoff(opts)
	if err != nil {
		return err
	}
	if s3Handoff != nil {
		storageOpts = append(storageOpts, parquet.WithPostCloseHandoff(s3Handoff))
		logEvent := log.Info().
			Str("endpoint", opts.S3Endpoint()).
			Str("bucket", opts.S3Bucket()).
			Str("keyPrefix", opts.S3KeyPrefix()).
			Dur("uploadDelay", opts.S3UploadDelay()).
			Dur("scanInterval", opts.S3ScanInterval())
		if opts.S3UploadDelay() > 0 {
			logEvent.Msg("Enabled parquet S3 archival with delayed upload")
		} else {
			logEvent.Msg("Enabled parquet S3 handoff on close")
		}
	} else {
		log.Info().Msg("Parquet S3 handoff disabled; finalized files remain on local disk")
	}

	storage, err := parquet.New(opts.ParquetDir(), opts.MaxLinesPerFile(), storageOpts...)
	if err != nil {
		return fmt.Errorf("failed to initialize parquet storage: %w", err)
	}
	defer func() {
		log.Info().Msg("Closing parquet storage")
		if err := storage.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close parquet storage")
		}
	}()

	log.Info().
		Str("dir", opts.ParquetDir()).
		Int64("maxLinesPerFile", opts.MaxLinesPerFile()).
		Msg("Initialized parquet storage backend")

	logServer := logservice.New(storage)
	logV1.RegisterLogServer(grpcServer, logServer)

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
		log.Warn().Err(err).Msg("Telemetry server stopped")
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	log.Info().Str("address", opts.TelemetryAddr()).Msg("Telemetry server listening")

	addr := fmt.Sprintf("%s:%d", opts.Host(), opts.Port())

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	log.Info().Msgf("gRPC server listening on %s", addr)

	g.Go(func() error { return grpcServer.Serve(listener) })

	<-groupCtx.Done()

	log.Info().Msg("Shutting down gracefully")

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
			log.Error().Err(err).Str("path", file.Path).Str("table", file.Table).Str("collection", file.Collection).Msg("Parquet S3 handoff failed")
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
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	switch strings.ToLower(level) {
	case "panic":
		log.Logger = log.Level(zerolog.PanicLevel)
	case "fatal":
		log.Logger = log.Level(zerolog.FatalLevel)
	case "error":
		log.Logger = log.Level(zerolog.ErrorLevel)
	case "warn":
		log.Logger = log.Level(zerolog.WarnLevel)
	case "info":
		log.Logger = log.Level(zerolog.InfoLevel)
	case "debug":
		log.Logger = log.Level(zerolog.DebugLevel)
	case "trace":
		log.Logger = log.Level(zerolog.TraceLevel)
	}

	if format == "logfmt" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}

	if logCaller {
		log.Logger = log.With().Caller().Logger()
	}

	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)
}

// Init returns an instance of Jaeger Tracer that samples 100% of traces and logs all spans to stdout.
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
