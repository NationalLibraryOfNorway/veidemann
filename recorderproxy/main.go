package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/mitmcert"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/recorderproxy"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/serviceconnections"
	"github.com/spf13/pflag"
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

	cacheAddr, err := cacheAddress(opts.CacheHost(), opts.CachePort())
	if err != nil {
		return err
	}

	err = logger.InitLog(opts.LogLevel(), opts.LogFormatter(), opts.LogMethod())
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	slog.Info(name, "version", version, "commit", commit, "date", date)

	mitmIdentity, err := mitmcert.LoadIdentity(opts.MITMCertFile(), opts.MITMKeyFile())
	if err != nil {
		return fmt.Errorf("failed to load MITM identity: %w", err)
	}

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

	slog.Info("Using cache", "address", cacheAddr)

	metricsAddr := net.JoinHostPort(opts.MetricsInterface(), strconv.Itoa(opts.MetricsPort()))
	metricsListener, err := net.Listen("tcp", metricsAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on metrics address %s: %w", metricsAddr, err)
	}
	metricsServer := newMetricsServer(metricsAddr, opts.MetricsPath())
	defer shutdownHTTPServer(metricsServer)

	go serveHTTP("Metrics", metricsServer, metricsListener)

	if opts.ProfilingEnabled() {
		profilingAddr := net.JoinHostPort(opts.ProfilingInterface(), strconv.Itoa(opts.ProfilingPort()))
		profilingListener, err := net.Listen("tcp", profilingAddr)
		if err != nil {
			return fmt.Errorf("failed to listen on profiling address %s: %w", profilingAddr, err)
		}
		profilingServer := newProfilingServer(profilingAddr)
		defer shutdownHTTPServer(profilingServer)

		go serveHTTP("Profiling", profilingServer, profilingListener)
	}

	iface := opts.Interface()
	firstPort := opts.Port()
	proxyCount := opts.ProxyCount()

	var startedProxies []*recorderproxy.RecorderProxy

	for i := range proxyCount {
		r := recorderproxy.NewRecorderProxy(
			i,
			conn,
			cacheAddr,
			recorderproxy.WithMITMIdentity(mitmIdentity),
			recorderproxy.WithFinalizationTimeout(opts.FinalizationTimeout()),
			recorderproxy.WithIdleTimeout(opts.IdleTimeout()),
		)

		ln, err := r.Listen(iface, firstPort)
		if err != nil {
			return fmt.Errorf("failed to listen (proxy %v): %w", i, err)
		}

		slog.Info("Proxy is listening ...", "id", i, "address", ln.Addr())

		go func() {
			err := r.Serve(ln)
			if err != nil {
				slog.Error("Recorder proxy stopped", "id", i, "error", err)
			}
		}()

		startedProxies = append(startedProxies, r)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("Server ready", "proxies", proxyCount)
	<-ctx.Done()
	slog.Info("Server shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), opts.FinalizationTimeout()+5*time.Second)
	defer shutdownCancel()

	shutdownRecorderProxies(shutdownCtx, startedProxies)

	return nil
}

func shutdownRecorderProxies(ctx context.Context, proxies []*recorderproxy.RecorderProxy) {
	var wg sync.WaitGroup
	for i, proxy := range proxies {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := proxy.Shutdown(ctx); err != nil {
				slog.Error("Error closing recorder proxy", "error", err, "id", i)
			}
		}()
	}
	wg.Wait()
}

func cacheAddress(host, port string) (string, error) {
	if host == "" || port == "" {
		return "", errors.New("both cache-host and cache-port are required")
	}
	return net.JoinHostPort(host, port), nil
}

func serveHTTP(name string, server *http.Server, listener net.Listener) {
	slog.Info(name+" server listening", "address", listener.Addr())
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error(name+" server stopped", "error", err)
	}
}

func shutdownHTTPServer(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Failed to shut down HTTP server", "address", server.Addr, "error", err)
	}
}
