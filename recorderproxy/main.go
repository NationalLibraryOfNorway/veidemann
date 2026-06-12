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

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/recorderproxy"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/serviceconnections"
	"github.com/opentracing/opentracing-go"
	"github.com/spf13/pflag"
	"github.com/uber/jaeger-client-go/config"
)

var (
	name    = "recorderproxy"
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
	opts, err := parseFlags()
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if opts.Help() {
		pflag.Usage()
		return nil
	}

	err = logger.InitLog(opts.LogLevel(), opts.LogFormatter(), opts.LogMethod())
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	slog.Info(name, "version", version, "commit", commit, "date", date)

	closer := initTracer(name)
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	//err := recorderproxy.SetCA(viper.GetString("ca"), viper.GetString("ca-key"))
	//if err != nil {
	//	log.Fatal(err)
	//}

	contentWriterOpts := serviceconnections.NewConnectionOptions(
		"ContentWriter",
		serviceconnections.WithHost(opts.ContentWriterHost()),
		serviceconnections.WithPort(opts.ContentWriterPort()),
	)
	dnsOpts := serviceconnections.NewConnectionOptions(
		"DnsService",
		serviceconnections.WithHost(opts.DnsResolverHost()),
		serviceconnections.WithPort(opts.DnsResolverPort()),
	)
	browserControllerOpts := serviceconnections.NewConnectionOptions(
		"BrowserController",
		serviceconnections.WithHost(opts.BrowserControllerHost()),
		serviceconnections.WithPort(opts.BrowserControllerPort()),
	)

	conn := serviceconnections.NewConnections(contentWriterOpts, dnsOpts, browserControllerOpts)
	defer func() {
		err := conn.Close()
		if err != nil {
			slog.Error("Error closing gRPC connections", "error", err)
		}
	}()

	err = conn.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to services: %w", err)
	}

	cacheAddr := opts.CacheHost() + ":" + opts.CachePort()
	slog.Info("Using cache", "address", cacheAddr)

	iface := opts.Interface()
	firstPort := opts.Port()
	proxyCount := opts.ProxyCount()

	var startedProxies []*recorderproxy.RecorderProxy

	for i := range proxyCount {
		r, err := recorderproxy.NewRecorderProxy(i, iface, firstPort, conn, cacheAddr)
		if err != nil {
			return fmt.Errorf("failed to create recorder proxy %v: %w", i, err)
		}
		slog.Info("Proxy is listening ...", "id", i, "address", r.Addr)

		go func() {
			err := r.Start()
			if err != nil {
				log.Printf("Recorder proxy %d stopped: %v", i, err)
			}
		}()

		startedProxies = append(startedProxies, r)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("Server ready", "proxies", proxyCount)
	<-ctx.Done()
	slog.Info("Server shutting down")

	for i, r := range startedProxies {
		err = r.Close()
		if err != nil {
			slog.Error("Error closing recorder proxy", "error", err, "id", i)
		}
	}

	return nil
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
