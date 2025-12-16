package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	"github.com/NationalLibraryOfNorway/veidemann/frontier-queue-workers/database"
)

type App struct {
	Addr          string
	DbOptions     database.RethinkDbOptions
	RedisOptions  *redis.UniversalOptions
	TelemetryAddr string

	ready atomic.Bool
}

func (app *App) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	const readyPath = "/readyz"

	mux := http.NewServeMux()
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

	// setup rethinkdb connection
	rethinkDb := database.NewRethinkDbConnection(app.DbOptions)
	defer func() {
		_ = rethinkDb.Close()
	}()

	redisClient := redis.NewUniversalClient(app.RedisOptions)
	defer func() {
		_ = redisClient.Close()
	}()

	init := new(errgroup.Group)
	init.Go(backoff(ctx, "rethinkdb", func() error {
		return rethinkDb.Connect()
	}))
	init.Go(backoff(ctx, "redis", func() (err error) {
		err = redisClient.Ping(ctx).Err()
		if err == nil {
			log.Info().Str("address", app.RedisOptions.Addrs[0]).Msg("Connected to Redis")
		}
		return err
	}))
	err := init.Wait()
	if err != nil {
		return err
	}

	db, err := database.NewDatabase(ctx, redisClient, rethinkDb)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	for _, worker := range []struct {
		name  string
		delay time.Duration
		fn    func() error
	}{
		{"update-job-executions", 5 * time.Second, updateJobExecutions(ctx, db)},
		{"ceid-timeout-queue", 1100 * time.Millisecond, crawlExecutionTimeoutQueueWorker(ctx, db)},
		{"remuri-queue", 200 * time.Millisecond, removeUriQueueWorker(ctx, db)},
		{"busy-queue", 50 * time.Millisecond, chgBusyQueueWorker(ctx, db)},
		{"wait-queue", 50 * time.Millisecond, chgWaitQueueWorker(ctx, db)},
		{"ceid-running-queue", 50 * time.Millisecond, crawlExecutionRunningQueueWorker(ctx, db)},
	} {
		log.Info().Dur("delayMs", worker.delay).Msgf("Starting worker: %s", worker.name)

		g.Go(func() error {
			for {
				// io.EOF can be returned by the go-redis driver but
				// is to be seen as transient
				err := worker.fn()
				if err != nil && !errors.Is(err, io.EOF) {
					return fmt.Errorf("%s: %w", worker.name, err)
				}
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(worker.delay):
				}
			}
		})
	}

	app.ready.Store(true)

	<-ctx.Done()

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
			log.Warn().Err(err).Dur("backoff", backoff).Str("service", name).Msg("Connection failed, retrying...")

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
