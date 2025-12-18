package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
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
	Addr           string
	TelemetryAddr  string
	DbOptions      database.Options
	ConfigCacheTTL time.Duration
	RedisOptions   *redis.Options
	RecordOptions  []gowarc.WarcRecordOption
	GrpcOptions    []grpc.ServerOption
	WriterOpts     writer.Options
	Uploader       Uploader

	ready atomic.Bool
}

func normalizeShutdownError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}

func (app *App) Run(ctx context.Context) error {
	g := new(errgroup.Group)

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
		log.Warn().Err(err).Msg("Telemetry server stopped")
		return nil
	})

	log.Info().Str("address", app.TelemetryAddr).Msg("Telemetry server listening")

	rethinkdb := database.NewRethinkDbConnection(app.DbOptions)
	defer func() {
		_ = rethinkdb.Close()
	}()
	redisClient := redis.NewClient(app.RedisOptions)
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
			log.Info().Str("address", app.RedisOptions.Addr).Msg("Connected to Redis")
		}
		return err
	}))
	if err := init.Wait(); err != nil {
		return normalizeShutdownError(err)
	}

	writerOpts := app.WriterOpts
	var manager *upload.Manager

	if app.Uploader != nil {
		manager = upload.NewManager(1024, func(ctx context.Context, filePath string) error {
			return finalize(ctx, filePath, app.Uploader)
		})

		writerOpts.AfterFileCreationHook = func(filename string, size int64, warcInfoId string) error {
			manager.Enqueue(filename)
			return nil
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

	if app.Uploader != nil {
		// Use a ctx that survives SIGTERM
		hardCtx := context.Background()
		g.Go(func() error { return manager.Run(hardCtx) })
	}

	app.ready.Store(true)

	<-ctx.Done()

	app.ready.Store(false)

	log.Warn().Err(ctx.Err()).Msg("Shutting down")

	grpcServer.GracefulStop()

	warcWriterRegistry.Close()

	if app.Uploader != nil {
		mapLeftovers(manager.Enqueue, writerOpts.WarcDir)
		manager.Close()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = telemetry.Shutdown(shutdownCtx)

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

func finalize(ctx context.Context, filePath string, uploader Uploader) error {
	md5sum, err := calculateMD5(filePath)
	if err != nil {
		return err
	}
	start := time.Now()
	info, err := uploader.Upload(ctx, filePath, md5sum)
	if err != nil {
		return fmt.Errorf("failed to upload file: %s: %w", filePath, err)
	}

	metrics.Duration(time.Since(start))

	metrics.Size(info.Size)

	log.Debug().
		Str("key", info.Key).
		Int64("size", info.Size).
		Str("etag", info.ETag).
		Str("duration", time.Since(start).String()).
		Str("md5", md5sum).
		Msg("Uploaded file")

	err = os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to remove file after upload: %s: %w", filePath, err)
	}

	return nil
}

func mapLeftovers(u func(string), dir string) {
	patterns := []string{
		filepath.Join(dir, "*.open"),
		filepath.Join(dir, "*.warc.gz"),
	}

	seen := map[string]struct{}{}
	for _, p := range patterns {
		matches, _ := filepath.Glob(p)
		for _, f := range matches {
			// dedupe
			if _, ok := seen[f]; ok {
				continue
			}
			seen[f] = struct{}{}

			u(f)
		}
	}
}
