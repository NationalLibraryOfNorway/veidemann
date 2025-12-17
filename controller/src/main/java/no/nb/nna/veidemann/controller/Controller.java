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

import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Objects;
import java.util.concurrent.CountDownLatch;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.typesafe.config.ConfigException;

import no.nb.nna.veidemann.commons.auth.UserRoleMapper;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.commons.db.DbService;
import no.nb.nna.veidemann.controller.scheduler.CrawlJobScheduler;
import no.nb.nna.veidemann.controller.settings.Settings;

public final class Controller implements AutoCloseable {

    private static final Logger LOG = LoggerFactory.getLogger(Controller.class);

    private final Settings settings;
    private final UserRoleMapper userRoleMapper;
    private final CountDownLatch shutdownLatch = new CountDownLatch(1);

    // Resources that need closing
    private DbService db;
    private FrontierClient urlFrontierClient;
    private ScopeServiceClient scopeServiceClient;
    private LogServiceClient logServiceClient;
    private ControllerApiServer apiServer;
    private CrawlJobScheduler scheduler;

    public Controller(Settings settings) {
        this(settings, new UserRoleMapper());
    }

    public Controller(Settings settings, UserRoleMapper userRoleMapper) {
        this.settings = Objects.requireNonNull(settings, "settings");
        this.userRoleMapper = Objects.requireNonNull(userRoleMapper, "userRoleMapper");
    }

    public void start() throws ConfigException, DbException {
        importTrustedCaIfPresent();

        db = DbService.configure(settings);
        urlFrontierClient = new FrontierClient(
                settings.getFrontierHost(),
                settings.getFrontierPort(),
                "url");
        scopeServiceClient = new ScopeServiceClient(
                settings.getScopeserviceHost(),
                settings.getScopeservicePort());
        logServiceClient = new LogServiceClient(
                settings.getLogServiceHost(),
                settings.getLogServicePort());
        apiServer = new ControllerApiServer(
                settings,
                userRoleMapper,
                scopeServiceClient,
                logServiceClient);
        scheduler = new CrawlJobScheduler();

        userRoleMapper.start();
        scheduler.start();
        apiServer.start();

        LOG.info("Veidemann Controller (v. {}) started",
                Controller.class.getPackage().getImplementationVersion());
    }

    public void blockUntilShutdown() {
        try {
            shutdownLatch.await();
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
    }

    /** Called from shutdown hook or tests. */
    public void initiateShutdown() {
        shutdownLatch.countDown();
    }

    @Override
    public void close() {
        if (userRoleMapper != null) {
            try {
                userRoleMapper.close();
            } catch (Exception e) {
                LOG.warn("Error closing userRoleMapper", e);
            }
        }
        // close in reverse startup order, guarding nulls
        if (scheduler != null) {
            try {
                scheduler.close();
            } catch (Exception e) {
                LOG.warn("Error closing scheduler", e);
            }
        }
        if (apiServer != null) {
            try {
                apiServer.close();
            } catch (Exception e) {
                LOG.warn("Error closing apiServer", e);
            }
        }
        if (logServiceClient != null) {
            try {
                logServiceClient.close();
            } catch (Exception e) {
                LOG.warn("Error closing logServiceClient", e);
            }
        }
        if (scopeServiceClient != null) {
            try {
                scopeServiceClient.close();
            } catch (Exception e) {
                LOG.warn("Error closing scopeServiceClient", e);
            }
        }
        if (urlFrontierClient != null) {
            try {
                urlFrontierClient.close();
            } catch (Exception e) {
                LOG.warn("Error closing urlFrontierClient", e);
            }
        }
        if (db != null) {
            try {
                db.close();
            } catch (Exception e) {
                LOG.warn("Error closing db", e);
            }
        }
    }

    private void importTrustedCaIfPresent() {
        Path certPath = Path.of(settings.getTrustedCaCertPath());

        if (!Files.exists(certPath)) {
            return;
        }
        try {
            CaImporter.importFromFile(certPath.toString());
        } catch (Exception e) {
            LOG.error("Failed loading certificates from {}", certPath, e);
        }
    }
}
