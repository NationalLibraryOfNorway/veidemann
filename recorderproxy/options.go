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
	flags.String("ca", "", "Path to CA certificate used for signing client connections")
	flags.String("ca-key", "", "Path to private key for CA certificate used for signing client connections")
	flags.String("cache-host", "", "Cache host")
	flags.String("cache-port", "", "Cache port")
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

func (o Options) CA() string {
	return o.v.GetString("ca")
}

func (o Options) CAKey() string {
	return o.v.GetString("ca-key")
}
