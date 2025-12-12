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
package no.nb.nna.veidemann.frontier;

import java.io.IOException;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.slf4j.bridge.SLF4JBridgeHandler;

import com.typesafe.config.ConfigException;

import io.jaegertracing.Configuration;
import io.opentracing.Tracer;
import io.opentracing.util.GlobalTracer;
import io.prometheus.client.exporter.HTTPServer;
import io.prometheus.client.hotspot.DefaultExports;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.frontier.settings.Settings;

/**
 * Main class for launching the service.
 */
public final class Main {

    private static final Logger LOG = LoggerFactory.getLogger(Main.class);

    private Main() {
        // no instances
    }

    public static void main(String[] args) {
        initLoggingBridge();
        Tracer tracer = initTracing();

        try {
            Settings settings = Settings.load();

            // process-wide metrics
            DefaultExports.initialize();

            try (HTTPServer metrics = new HTTPServer(settings.getPrometheusPort());
                    FrontierService frontierService = new FrontierService(settings, tracer)) {

                registerShutdownHook(frontierService);

                frontierService.start();
                frontierService.blockUntilShutdown();
            }

        } catch (ConfigException | DbException ex) {
            LOG.error("Configuration error: {}", ex.getLocalizedMessage(), ex);
            System.exit(1);
        } catch (IOException ex) {
            LOG.error("Failed to start Prometheus server", ex);
            System.exit(1);
        } catch (Exception ex) {
            LOG.error("Fatal error in frontier", ex);
            System.exit(1);
        }
    }

    /**
     * Route all java.util.logging (JUL) logs through SLF4J/Log4j2.
     */
    private static void initLoggingBridge() {
        SLF4JBridgeHandler.removeHandlersForRootLogger();
        SLF4JBridgeHandler.install();
    }

    private static Tracer initTracing() {
        Tracer tracer = Configuration.fromEnv().getTracer();
        GlobalTracer.registerIfAbsent(tracer);
        return tracer;
    }

    private static void registerShutdownHook(FrontierService frontierService) {
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            LOG.info("JVM shutdown detected, initiating shutdown");
            frontierService.initiateShutdown();
        }, "frontier-service-shutdown-hook"));
    }
}
