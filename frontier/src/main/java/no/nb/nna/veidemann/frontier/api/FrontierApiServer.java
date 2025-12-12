/*
 * Copyright 2017 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package no.nb.nna.veidemann.frontier.api;

import java.io.IOException;
import java.io.UncheckedIOException;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import io.grpc.Server;
import io.grpc.ServerBuilder;
import io.grpc.ServerInterceptors;
import io.grpc.protobuf.services.HealthStatusManager;
import io.opentracing.contrib.grpc.TracingServerInterceptor;
import io.opentracing.contrib.grpc.TracingServerInterceptor.ServerRequestAttribute;
import no.nb.nna.veidemann.frontier.worker.Frontier;

public class FrontierApiServer implements AutoCloseable {

    private static final Logger LOG = LoggerFactory.getLogger(FrontierApiServer.class);

    private final Server server;
    private final ScheduledExecutorService healthCheckerExecutorService;
    private long shutdownTimeoutMillis = 60 * 1000;
    final FrontierGrpcService frontierService;
    final HealthStatusManager health;

    public FrontierApiServer(int port, int shutdownTimeoutSeconds, Frontier frontier) {
        this(ServerBuilder.forPort(port), shutdownTimeoutSeconds, frontier);
    }

    public FrontierApiServer(ServerBuilder<?> serverBuilder, int shutdownTimeoutSeconds, Frontier frontier) {
        this.shutdownTimeoutMillis = shutdownTimeoutSeconds * 1000L;

        TracingServerInterceptor tracingInterceptor = TracingServerInterceptor
                .newBuilder()
                .withTracer(frontier.getTracer())
                .withStreaming()
                .withTracedAttributes(ServerRequestAttribute.CALL_ATTRIBUTES, ServerRequestAttribute.METHOD_TYPE)
                .build();

        healthCheckerExecutorService = Executors.newScheduledThreadPool(1);
        health = new HealthStatusManager();
        healthCheckerExecutorService.scheduleAtFixedRate(
                new HealthChecker(frontier, health), 0, 1, TimeUnit.SECONDS);

        frontierService = new FrontierGrpcService(frontier);

        SimpleConcurrencyLimitInterceptor limitInterceptor = new SimpleConcurrencyLimitInterceptor(
                /* pick a number */ 100);

        server = serverBuilder
                .addService(ServerInterceptors.intercept(
                        frontierService,
                        limitInterceptor,
                        tracingInterceptor))
                .addService(health.getHealthService())
                .build();
    }

    public FrontierApiServer start() {
        try {
            server.start();
            LOG.info("Frontier gRPC server listening on {}", server.getPort());
            return this;
        } catch (IOException ex) {
            shutdown();
            throw new UncheckedIOException(ex);
        }
    }

    public void shutdown() {
        LOG.info("Shutting down FrontierApiServer");
        health.enterTerminalState();

        long startTime = System.currentTimeMillis();
        server.shutdown();
        frontierService.shutdown();

        long remaining = shutdownTimeoutMillis - (System.currentTimeMillis() - startTime);
        if (remaining < 0) {
            remaining = 0;
        }

        try {
            server.awaitTermination(remaining, TimeUnit.MILLISECONDS);
        } catch (InterruptedException e) {
            LOG.warn("Interrupted while waiting for gRPC server shutdown", e);
            Thread.currentThread().interrupt();
        }

        remaining = shutdownTimeoutMillis - (System.currentTimeMillis() - startTime);
        if (remaining < 0) {
            remaining = 0;
        }

        try {
            boolean gracefulStop = frontierService.awaitTermination(remaining, TimeUnit.MILLISECONDS);
            if (gracefulStop) {
                LOG.info("Frontier gRPC service shut down gracefully");
            } else {
                LOG.warn("Frontier gRPC service shutdown timed out");
            }
        } catch (InterruptedException e) {
            LOG.warn("Interrupted while waiting for Frontier gRPC service termination", e);
            Thread.currentThread().interrupt();
        }

        healthCheckerExecutorService.shutdownNow();
        try {
            if (!healthCheckerExecutorService.awaitTermination(5, TimeUnit.SECONDS)) {
                LOG.warn("Health checker executor did not terminate");
            }
        } catch (InterruptedException e) {
            LOG.warn("Interrupted while waiting for health checker shutdown", e);
            Thread.currentThread().interrupt();
        }
    }

    /**
     * Await termination on the main thread since the grpc library uses daemon
     * threads.
     */
    public void blockUntilShutdown() {
        if (server != null) {
            try {
                frontierService.awaitTermination();
            } catch (InterruptedException ex) {
                Thread.currentThread().interrupt();
                shutdown();
                throw new RuntimeException("Interrupted while waiting for Frontier gRPC service termination", ex);
            }
        }
    }

    @Override
    public void close() {
        shutdown();
    }

    class HealthChecker implements Runnable {
        private final Frontier frontier;
        private final HealthStatusManager health;

        public HealthChecker(Frontier frontier, HealthStatusManager health) {
            this.frontier = frontier;
            this.health = health;
        }

        @Override
        public void run() {
            health.setStatus("", frontier.checkHealth());
        }
    }
}
