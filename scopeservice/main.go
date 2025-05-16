package main

import (
	"os"
	"os/signal"
	"syscall"
	"github.com/NationalLibraryOfNorway/veidemann/scopeservice/pkg/logger"
	"github.com/NationalLibraryOfNorway/veidemann/scopeservice/pkg/script"
	"github.com/NationalLibraryOfNorway/veidemann/scopeservice/pkg/server"
	"github.com/NationalLibraryOfNorway/veidemann/scopeservice/pkg/telemetry"

	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"

	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	pflag.String("interface", "", "interface the browser controller api listens to. No value means all interfaces.")
	pflag.Int("port", 8080, "port the browser controller api listens to.")
	pflag.Bool("include-fragment", false, "if true, do not remove fragment from URI during canonicalization.")

	pflag.String("metrics-interface", "", "Interface for exposing metrics. Empty means all interfaces")
	pflag.Int("metrics-port", 9153, "Port for exposing metrics")
	pflag.String("metrics-path", "/metrics", "Path for exposing metrics")

	pflag.String("log-level", "info", "log level, available levels are panic, fatal, error, warn, info, debug and trace")
	pflag.String("log-formatter", "logfmt", "log formatter, available values are logfmt and json")
	pflag.Bool("log-method", false, "log method names")

	pflag.Parse()
	_ = viper.BindPFlags(pflag.CommandLine)

	viper.SetDefault("ContentDir", "content")
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not parse flags")
	}

	logger.InitLog(viper.GetString("log-level"), viper.GetString("log-formatter"), viper.GetBool("log-method"))

	script.InitializeCanonicalizationProfiles(viper.GetBool("include-fragment"))
	scopeservice := server.New(viper.GetString("interface"), viper.GetInt("port"))

	// telemetry setup
	tracer, closer := telemetry.InitTracer("Scope checker")
	if tracer != nil {
		opentracing.SetGlobalTracer(tracer)
		defer closer.Close()
	}

	errc := make(chan error, 1)

	ms := telemetry.NewMetricsServer(viper.GetString("metrics-interface"), viper.GetInt("metrics-port"), viper.GetString("metrics-path"))
	go func() { errc <- ms.Start() }()
	defer ms.Close()

	go func() {
		signals := make(chan os.Signal, 2)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

		select {
		case err := <-errc:
			log.Err(err).Msg("Metrics server failed")
			scopeservice.Shutdown()
		case sig := <-signals:
			log.Debug().Msgf("Received signal: %scopeservice", sig)
			scopeservice.Shutdown()
		}
	}()

	err = scopeservice.Start()
	if err != nil {
		log.Err(err).Msg("")
	}
}
