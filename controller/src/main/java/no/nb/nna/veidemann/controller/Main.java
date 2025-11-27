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
package no.nb.nna.veidemann.controller;

import com.typesafe.config.ConfigException;
import io.jaegertracing.Configuration;
import io.opentracing.Tracer;
import io.opentracing.util.GlobalTracer;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.controller.settings.Settings;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.slf4j.bridge.SLF4JBridgeHandler;

/**
 * Main class for launching the controller service.
 */
public final class Main {

    private static final Logger LOG = LoggerFactory.getLogger(Main.class);

    private Main() {
        // no instances
    }

    public static void main(String[] args) {
        initLoggingBridge();
        initTracing();

        try (Controller controller = new Controller(Settings.load())) {
            registerShutdownHook(controller);

            controller.start();
            controller.blockUntilShutdown();

        } catch (ConfigException | DbException ex) {
            LOG.error("Configuration error: {}", ex.getLocalizedMessage(), ex);
            System.exit(1);
        } catch (Exception ex) {
            LOG.error("Fatal error in controller", ex);
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

    private static void initTracing() {
        Tracer tracer = Configuration.fromEnv().getTracer();
        GlobalTracer.registerIfAbsent(tracer);
    }

    private static void registerShutdownHook(Controller controller) {
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            System.err.println("*** JVM shutdown detected, initiating controller shutdown");
            controller.initiateShutdown();
            System.err.println("*** controller shutdown signal sent");
        }, "controller-shutdown-hook"));
    }
}
