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

import java.io.File;
import java.io.IOException;
import java.io.UncheckedIOException;
import java.time.Duration;
import java.util.ArrayList;
import java.util.List;
import java.util.Objects;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import io.grpc.BindableService;
import io.grpc.Server;
import io.grpc.ServerBuilder;
import io.grpc.ServerInterceptor;
import io.grpc.ServerInterceptors;
import io.grpc.ServerServiceDefinition;
import io.opentracing.contrib.grpc.TracingServerInterceptor;
import io.opentracing.contrib.grpc.TracingServerInterceptor.ServerRequestAttribute;
import io.opentracing.util.GlobalTracer;
import no.nb.nna.veidemann.commons.auth.ApiKeyAuAuServerInterceptor;
import no.nb.nna.veidemann.commons.auth.ApiKeyRoleMapper;
import no.nb.nna.veidemann.commons.auth.ApiKeyRoleMapperFromConfig;
import no.nb.nna.veidemann.commons.auth.ApiKeyRoleMapperFromFile;
import no.nb.nna.veidemann.commons.auth.AuthorisationAuAuServerInterceptor;
import no.nb.nna.veidemann.commons.auth.IdTokenAuAuServerInterceptor;
import no.nb.nna.veidemann.commons.auth.IdTokenValidator;
import no.nb.nna.veidemann.commons.auth.NoopAuAuServerInterceptor;
import no.nb.nna.veidemann.commons.auth.UserRoleMapper;
import no.nb.nna.veidemann.controller.settings.Settings;

public class ControllerApiServer implements AutoCloseable {

    private static final Logger LOG = LoggerFactory.getLogger(ControllerApiServer.class);

    private static final String TLS_DIR = "/veidemann/tls";
    private static final String TLS_CERT = "tls.crt";
    private static final String TLS_KEY = "tls.key";

    private static final Duration SERVER_SHUTDOWN_TIMEOUT = Duration.ofSeconds(5);
    private static final Duration EXECUTOR_SHUTDOWN_TIMEOUT = Duration.ofSeconds(5);
    private static final int IDP_MAX_RETRIES = 20;
    private static final Duration IDP_RETRY_DELAY = Duration.ofSeconds(20);

    final Settings settings;
    final UserRoleMapper userRoleMapper;
    final ScopeServiceClient scopeServiceClient;
    final LogServiceClient logServiceClient;

    private final ServerBuilder<?> serverBuilder;
    private final ExecutorService executor;
    private final List<JobExecutionListener> jobExecutionListeners = new CopyOnWriteArrayList<>();

    private Server server;
    private boolean closed;

    public interface JobExecutionListener {
        void onJobStarting(String jobExecutionId);

        void onJobStarted(String jobExecutionId);
    }

    public ControllerApiServer(Settings settings,
            UserRoleMapper userRoleMapper,
            ScopeServiceClient scopeServiceClient,
            LogServiceClient logServiceClient) {
        this(
                settings,
                ServerBuilder.forPort(settings.getApiPort()),
                userRoleMapper,
                scopeServiceClient,
                logServiceClient);
    }

    public ControllerApiServer(Settings settings,
            ServerBuilder<?> serverBuilder,
            UserRoleMapper userRoleMapper,
            ScopeServiceClient scopeServiceClient,
            LogServiceClient logServiceClient) {
        this.settings = Objects.requireNonNull(settings, "settings");
        this.serverBuilder = Objects.requireNonNull(serverBuilder, "serverBuilder");
        this.userRoleMapper = userRoleMapper; // may be null in tests
        this.scopeServiceClient = scopeServiceClient; // may be null in tests
        this.logServiceClient = logServiceClient; // may be null in tests

        this.executor = Executors.newVirtualThreadPerTaskExecutor();
        this.serverBuilder.executor(executor);

        Runtime.getRuntime().addShutdownHook(
                Thread.ofPlatform()
                        .name("controller-api-shutdown-hook")
                        .unstarted(this::safeClose));
    }

    public void addJobExecutionListener(JobExecutionListener listener) {
        jobExecutionListeners.add(Objects.requireNonNull(listener, "listener"));
    }

    public void removeJobExecutionListener(JobExecutionListener listener) {
        jobExecutionListeners.remove(listener);
    }

    public ControllerApiServer start() {
        List<ServerInterceptor> interceptors = buildInterceptors();
        configureTlsIfAvailable();

        server = serverBuilder
                .addService(createService(new ConfigService(scopeServiceClient), interceptors))
                .addService(createService(new ControllerService(settings, jobExecutionListeners), interceptors))
                .addService(createService(new ReportService(settings), interceptors))
                .addService(createService(new EventService(), interceptors))
                .addService(createService(new LogService(logServiceClient), interceptors))
                .build();

        try {
            server.start();
            LOG.info("Controller API listening on {}", server.getPort());
            return this;
        } catch (IOException e) {
            close();
            throw new UncheckedIOException(e);
        }
    }

    private List<ServerInterceptor> buildInterceptors() {
        var tracing = TracingServerInterceptor
                .newBuilder()
                .withTracer(GlobalTracer.get())
                .withTracedAttributes(
                        ServerRequestAttribute.CALL_ATTRIBUTES,
                        ServerRequestAttribute.METHOD_TYPE)
                .build();

        var interceptors = new ArrayList<ServerInterceptor>();
        interceptors.add(tracing);

        try {
            return getAuAuServerInterceptors(interceptors);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new IllegalStateException("Interrupted while building auth interceptors", e);
        }
    }

    /**
     * Authentication/authorization interceptors.
     */
    List<ServerInterceptor> getAuAuServerInterceptors(List<ServerInterceptor> interceptors)
            throws InterruptedException {

        if (settings.getSkipAuthentication()) {
            LOG.warn("Authentication is skipped (settings.skipAuthentication == true). For tests/dev only.");
            interceptors.add(new NoopAuAuServerInterceptor());
            return interceptors;
        }

        String issuerUrl = settings.getOpenIdConnectIssuer();
        if (issuerUrl != null && !issuerUrl.isEmpty()) {
            IdTokenValidator idTokenValidator = null;

            int retryAttempts = 0;
            while (idTokenValidator == null && retryAttempts < IDP_MAX_RETRIES) {
                try {
                    idTokenValidator = new IdTokenValidator(issuerUrl);
                } catch (Exception e) {
                    retryAttempts++;
                    LOG.warn("Failed to initialize IdTokenValidator (attempt {}/{}). Retrying in {}...",
                            retryAttempts, IDP_MAX_RETRIES, IDP_RETRY_DELAY, e);
                    Thread.sleep(IDP_RETRY_DELAY.toMillis());
                }
            }

            if (idTokenValidator != null) {
                interceptors.add(new IdTokenAuAuServerInterceptor(userRoleMapper, idTokenValidator));
            } else {
                LOG.error("Unable to initialize IdTokenValidator after {} attempts. " +
                        "Continuing without ID token auth.", IDP_MAX_RETRIES);
            }
        }

        ApiKeyRoleMapper apiKeyDbRoleMapper = new ApiKeyRoleMapperFromConfig(userRoleMapper);
        interceptors.add(new ApiKeyAuAuServerInterceptor(apiKeyDbRoleMapper));

        ApiKeyRoleMapper apiKeyFileRoleMapper = new ApiKeyRoleMapperFromFile(settings.getApiKeyRoleMappingFile());
        interceptors.add(new ApiKeyAuAuServerInterceptor(apiKeyFileRoleMapper));

        return interceptors;
    }

    private void configureTlsIfAvailable() {
        File dir = new File(TLS_DIR);
        File cert = new File(dir, TLS_CERT);
        File key = new File(dir, TLS_KEY);

        if (cert.isFile() && cert.canRead() && key.isFile() && key.canRead()) {
            LOG.info("Found TLS certificate and key in {}. Enabling TLS.", dir.getAbsolutePath());
            serverBuilder.useTransportSecurity(cert, key);
        } else {
            LOG.warn("No readable TLS cert/key found in {}. Using insecure plaintext.", dir.getAbsolutePath());
        }
    }

    @SuppressWarnings("resource") // interceptor only used to build wrapped service; lifecycle is owned by gRPC
    private ServerServiceDefinition createService(BindableService service,
            List<ServerInterceptor> interceptors) {
        var authInterceptor = new AuthorisationAuAuServerInterceptor(service);
        var interceptedService = authInterceptor.intercept(service);
        return ServerInterceptors.interceptForward(interceptedService, interceptors);
    }

    @Override
    public synchronized void close() {
        if (closed) {
            return;
        }
        closed = true;

        shutdownServer();
        shutdownExecutor();

        LOG.info("Controller API server shut down");
    }

    private void safeClose() {
        try {
            close();
        } catch (Exception e) {
            LOG.warn("Error during shutdown hook close()", e);
        }
    }

    private void shutdownServer() {
        if (server == null) {
            return;
        }

        server.shutdown();
        try {
            boolean terminated = server.awaitTermination(SERVER_SHUTDOWN_TIMEOUT.toMillis(), TimeUnit.MILLISECONDS);
            if (!terminated) {
                LOG.warn("Server did not terminate within {}. Forcing shutdownNow().",
                        SERVER_SHUTDOWN_TIMEOUT);
                server.shutdownNow();
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            LOG.warn("Interrupted while awaiting server termination, forcing shutdownNow().", e);
            server.shutdownNow();
        }
    }

    private void shutdownExecutor() {
        executor.shutdown();
        try {
            boolean terminated = executor.awaitTermination(EXECUTOR_SHUTDOWN_TIMEOUT.toMillis(), TimeUnit.MILLISECONDS);
            if (!terminated) {
                LOG.warn("Executor did not terminate within {}. Forcing shutdownNow().",
                        EXECUTOR_SHUTDOWN_TIMEOUT);
                executor.shutdownNow();
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            LOG.warn("Interrupted while awaiting executor termination, forcing shutdownNow().", e);
            executor.shutdownNow();
        }
    }
}
