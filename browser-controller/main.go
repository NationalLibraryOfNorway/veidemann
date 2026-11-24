/*
 * Copyright 2020 National Library of Norway.
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
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/controller"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/database"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/frontier"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/screenshotwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/uber/jaeger-client-go/config"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	name    = "browser-controller"
	version = ""
	commit  = ""
	date    = ""
)

const readyPath = "/readyz"

func main() {
	initLogger(os.Stdout, "INFO", false)
	err := run()
	if err != nil {
		serviceLogger().Error("Service failed", "error", err)
		os.Exit(1)
	}
	serviceLogger().Info("Service stopped")
}

func run() error {
	opts, err := parseFlags()
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if opts.Help() {
		pflag.Usage()
		return nil
	}
	if err := opts.ValidateCompletionTimeouts(); err != nil {
		return err
	}

	// init logger
	initLogger(
		os.Stdout,
		opts.LogLevel(),
		opts.LogMethod(),
	)

	fields := []any{}
	for _, field := range []struct{ key, value string }{{"version", version}, {"commit", commit}, {"date", date}} {
		if field.value != "" {
			fields = append(fields, field.key, field.value)
		}
	}
	serviceLogger().Info("Service starting", fields...)

	recorderCertificateSPKI := ""
	if opts.ProxyHost() != "" {
		recorderCertificateSPKI, err = loadRecorderCertificateSPKI(opts.RecorderCertFile(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to configure recorder certificate trust: %w", err)
		}
	}

	closer := initTracer(name)
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	contentWriterAddr := net.JoinHostPort(opts.ContentWriterHost(), strconv.Itoa(opts.ContentWriterPort()))
	contentWriterConn, err := grpc.NewClient(contentWriterAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to content writer at %s: %w", contentWriterAddr, err)
	}
	defer closeGRPCConnection("ContentWriter", contentWriterConn)
	screenshotWriter := screenshotwriter.New(contentwriterV1.NewContentWriterClient(contentWriterConn))

	tracer := opentracing.GlobalTracer()
	frontierAddr := net.JoinHostPort(opts.FrontierHost(), strconv.Itoa(opts.FrontierPort()))
	frontierConn, err := grpc.NewClient(
		frontierAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(tracer)),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to frontier at %s: %w", frontierAddr, err)
	}
	defer closeGRPCConnection("Frontier", frontierConn)
	frontierClient := frontier.New(frontierV1.NewFrontierClient(frontierConn))

	recorderAddr := net.JoinHostPort(opts.ProxyHost(), strconv.Itoa(opts.RecorderControlPort()))
	recorderConn, err := grpc.NewClient(recorderAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to recorder control at %s: %w", recorderAddr, err)
	}
	defer closeGRPCConnection("RecorderControl", recorderConn)
	recorder := recorderV1.NewRecorderControlClient(recorderConn)

	db := database.NewRethinkDbConnection(
		database.Options{
			Address:            net.JoinHostPort(opts.DBHost(), strconv.Itoa(opts.DBPort())),
			Username:           opts.DBUser(),
			Password:           opts.DBPassword(),
			Database:           opts.DBName(),
			QueryTimeout:       opts.DBQueryTimeout(),
			MaxOpenConnections: opts.DBMaxOpenConn(),
			MaxRetries:         opts.DBMaxRetries(),
			UseOpenTracing:     opts.DBUseOpentracing(),
		},
	)
	err = db.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() { _ = db.Close() }()

	configAdapter := database.NewConfigAdapter(db)

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	tokens, err := controller.RecoverSlots(rootCtx, recorder, opts.ProxyCount(), opts.CompletionDrainTimeout())
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(rootCtx)

	var ready atomic.Bool

	mux := http.NewServeMux()
	mux.HandleFunc(readyPath, func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle(opts.MetricsPath(), promhttp.Handler())

	sessionOptions := []session.Option{
		session.WithRecorder(recorder),
		session.WithCaptureTimeout(opts.CaptureTimeout()),
		session.WithPersistenceTimeout(opts.PersistenceTimeout()),
		session.WithScreenshotWriter(screenshotWriter),
		session.WithBrowserHost(opts.BrowserHost()),
		session.WithBrowserPort(opts.BrowserPort()),
		session.WithProxyHost(opts.ProxyHost()),
		session.WithProxyPort(opts.ProxyPort()),
		session.WithRecorderCertificateSPKI(recorderCertificateSPKI),
		session.WithConfigAdapter(configAdapter),
		session.WithMediaSettleTimeout(opts.MediaSettleTimeout()),
		session.WithCompletionDrainTimeout(opts.CompletionDrainTimeout()),
	}

	telemetryAddr := net.JoinHostPort(opts.MetricsInterface(), strconv.Itoa(opts.MetricsPort()))
	telemetryListener, err := net.Listen("tcp", telemetryAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on telemetry address %s: %w", telemetryAddr, err)
	}
	defer func() { _ = telemetryListener.Close() }()

	serviceLogger().Log(ctx, session.LevelTrace, "Listener ready", "phase", "telemetry", "address", telemetryAddr)

	telemetryServer := &http.Server{
		Addr:    telemetryAddr,
		Handler: mux,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	// Start telemetry server
	g.Go(func() error {
		err := telemetryServer.Serve(telemetryListener)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	mainLoop := controller.New(tokens, frontierClient, opts.FetchTimeout(), opts.ReportTimeout(), sessionOptions...)
	mainLoopDone := make(chan struct{})

	// Browser controller main loop
	g.Go(func() error {
		defer close(mainLoopDone)
		err := mainLoop.Run(ctx)

		if errors.Is(err, context.Canceled) {
			return nil
		}

		return err
	})

	ready.Store(true)
	serviceLogger().Info("Service ready", "telemetryAddress", telemetryAddr, "slots", opts.ProxyCount())

	<-ctx.Done()

	ready.Store(false)
	serviceLogger().Info("Service stopping")

	// Run stops scheduling and joins its active fetches. Their recorder RPCs
	// must remain available through browser shutdown and completion draining.
	<-mainLoopDone

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = telemetryServer.Shutdown(shutdownCtx)

	return g.Wait()
}

func closeGRPCConnection(service string, conn *grpc.ClientConn) {
	if err := conn.Close(); err != nil {
		serviceLogger().Error("Failed to close gRPC connection", "service", service, "error", err)
	}
}

func loadRecorderCertificateSPKI(path string, now time.Time) (string, error) {
	if path == "" {
		return "", fmt.Errorf("recorder certificate file is required when proxying is enabled")
	}
	certPEM, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read recorder certificate: %w", err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", fmt.Errorf("recorder certificate file does not contain a PEM certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse recorder certificate: %w", err)
	}
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return "", fmt.Errorf("recorder certificate is not currently valid")
	}
	if cert.IsCA {
		return "", fmt.Errorf("recorder certificate must be a server leaf, not a CA")
	}
	if len(cert.ExtKeyUsage) > 0 {
		serverAuth := false
		for _, usage := range cert.ExtKeyUsage {
			if usage == x509.ExtKeyUsageAny || usage == x509.ExtKeyUsageServerAuth {
				serverAuth = true
				break
			}
		}
		if !serverAuth {
			return "", fmt.Errorf("recorder certificate is not valid for TLS server authentication")
		}
	}
	digest := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	return base64.StdEncoding.EncodeToString(digest[:]), nil
}

func initLogger(w io.Writer, level string, source bool) {
	levelVar := new(slog.LevelVar)
	levelVar.Set(toLogLevel(level))

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		AddSource: source,
		Level:     levelVar,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.LevelKey && attr.Value.Any() == session.LevelTrace {
				return slog.String(slog.LevelKey, "TRACE")
			}
			return attr
		},
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Redirect package-level log.Print/log.Printf/etc. to the same slog handler.
	log.SetOutput(slog.NewLogLogger(handler, slog.LevelInfo).Writer())
	log.SetFlags(0)
}

func toLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "trace":
		return session.LevelTrace
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

// initTracer initializes the global OpenTracing tracer using Jaeger configuration from environment variables.
func initTracer(service string) io.Closer {
	cfg, err := config.FromEnv()
	if err != nil {
		return nil
	}

	if cfg.ServiceName == "" {
		cfg.ServiceName = service
	}

	tracer, closer, err := cfg.NewTracer()
	if err == nil {
		opentracing.SetGlobalTracer(tracer)
	}

	return closer
}

func serviceLogger() *slog.Logger { return slog.Default().With("component", name) }
