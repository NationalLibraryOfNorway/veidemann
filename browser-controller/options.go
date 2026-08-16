package main

import (
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func parseFlags() (Options, error) {
	flags := pflag.CommandLine

	flags.BoolP("help", "h", false, "Usage instructions")
	flags.String("interface", "", "interface the browser controller api listens to. No value means all interfaces.")
	flags.Int("port", 8080, "port the browser controller api listens to.")

	flags.Int("proxy-count", 2, "max number of simultaneous sessions. Must match RecorderProxy's proxy-count setting.")
	flags.Duration("fetch-timeout", 5*time.Minute, "Timeout for fetching a page in the browser")
	flags.Duration("report-timeout", 1*time.Minute, "Timeout for reporting fetch result to Frontier")

	flags.String("browser-host", "localhost", "Browser host")
	flags.Int("browser-port", 3000, "Browser port")

	flags.String("proxy-host", "localhost", "Recorder proxy host")
	flags.Int("proxy-port", 9900, "Recorder proxy port")
	flags.String("recorder-cert-file", "", "Recorder proxy public certificate used to scope Chromium TLS interception")

	flags.String("content-writer-host", "veidemann-contentwriter", "Content writer host")
	flags.Int("content-writer-port", 8082, "Content writer port")

	flags.String("frontier-host", "veidemann-frontier", "Frontier host")
	flags.Int("frontier-port", 7700, "Frontier port")

	flags.String("log-service-host", "veidemann-log-service", "Log service host")
	flags.Int("log-service-port", 8080, "Log service port")

	flags.String("robots-evaluator-host", "veidemann-robotsevaluator-service", "Robots evaluator host")
	flags.Int("robots-evaluator-port", 7053, "Robots evaluator port")

	flags.String("db-host", "rethinkdb-proxy", "DB host")
	flags.Int("db-port", 28015, "DB port")
	flags.String("db-name", "veidemann", "DB name")
	flags.String("db-user", "", "Database username")
	flags.String("db-password", "", "Database password")
	flags.Duration("db-query-timeout", 1*time.Minute, "Database query timeout")
	flags.Int("db-max-retries", 5, "Max retries when database query fails")
	flags.Int("db-max-open-conn", 10, "Max open database connections")
	flags.Bool("db-use-opentracing", false, "Use opentracing for database queries")

	flags.String("metrics-interface", "", "Interface for exposing metrics. Empty means all interfaces")
	flags.Int("metrics-port", 9153, "Port for exposing metrics")
	flags.String("metrics-path", "/metrics", "Path for exposing metrics")

	flags.String("log-level", "info", "log level, available levels are panic, fatal, error, warn, info, debug and trace")
	flags.Bool("log-method", false, "log method names")

	pflag.Parse()

	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	err := v.BindPFlags(flags)
	if err != nil {
		return Options{}, err
	}

	return Options{v: v}, nil
}

type Options struct {
	v *viper.Viper
}

func (o Options) Help() bool {
	return o.v.GetBool("help")
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

func (o Options) FetchTimeout() time.Duration {
	return o.v.GetDuration("fetch-timeout")
}

func (o Options) ReportTimeout() time.Duration {
	return o.v.GetDuration("report-timeout")
}

func (o Options) BrowserHost() string {
	return o.v.GetString("browser-host")
}

func (o Options) BrowserPort() int {
	return o.v.GetInt("browser-port")
}

func (o Options) ProxyHost() string {
	return o.v.GetString("proxy-host")
}

func (o Options) ProxyPort() int {
	return o.v.GetInt("proxy-port")
}

func (o Options) RecorderCertFile() string {
	return o.v.GetString("recorder-cert-file")
}

func (o Options) ContentWriterHost() string {
	return o.v.GetString("content-writer-host")
}

func (o Options) ContentWriterPort() int {
	return o.v.GetInt("content-writer-port")
}

func (o Options) FrontierHost() string {
	return o.v.GetString("frontier-host")
}

func (o Options) FrontierPort() int {
	return o.v.GetInt("frontier-port")
}

func (o Options) LogServiceHost() string {
	return o.v.GetString("log-service-host")
}

func (o Options) LogServicePort() int {
	return o.v.GetInt("log-service-port")
}

func (o Options) RobotsEvaluatorHost() string {
	return o.v.GetString("robots-evaluator-host")
}

func (o Options) RobotsEvaluatorPort() int {
	return o.v.GetInt("robots-evaluator-port")
}

func (o Options) DBHost() string {
	return o.v.GetString("db-host")
}

func (o Options) DBPort() int {
	return o.v.GetInt("db-port")
}

func (o Options) DBName() string {
	return o.v.GetString("db-name")
}

func (o Options) DBUser() string {
	return o.v.GetString("db-user")
}

func (o Options) DBPassword() string {
	return o.v.GetString("db-password")
}

func (o Options) DBQueryTimeout() time.Duration {
	return o.v.GetDuration("db-query-timeout")
}

func (o Options) DBMaxRetries() int {
	return o.v.GetInt("db-max-retries")
}

func (o Options) DBMaxOpenConn() int {
	return o.v.GetInt("db-max-open-conn")
}

func (o Options) DBUseOpentracing() bool {
	return o.v.GetBool("db-use-opentracing")
}

func (o Options) DBCacheTTL() time.Duration {
	return o.v.GetDuration("db-cache-ttl")
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

func (o Options) LogLevel() string {
	return o.v.GetString("log-level")
}

func (o Options) LogMethod() bool {
	return o.v.GetBool("log-method")
}
