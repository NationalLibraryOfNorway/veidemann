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
	"syscall"
	"time"

	scopecheckerV1 "github.com/NationalLibraryOfNorway/veidemann/api/scopechecker/v1"
	uricanonicalizerV1 "github.com/NationalLibraryOfNorway/veidemann/api/uricanonicalizer/v1"
	"github.com/NationalLibraryOfNorway/veidemann/scopeservice/internal/script"
	"github.com/NationalLibraryOfNorway/veidemann/scopeservice/internal/service"
	otgrpc "github.com/opentracing-contrib/go-grpc"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

var (
	name    = "scopeservice"
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

	slog.Info(name, "version", version, "commit", commit, "date", date)

	// Initialize tracer
	closer, err := initTracer(name)
	if err != nil {
		slog.Warn("Failed to initialize tracer", "error", err)
	}
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	tracer := opentracing.GlobalTracer()
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)),
		grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(tracer)),
	)

	script.InitializeCanonicalizationProfiles(opts.IncludeFragments())

	scopecheckerV1.RegisterScopesCheckerServiceServer(grpcServer, service.ScopeChecker{})
	uricanonicalizerV1.RegisterUriCanonicalizerServiceServer(grpcServer, service.UriCanonicalizer{})

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
