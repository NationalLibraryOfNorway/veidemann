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

func (o Options) MetricsPath() string {
	return metricsPath
}

func (o Options) ReadyPath() string {
	return readyPath
}

func (o Options) DBHost() string {
	return viper.GetString("db-host")
}

func (o Options) DBPort() int {
	return viper.GetInt("db-port")
}

func (o Options) DBName() string {
	return viper.GetString("db-name")
}

func (o Options) DBUsername() string {
	return viper.GetString("db-username")
}

func (o Options) DBPassword() string {
	return viper.GetString("db-password")
}

func (o Options) FrontierHost() string {
	return viper.GetString("frontier-host")
}

func (o Options) FrontierPort() int {
	return viper.GetInt("frontier-port")
}

func parseFlags() error {
	flags := pflag.CommandLine

	pflag.String("address", ":9301", "Address to listen on")

	pflag.String("db-host", "rethinkdb", "Database host")
	pflag.Int("db-port", 28015, "Database port")
	pflag.String("db-name", "veidemann", "Database name")
	pflag.String("db-username", "admin", "Database username")
	pflag.String("db-password", "", "Database password")

	pflag.String("frontier-host", "veidemann-frontier", "Frontier host")
	pflag.Int("frontier-port", 7700, "Frontier port")

	pflag.String("log-level", "info", "log level, available levels are error, warn, info anddebug")
	pflag.Bool("log-method", false, "log method names")

	pflag.Parse()

	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()

	return viper.BindPFlags(flags)
}
