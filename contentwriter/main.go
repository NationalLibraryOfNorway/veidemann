package main

import (
	"context"
	"fmt"
	"io"
	stdlog "log"

	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/database"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/internal/flags"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/server"
	"github.com/nlnwa/gowarc"
	otgrpc "github.com/opentracing-contrib/go-grpc"
	"github.com/opentracing/opentracing-go"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/uber/jaeger-client-go/config"
	jaegerLog "github.com/uber/jaeger-client-go/log"

	"google.golang.org/grpc"
)

var (
	name    = "contentwriter"
	version = ""
	commit  = ""
	date    = ""
)

func main() {
	opts, err := flags.ParseFlags()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse flags")
	}

	initLogging(opts.LogLevel(), opts.LogFormatter())

	closer := initTracer(name)
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	log.Info().Msgf("%s version %s, commit %s, date %s", name, version, commit, date)

	err = run(opts)
	if err != nil {
		log.Fatal().Err(err).Msg("Goodbye")
	}
}

func run(opts flags.Options) error {
	recordOpts := []gowarc.WarcRecordOption{
		gowarc.WithBufferTmpDir(opts.WorkDir()),
		gowarc.WithVersion(opts.WarcVersion()),
	}
	if opts.UseStrictValidation() {
		recordOpts = append(recordOpts, gowarc.WithStrictValidation())
	}

	tracer := opentracing.GlobalTracer()
	grpcServerOptions := []grpc.ServerOption{
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)),
		grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(tracer)),
	}

	app := &server.App{
		Addr:           fmt.Sprintf("%s:%d", opts.Interface(), opts.Port()),
		ConfigCacheTTL: opts.DbCacheTTL(),
		DbOptions: database.Options{
			Address:            fmt.Sprintf("%s:%d", opts.DbHost(), opts.DbPort()),
			Username:           opts.DbUser(),
			Password:           opts.DbPassword(),
			Database:           opts.DbName(),
			QueryTimeout:       opts.DbQueryTimeout(),
			MaxOpenConnections: opts.DbMaxOpenConn(),
			MaxRetries:         opts.DbMaxRetries(),
			UseOpenTracing:     opts.DbUseOpenTracing(),
		},
		RedisOptions: &redis.Options{
			Addr:     fmt.Sprintf("%s:%d", opts.RedisHost(), opts.RedisPort()),
			DB:       opts.RedisDb(),
			Password: opts.RedisPassword(),
		},
		RecordOptions: recordOpts,
		GrpcOptions:   grpcServerOptions,
		WriterOpts: server.WriterOptions{
			WarcDir:            opts.WarcDir(),
			WarcVersion:        opts.WarcVersion(),
			WarcWriterPoolSize: opts.WarcWriterPoolSize(),
			FlushRecord:        opts.FlushRecord(),
		},
		TelemetryAddr: fmt.Sprintf("%s:%d", opts.MetricsInterface(), opts.MetricsPort()),
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	return app.Run(ctx)
}

func initLogging(level string, format string) {
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

	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)

	log.Info().Msgf("Setting log level to %s", level)
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
