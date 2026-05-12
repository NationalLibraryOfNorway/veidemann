package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/database"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/internal/metrics"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/internal/upload"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/internal/writer"
	"github.com/minio/minio-go/v7"
	"github.com/nlnwa/gowarc"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type Uploader interface {
	Upload(ctx context.Context, filePath string, md5sum string) (minio.UploadInfo, error)
}

type App struct {
	Addr                 string
	TelemetryAddr        string
	UploadFallbackDir    string
	UploadInstanceID     string
	UploadScanInterval   time.Duration
	UploadTimeout        time.Duration
	DbOptions            database.Options
	ConfigCacheTTL       time.Duration
	RedisOptions         *redis.Options
	RedisFailoverOptions *redis.FailoverOptions
	RecordOptions        []gowarc.WarcRecordOption
	GrpcOptions          []grpc.ServerOption
	WriterOpts           writer.Options
	Uploader             Uploader

	ready atomic.Bool
}

func normalizeShutdownError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}

func (app *App) Run(ctx context.Context) error {
	g, gctx := errgroup.WithContext(ctx)

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
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("telemetry server: %w", err)
		}
		log.Warn().Err(err).Msg("Telemetry server stopped")
		return nil
	})

	log.Info().Str("address", app.TelemetryAddr).Msg("Telemetry server listening")

	rethinkdb := database.NewRethinkDbConnection(app.DbOptions)
	defer func() {
		_ = rethinkdb.Close()
	}()
	var redisClient *redis.Client
	var redisTarget string
	if app.RedisFailoverOptions != nil {
		redisClient = redis.NewFailoverClient(app.RedisFailoverOptions)
		redisTarget = fmt.Sprintf(
			"sentinel master %s via %s",
			app.RedisFailoverOptions.MasterName,
			strings.Join(app.RedisFailoverOptions.SentinelAddrs, ","),
		)
	} else {
		redisClient = redis.NewClient(app.RedisOptions)
		redisTarget = app.RedisOptions.Addr
	}
	defer func() {
		_ = redisClient.Close()
	}()

	init, ictx := errgroup.WithContext(ctx)
	init.Go(backoff(ictx, "rethinkdb", func() error {
		return rethinkdb.Connect()
	}))
	init.Go(backoff(ictx, "redis", func() (err error) {
		err = redisClient.Ping(ictx).Err()
		if err == nil {
			log.Info().Str("address", redisTarget).Msg("Connected to Redis")
		}
		return err
	}))
	if err := init.Wait(); err != nil {
		return normalizeShutdownError(err)
	}

	writerOpts := app.WriterOpts
	var manager *upload.Manager
	var managerErr error
	managerEnabled := app.Uploader != nil || app.UploadFallbackDir != ""

	if managerEnabled {
		manager, managerErr = upload.NewManager(upload.ManagerConfig{
			QueueSize:     1024,
			WarcDir:       writerOpts.WarcDir,
			FallbackDir:   app.UploadFallbackDir,
			InstanceID:    app.UploadInstanceID,
			ScanInterval:  app.UploadScanInterval,
			UploadTimeout: app.UploadTimeout,
			Uploader:      app.Uploader,
		})
		if managerErr != nil {
			return fmt.Errorf("failed to initialize upload manager: %w", managerErr)
		}

		writerOpts.AfterFileCreationHook = func(filename string, size int64, warcInfoId string) error {
			metrics.WrittenSizeBytes(size)
			return manager.Enqueue(filename)
		}
	}

	configAdapter := database.NewConfigCache(rethinkdb, app.ConfigCacheTTL)
	contentAdapter := &database.CrawledContentHashCache{Client: redisClient}
	warcWriterRegistry := newWarcWriterRegistry(writerOpts, configAdapter, contentAdapter)

	service := &ContentWriterService{
		warcWriterRegistry: warcWriterRegistry,
		configCache:        configAdapter,
		recordOptions:      app.RecordOptions,
	}

	grpcServer := grpc.NewServer(app.GrpcOptions...)
	contentwriterV1.RegisterContentWriterServer(grpcServer, service)

	listener, err := net.Listen("tcp", app.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", app.Addr, err)
	}

	log.Info().Msgf("gRPC server listening on %s", app.Addr)

	g.Go(func() error { return grpcServer.Serve(listener) })

	if managerEnabled {
		// Use a ctx that survives SIGTERM
		hardCtx := context.Background()
		g.Go(func() error { return manager.Run(hardCtx) })
	}

	app.ready.Store(true)

	backgroundFailure := false
	select {
	case <-ctx.Done():
	case <-gctx.Done():
		backgroundFailure = ctx.Err() == nil
	}

	app.ready.Store(false)

	if backgroundFailure {
		log.Warn().Msg("Shutting down after background task failure")
	} else {
		log.Warn().Err(ctx.Err()).Msg("Shutting down")
	}

	grpcServer.GracefulStop()

	warcWriterRegistry.Close()

	var scanErr error
	if managerEnabled {
		if !backgroundFailure {
			scanErr = manager.ScanNow()
		}
		manager.Close()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = telemetry.Shutdown(shutdownCtx)

	if scanErr != nil {
		return normalizeShutdownError(scanErr)
	}
	return normalizeShutdownError(g.Wait())
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
		const maxBackoff = 30 * time.Second

		for {
			err := fn()
			if err == nil {
				return nil
			}

			log.Warn().Err(err).Dur("backoff", backoff).Str("service", name).
				Msg("Connection failed, retrying...")

			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}

			if backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
		}
	}
}
