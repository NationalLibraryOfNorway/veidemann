package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"

	"os"
	"os/signal"
	"syscall"

	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/database"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/internal/flags"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/internal/upload"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/internal/writer"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/server"
	"github.com/nlnwa/gowarc"
	otgrpc "github.com/opentracing-contrib/go-grpc"
	"github.com/opentracing/opentracing-go"
	"github.com/redis/go-redis/v9"
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
	err := run()
	if err != nil {
		slog.Error("Bye", "error", err)
		os.Exit(1)
	}
	slog.Info("Goodbye!")
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	opts, err := flags.ParseFlags()
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	initLogger(os.Stderr, opts.LogLevel())

	closer := initTracer(name)
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	slog.Info(name, "version", version, "commit", commit, "date", date)

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

	if opts.MaxReceiveMessageSize() > 0 {
		grpcServerOptions = append(grpcServerOptions, grpc.MaxRecvMsgSize(opts.MaxReceiveMessageSize()))
	}
	if opts.MaxSendMessageSize() > 0 {
		grpcServerOptions = append(grpcServerOptions, grpc.MaxSendMsgSize(opts.MaxSendMessageSize()))
	}

	var uploader server.Uploader

	if opts.S3Address() != "" && opts.S3BucketName() != "" {
		uploader, err = upload.NewS3Uploader(
			upload.WithS3Address(opts.S3Address()),
			upload.WithS3BucketName(opts.S3BucketName()),
			upload.WithS3AccessKeyID(opts.S3AccessKeyID()),
			upload.WithS3SecretAccessKey(opts.S3SecretAccessKey()),
			upload.WithS3Token(opts.S3Token()),
			upload.WithSecure(opts.S3Secure()),
		)
		if err != nil {
			return fmt.Errorf("failed to create S3 uploader: %w", err)
		}
	}

	app := &server.App{
		Addr:               fmt.Sprintf("%s:%d", opts.Interface(), opts.Port()),
		UploadFallbackDir:  opts.WarcFallbackDir(),
		UploadInstanceID:   opts.HostName(),
		UploadScanInterval: opts.UploadRetryScanInterval(),
		UploadTimeout:      opts.UploadTimeout(),
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
		RecordOptions: recordOpts,
		GrpcOptions:   grpcServerOptions,
		WriterOpts: writer.Options{
			WarcDir:     opts.WarcDir(),
			WarcVersion: opts.WarcVersion(),
			Flush:       opts.FlushRecord(),
			PoolSize:    opts.WarcWriterPoolSize(),
		},
		TelemetryAddr: opts.MetricsAddress(),
		Uploader:      uploader,
	}

	if opts.UseRedisSentinel() {
		app.RedisFailoverOptions = &redis.FailoverOptions{
			MasterName:       opts.RedisMasterName(),
			SentinelAddrs:    opts.RedisSentinelAddrs(),
			Password:         opts.RedisPassword(),
			SentinelPassword: opts.RedisSentinelPassword(),
			DB:               opts.RedisDb(),
		}
	} else {
		app.RedisOptions = &redis.Options{
			Addr:     fmt.Sprintf("%s:%d", opts.RedisHost(), opts.RedisPort()),
			DB:       opts.RedisDb(),
			Password: opts.RedisPassword(),
		}
	}

	return app.Run(ctx)
}

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

// initTracer initializes the Jaeger tracer based on environment variables and sets it as the global tracer.
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
