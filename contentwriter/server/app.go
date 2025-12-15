package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/database"
	"github.com/nlnwa/gowarc"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type App struct {
	Addr           string
	TelemetryAddr  string
	DbOptions      database.Options
	ConfigCacheTTL time.Duration
	RedisOptions   *redis.Options
	RecordOptions  []gowarc.WarcRecordOption
	GrpcOptions    []grpc.ServerOption
	WriterOpts     WriterOptions

	ready atomic.Bool
}

func (app *App) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	const readyPath = "/readyz"
	const metricsPath = "/metrics"

	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.Handler())
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

	rethinkdb := database.NewRethinkDbConnection(app.DbOptions)
	defer func() {
		_ = rethinkdb.Close()
	}()
	redisClient := redis.NewClient(app.RedisOptions)
	defer func() {
		_ = redisClient.Close()
	}()

	init := new(errgroup.Group)
	init.Go(backoff(ctx, "rethinkdb", func() error {
		return rethinkdb.Connect()
	}))
	init.Go(backoff(ctx, "redis", func() (err error) {
		err = redisClient.Ping(ctx).Err()
		if err == nil {
			log.Info().Str("address", app.RedisOptions.Addr).Msg("Connected to Redis")
		}
		return err
	}))

	err := init.Wait()
	if err != nil {
		return err
	}

	configAdapter := database.NewConfigCache(rethinkdb, app.ConfigCacheTTL)
	contentAdapter := &database.CrawledContentHashCache{Client: redisClient}
	service := &ContentWriterService{
		warcWriterRegistry: newWarcWriterRegistry(app.WriterOpts, configAdapter, contentAdapter),
		configCache:        configAdapter,
		recordOptions:      app.RecordOptions,
	}
	defer service.Close()

	grpcServer := grpc.NewServer(app.GrpcOptions...)
	contentwriterV1.RegisterContentWriterServer(grpcServer, service)

	listener, err := net.Listen("tcp", app.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", app.Addr, err)
	}
	defer func() { _ = listener.Close() }()

	log.Info().Msgf("gRPC server listening on %s", app.Addr)

	g.Go(func() error { return grpcServer.Serve(listener) })

	app.ready.Store(true)

	<-ctx.Done()

	app.ready.Store(false)

	grpcServer.GracefulStop()
	_ = telemetry.Shutdown(context.Background())

	return g.Wait()
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !app.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready\n"))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func backoff(ctx context.Context, name string, fn func() error) func() error {
	return func() error {
		backoff := time.Second
		timer := time.NewTimer(backoff)
		const maxBackoff = 30 * time.Second

		for {
			err := fn()
			if err == nil {
				return nil
			}
			slog.Warn("Connection failed, retrying...", "error", err, "backoff", backoff.String(), "service", name)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
			if backoff < maxBackoff {
				backoff *= 2
			}
			timer.Reset(backoff)
		}
	}
}
