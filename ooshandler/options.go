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

func (o Options) LogMethod() bool {
	return viper.GetBool("log-method")
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

func (o Options) DataDir() string {
	return viper.GetString("data-dir")
}

func parseFlags(args []string) error {
	flags := pflag.NewFlagSet(name, pflag.ContinueOnError)

	flags.String("address", ":50052", "address and Port to bind GRPC service, in host:port format")
	flags.String("metrics-address", ":9301", "address and Port to bind prometheus exporter, in host:port format")

	flags.String("log-level", "info", "log level, available levels are error, warn, info and debug")
	flags.Bool("log-method", false, "log method names")

	flags.String("data-dir", "/data", "directory to store new seeds")
	if err := flags.Parse(args); err != nil {
		return err
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	return viper.BindPFlags(flags)
}
