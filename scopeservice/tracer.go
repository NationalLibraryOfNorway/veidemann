package main

import (
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/log"
)

// initTracer initializes a Jaeger tracer based on environment variables and sets it as the global tracer.
func initTracer(service string) (io.Closer, error) {
	cfg, err := config.FromEnv()
	if err != nil {
		return nil, err
	}

	if cfg.ServiceName == "" {
		cfg.ServiceName = service
	}

	tracer, closer, err := cfg.NewTracer(config.Logger(log.StdLogger))
	if err != nil {
		return nil, err
	}
	opentracing.SetGlobalTracer(tracer)

	return closer, nil
}
