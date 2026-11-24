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
	"strings"
	"sync"
	"syscall"
	"time"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	dnsresolverV1 "github.com/NationalLibraryOfNorway/veidemann/api/dnsresolver/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	robotsV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator/v1"
	"github.com/NationalLibraryOfNorway/veidemann/log-service/pkg/logservice"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/mitmcert"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/recorderproxy"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	cacheAddr, err := cacheAddress(opts.CacheHost(), opts.CachePort())
	if err != nil {
		return err
	}

	err = logger.InitLog(opts.LogLevel(), opts.LogFormatter(), opts.LogMethod())
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	fields := []any{}
	for _, field := range []struct{ key, value string }{{"version", version}, {"commit", commit}, {"date", date}} {
		if field.value != "" {
			fields = append(fields, field.key, field.value)
		}
	}
	serviceLogger().Info("Service starting", fields...)

	mitmIdentity, err := mitmcert.LoadIdentity(opts.MITMCertFile(), opts.MITMKeyFile())
	if err != nil {
		return fmt.Errorf("failed to load MITM identity: %w", err)
	}

	contentWriterAddr := net.JoinHostPort(opts.ContentWriterHost(), opts.ContentWriterPort())
	contentWriterConn, err := grpc.NewClient(contentWriterAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to content writer at %s: %w", contentWriterAddr, err)
	}
	defer closeGRPCConnection("ContentWriter", contentWriterConn)
	contentWriter := contentwriterV1.NewContentWriterClient(contentWriterConn)

	dnsResolverAddr := net.JoinHostPort(opts.DnsResolverHost(), opts.DnsResolverPort())
	dnsResolverConn, err := grpc.NewClient(dnsResolverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to DNS resolver at %s: %w", dnsResolverAddr, err)
	}
	defer closeGRPCConnection("DnsResolver", dnsResolverConn)
	dnsResolver := dnsresolverV1.NewDnsResolverClient(dnsResolverConn)

	robotsEvaluatorAddr := net.JoinHostPort(opts.RobotsEvaluatorHost(), opts.RobotsEvaluatorPort())
	robotsEvaluatorConn, err := grpc.NewClient(robotsEvaluatorAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to robots evaluator at %s: %w", robotsEvaluatorAddr, err)
	}
	defer closeGRPCConnection("RobotsEvaluator", robotsEvaluatorConn)
	robotsEvaluator := robotsV1.NewRobotsEvaluatorClient(robotsEvaluatorConn)

	logServiceAddr := net.JoinHostPort(opts.LogServiceHost(), opts.LogServicePort())
	logServiceConn, err := grpc.NewClient(logServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to log service at %s: %w", logServiceAddr, err)
	}
	defer closeGRPCConnection("LogService", logServiceConn)
	logs := &logservice.LogWriter{LogClient: logV1.NewLogClient(logServiceConn)}

	serviceLogger().Log(context.Background(), logger.TraceLevel, "Cache configured", "cacheAddress", cacheAddr)

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
	var proxyAddresses []string

	if proxyCount <= 0 {
		return fmt.Errorf("proxy-count must be positive")
	}
	for i := 1; i <= proxyCount; i++ {
		r := recorderproxy.NewRecorderProxy(
			i,
			contentWriter,
			dnsResolver,
			cacheAddr,
			recorderproxy.WithMITMIdentity(mitmIdentity),
			recorderproxy.WithRobotsEvaluator(robotsEvaluator),
			recorderproxy.WithFinalizationTimeout(opts.FinalizationTimeout()),
			recorderproxy.WithIdleTimeout(opts.IdleTimeout()),
		)

		ln, err := r.Listen(iface, firstPort)
		if err != nil {
			return fmt.Errorf("failed to listen (proxy %v): %w", i, err)
		}

		proxyAddresses = append(proxyAddresses, ln.Addr().String())

		go func() {
			err := r.Serve(ln)
			if err != nil {
				serviceLogger().Error("Recorder proxy stopped", "slot", i, "error", err)
			}
		}()

		startedProxies = append(startedProxies, r)
	}

	controlAddr := net.JoinHostPort(opts.ControlInterface(), opts.ControlPort())
	controlListener, err := net.Listen("tcp", controlAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on recorder control address %s: %w", controlAddr, err)
	}
	controlServer := grpc.NewServer()
	recorderV1.RegisterRecorderControlServer(controlServer, recorderproxy.NewControl(logs, startedProxies...))
	defer controlServer.Stop()
	go func() {
		if err := controlServer.Serve(controlListener); err != nil {
			serviceLogger().Error("Recorder control stopped", "error", err)
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serviceLogger().Info("Service ready", "slots", proxyCount, "proxyAddresses", proxyAddresses, "controlAddress", controlListener.Addr().String(), "metricsAddress", metricsListener.Addr().String(), "cacheAddress", cacheAddr)
	<-ctx.Done()
	serviceLogger().Info("Service stopping")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), opts.FinalizationTimeout()+5*time.Second)
	defer shutdownCancel()

	shutdownRecorderProxies(shutdownCtx, startedProxies)

	return nil
}

func closeGRPCConnection(service string, conn *grpc.ClientConn) {
	if err := conn.Close(); err != nil {
		serviceLogger().Error("Failed to close gRPC connection", "service", service, "error", err)
	}
}

func shutdownRecorderProxies(ctx context.Context, proxies []*recorderproxy.RecorderProxy) {
	var wg sync.WaitGroup
	for i, proxy := range proxies {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := proxy.Shutdown(ctx); err != nil {
				serviceLogger().Error("Error closing recorder proxy", "error", err, "slot", i+1)
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
	serviceLogger().Log(context.Background(), logger.TraceLevel, "Listener ready", "phase", strings.ToLower(name), "address", listener.Addr().String())
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		serviceLogger().Error(name+" server stopped", "error", err)
	}
}

func shutdownHTTPServer(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		serviceLogger().Error("Failed to shut down HTTP server", "address", server.Addr, "error", err)
	}
}

func serviceLogger() *slog.Logger { return slog.Default().With("component", name) }
