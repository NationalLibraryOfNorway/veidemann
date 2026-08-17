package main

import (
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Options struct {
	v *viper.Viper
}

func parseFlags() (Options, error) {
	flags := pflag.CommandLine

	flags.String("interface", "", "interface this proxy listens to. No value means all interfaces.")
	flags.Int("port", 8080, "first proxy listen port")
	flags.Int("proxy-count", 10, "number of proxies to start")
	flags.String("content-writer-host", "localhost", "Content writer host")
	flags.String("content-writer-port", "7777", "Content writer port")
	flags.String("dns-resolver-host", "localhost", "DNS resolver host")
	flags.String("dns-resolver-port", "7778", "DNS resolver port")
	flags.String("browser-controller-host", "localhost", "Browser controller host")
	flags.String("browser-controller-port", "7779", "Browser controller port")
	flags.Duration("timeout", 10*time.Minute, "Timeout used for connecting to GRPC services")
	flags.String("mitm-cert-file", "", "Path to the fixed MITM server certificate")
	flags.String("mitm-key-file", "", "Path to the fixed MITM private key")
	flags.Duration("finalization-timeout", 30*time.Second, "Maximum time for ContentWriter terminal operations")
	flags.Duration("idle-timeout", 2*time.Minute, "Maximum time to wait for downstream request headers; non-positive values disable it")
	flags.String("cache-host", "", "Cache host")
	flags.String("cache-port", "", "Cache port")
	flags.String("metrics-interface", "", "interface for exposing Prometheus metrics. Empty means all interfaces")
	flags.Int("metrics-port", 9302, "port for exposing Prometheus metrics")
	flags.String("metrics-path", "/metrics", "path for exposing Prometheus metrics")
	flags.Bool("profiling-enabled", false, "enable the Go pprof HTTP server")
	flags.String("profiling-interface", "127.0.0.1", "interface for exposing the pprof HTTP server")
	flags.Int("profiling-port", 6060, "port for exposing the pprof HTTP server")
	flags.String("log-level", "info", "log level, available levels are panic, fatal, error, warn, info, debug and trace")
	flags.String("log-formatter", "text", "log formatter, available values are text, logfmt and json")
	flags.Bool("log-method", false, "log method name")

	pflag.Parse()

	v := viper.New()
	v.SetDefault("ContentDir", "content")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	err := v.BindPFlags(flags)
	if err != nil {
		return Options{}, err
	}

	return Options{v: v}, nil
}

func (o Options) Interface() string {
	return o.v.GetString("interface")
}

func (o Options) Port() int {
	return o.v.GetInt("port")
}

func (o Options) ProxyCount() int {
	return o.v.GetInt("proxy-count")
}

func (o Options) MetricsInterface() string {
	return o.v.GetString("metrics-interface")
}

func (o Options) MetricsPort() int {
	return o.v.GetInt("metrics-port")
}

func (o Options) MetricsPath() string {
	return o.v.GetString("metrics-path")
}

func (o Options) ProfilingEnabled() bool {
	return o.v.GetBool("profiling-enabled")
}

func (o Options) ProfilingInterface() string {
	return o.v.GetString("profiling-interface")
}

func (o Options) ProfilingPort() int {
	return o.v.GetInt("profiling-port")
}

func (o Options) Help() bool {
	return o.v.GetBool("help")
}

func (o Options) LogLevel() string {
	return o.v.GetString("log-level")
}

func (o Options) LogFormatter() string {
	return o.v.GetString("log-formatter")
}

func (o Options) LogMethod() bool {
	return o.v.GetBool("log-method")
}

func (o Options) ContentWriterHost() string {
	return o.v.GetString("content-writer-host")
}

func (o Options) ContentWriterPort() string {
	return o.v.GetString("content-writer-port")
}

func (o Options) DnsResolverHost() string {
	return o.v.GetString("dns-resolver-host")
}

func (o Options) DnsResolverPort() string {
	return o.v.GetString("dns-resolver-port")
}

func (o Options) BrowserControllerHost() string {
	return o.v.GetString("browser-controller-host")
}

func (o Options) BrowserControllerPort() string {
	return o.v.GetString("browser-controller-port")
}

func (o Options) ConnectTimeout() time.Duration {
	return o.v.GetDuration("timeout")
}

func (o Options) CacheHost() string {
	return o.v.GetString("cache-host")
}

func (o Options) CachePort() string {
	return o.v.GetString("cache-port")
}

func (o Options) MITMCertFile() string {
	return o.v.GetString("mitm-cert-file")
}

func (o Options) MITMKeyFile() string {
	return o.v.GetString("mitm-key-file")
}

func (o Options) FinalizationTimeout() time.Duration {
	return o.v.GetDuration("finalization-timeout")
}

func (o Options) IdleTimeout() time.Duration {
	return o.v.GetDuration("idle-timeout")
}
