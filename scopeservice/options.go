package main

import (
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	readyPath   = "/readyz"
	metricsPath = "/metrics"
)

type Options struct{}

func (o Options) LogLevel() string {
	return viper.GetString("log-level")
}

func (o Options) LogFormatter() string {
	return viper.GetString("log-formatter")
}

func (o Options) LogMethod() bool {
	return viper.GetBool("log-method")
}

func (o Options) IncludeFragments() bool {
	return viper.GetBool("include-fragment")
}

func (o Options) Address() string {
	return viper.GetString("address")
}

func (o Options) TelemetryAddr() string {
	return viper.GetString("metrics-address")
}

func (o Options) MetricsPath() string {
	return metricsPath
}

func (o Options) ReadyPath() string {
	return readyPath
}

func parseFlags() error {
	flags := pflag.CommandLine

	pflag.String("address", ":8080", "Interface for gRPC server. Empty means all interfaces")
	pflag.String("metrics-address", ":9153", "Interface for telemetry server. Empty means all interfaces")

	pflag.String("log-level", "info", "log level, available levels are error, warn, info and debug")
	pflag.String("log-format", "json", "log format, available values are json and text")
	pflag.Bool("log-method", false, "log method names")

	pflag.Bool("include-fragment", false, "if true, do not remove fragment from URI during canonicalization.")

	pflag.Parse()

	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()

	return viper.BindPFlags(flags)
}
