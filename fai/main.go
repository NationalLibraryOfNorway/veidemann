/*
 * Copyright 2023 National Library of Norway.
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
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nationallibraryofnorway/veidemann/fai/internal/fai"
	"github.com/nationallibraryofnorway/veidemann/fai/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	name    = "fai"
)

func main() {
	pflag.String("dir", "", "path to source directory")
	pflag.String("pattern", "*.warc.gz", "glob pattern used to match filenames in source directory")
	pflag.Duration("sleep", 5*time.Second, "sleep duration between directory listings, set to 0 to only do a single pass")
	pflag.String("s3-address", "localhost:9000", "s3 endpoint (address:port)")
	pflag.String("s3-bucket-name", "", "name of bucket to upload files to")
	pflag.String("s3-access-key-id", "", "access key ID")
	pflag.String("s3-secret-access-key", "", "secret access key")
	pflag.String("s3-token", "", "token to use for s3 authentication (optional)")
	pflag.String("metrics-path", "/metrics", "path to expose metrics on")
	pflag.String("metrics-address", ":8081", "address to expose metrics on")
	pflag.Parse()

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))

	slog.Info(name, "version", version, "commit", commit, "date", date, "name", name)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err := run(ctx)
	if err != nil {
		slog.Error("Goodbye", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		return fmt.Errorf("failed to bind command line flags: %w", err)
	}

	sourceDir := viper.GetString("dir")
	sleep := viper.GetDuration("sleep")
	globPattern := viper.GetString("pattern")
	metricsAddr := viper.GetString("metrics-address")
	metricsPath := viper.GetString("metrics-path")
	s3bucketName := viper.GetString("s3-bucket-name")
	s3address := viper.GetString("s3-address")
	s3accessKeyID := viper.GetString("s3-access-key-id")
	s3secretAccessKey := viper.GetString("s3-secret-access-key")
	s3token := viper.GetString("s3-token")

	s3uploader, err := fai.NewS3Uploader(
		fai.WithS3Address(s3address),
		fai.WithS3AccessKeyID(s3accessKeyID),
		fai.WithS3SecretAccessKey(s3secretAccessKey),
		fai.WithS3Token(s3token),
		fai.WithS3BucketName(s3bucketName),
	)
	if err != nil {
		return fmt.Errorf("failed to create S3 uploader: %w", err)
	}

	slog.Info("S3 uploader", "bucket", s3bucketName, "address", s3address)

	worker := func(ctx context.Context, filePath string) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		slog.Info("Uploading file to S3", "path", filePath)

		start := time.Now()
		info, err := s3uploader.Upload(ctx, filePath)
		if err != nil {
			return fmt.Errorf("failed to upload file to S3: %s: %w", filePath, err)
		}
		metrics.Duration(time.Since(start))
		metrics.Size(info.Size)

		slog.Info("Uploaded file",
			"key", info.Key,
			"size", info.Size,
			"etag", info.ETag,
			"duration", time.Since(start).String(),
		)
		err = os.Remove(filePath)
		if err != nil {
			return fmt.Errorf("failed to remove file after upload: %s: %w", filePath, err)
		}
		return nil
	}

	f, err := fai.New(
		fai.WithSourceDir(sourceDir),
		fai.WithSleep(sleep),
		fai.WithGlobPattern(globPattern),
		fai.WithInspector(worker),
	)
	if err != nil {
		return fmt.Errorf("failed to create FAI: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.Handler())
	telemtryServer := &http.Server{
		Addr:    metricsAddr,
		Handler: mux,
	}

	slog.Info("Starting metrics server", "address", metricsAddr)

	g.Go(func() error {
		err := telemtryServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	slog.Info("Starting FAI",
		"sourceDir", sourceDir,
		"globPattern", globPattern,
		"sleep", sleep.String())

	g.Go(func() error { return f.Run(ctx) })

	<-ctx.Done()

	slog.Warn("Shutting down")
	_ = telemtryServer.Shutdown(context.Background())

	return g.Wait()
}
