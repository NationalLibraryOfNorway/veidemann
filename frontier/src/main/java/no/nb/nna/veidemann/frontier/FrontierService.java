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

import java.util.Optional;
import java.util.Set;
import java.util.concurrent.CountDownLatch;
import java.util.function.Supplier;
import java.util.stream.Collectors;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.typesafe.config.ConfigException;

import io.opentracing.Tracer;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.commons.db.DbService;
import no.nb.nna.veidemann.db.RethinkDbConnection;
import no.nb.nna.veidemann.db.initializer.RethinkDbInitializer;
import no.nb.nna.veidemann.frontier.api.FrontierApiServer;
import no.nb.nna.veidemann.frontier.settings.Settings;
import no.nb.nna.veidemann.frontier.worker.DnsServiceClient;
import no.nb.nna.veidemann.frontier.worker.Frontier;
import no.nb.nna.veidemann.frontier.worker.LogServiceClient;
import no.nb.nna.veidemann.frontier.worker.OutOfScopeHandlerClient;
import no.nb.nna.veidemann.frontier.worker.RobotsServiceClient;
import no.nb.nna.veidemann.frontier.worker.ScopeServiceClient;
import redis.clients.jedis.DefaultJedisClientConfig;
import redis.clients.jedis.HostAndPort;
import redis.clients.jedis.Jedis;
import redis.clients.jedis.JedisPool;
import redis.clients.jedis.JedisPoolConfig;
import redis.clients.jedis.JedisSentinelPool;
import redis.clients.jedis.Protocol;

/**
 * Class for launching the service.
 */
public class FrontierService implements AutoCloseable {

    private static final Logger LOG = LoggerFactory.getLogger(FrontierService.class);

    private final Settings settings;
    private final Tracer tracer;
    private final CountDownLatch shutdownLatch = new CountDownLatch(1);

    // Resources that need closing
    private DbService db;
    private AutoCloseable redisResource;
    private RobotsServiceClient robotsServiceClient;
    private DnsServiceClient dnsServiceClient;
    private ScopeServiceClient scopeServiceClient;
    private OutOfScopeHandlerClient outOfScopeHandlerClient;
    private LogServiceClient logServiceClient;
    private FrontierApiServer apiServer;
    private Frontier frontier;

    /**
     * Create a new Frontier service.
     */
    public FrontierService(Settings settings, Tracer tracer) {
        this.settings = settings;
        this.tracer = tracer;
    }

    /**
     * Start the service.
     * <p>
     *
     * @return this instance
     */
    public void start() throws ConfigException, DbException {

        db = DbService.configure(settings);
        RethinkDbConnection conn = ((RethinkDbInitializer) db.getDbInitializer()).getDbConnection();

        JedisPoolConfig jedisPoolConfig = new JedisPoolConfig();
        jedisPoolConfig.setMaxTotal(256);
        jedisPoolConfig.setMaxIdle(16);
        jedisPoolConfig.setMinIdle(2);

        Supplier<Jedis> jedisSupplier;

        if (settings.getRedisSentinelMasterName() != null &&
                !settings.getRedisSentinelMasterName().isBlank()) {

            // Sentinel mode
            LOG.info("Using Redis Sentinel with master '{}' at {}:{}",
                    settings.getRedisSentinelMasterName(),
                    settings.getRedisHost(),
                    settings.getRedisPort());

            var masterCfg = clientConfigBuilder()
                    .database(0)
                    .build();

            var sentinelCfg = clientConfigBuilder().build();

            Set<HostAndPort> sentinels = Set.of(
                    new HostAndPort(settings.getRedisHost(), settings.getRedisPort()));

            JedisSentinelPool sentinelPool = new JedisSentinelPool(
                    settings.getRedisSentinelMasterName(),
                    sentinels,
                    jedisPoolConfig,
                    masterCfg,
                    sentinelCfg);

            redisResource = sentinelPool;
            jedisSupplier = sentinelPool::getResource;

        } else {

            // Standalone mode
            LOG.info("Using standalone Redis at {}:{}",
                    settings.getRedisHost(), settings.getRedisPort());

            JedisPool pool = new JedisPool(
                    jedisPoolConfig,
                    new HostAndPort(settings.getRedisHost(), settings.getRedisPort()),
                    clientConfigBuilder()
                            .database(0)
                            .build());

            redisResource = pool;
            jedisSupplier = pool::getResource;
        }

        robotsServiceClient = new RobotsServiceClient(
                settings.getRobotsEvaluatorHost(),
                settings.getRobotsEvaluatorPort());
        dnsServiceClient = new DnsServiceClient(
                settings.getDnsResolverHost(),
                settings.getDnsResolverPort());
        scopeServiceClient = new ScopeServiceClient(
                settings.getScopeserviceHost(),
                settings.getScopeservicePort());
        outOfScopeHandlerClient = new OutOfScopeHandlerClient(
                settings.getOutOfScopeHandlerHost(),
                settings.getOutOfScopeHandlerPort());
        logServiceClient = new LogServiceClient(
                settings.getLogServiceHost(),
                settings.getLogServicePort());

        frontier = new Frontier(
                tracer,
                settings,
                jedisSupplier,
                robotsServiceClient,
                dnsServiceClient,
                scopeServiceClient,
                outOfScopeHandlerClient,
                logServiceClient,
                conn,
                db.getConfigAdapter());

        apiServer = new FrontierApiServer(
                settings.getApiPort(),
                settings.getTerminationGracePeriodSeconds(),
                frontier);

        apiServer.start();

        LOG.info("Veidemann Frontier (v. {}) started",
                FrontierService.class.getPackage().getImplementationVersion());
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
        LOG.info("Shutting down FrontierService");

        if (apiServer != null) {
            try {
                apiServer.close();
            } catch (Exception e) {
                LOG.warn("Error closing API server", e);
            }
        }
        if (frontier != null) {
            try {
                frontier.close();
            } catch (Exception e) {
                LOG.warn("Error closing frontier", e);
            }
        }
        if (logServiceClient != null) {
            try {
                logServiceClient.close();
            } catch (Exception e) {
                LOG.warn("Error closing logServiceClient", e);
            }
        }
        if (outOfScopeHandlerClient != null) {
            try {
                outOfScopeHandlerClient.close();
            } catch (Exception e) {
                LOG.warn("Error closing outOfScopeHandlerClient", e);
            }
        }
        if (scopeServiceClient != null) {
            try {
                scopeServiceClient.close();
            } catch (Exception e) {
                LOG.warn("Error closing scopeServiceClient", e);
            }
        }
        if (dnsServiceClient != null) {
            try {
                dnsServiceClient.close();
            } catch (Exception e) {
                LOG.warn("Error closing dnsServiceClient", e);
            }
        }
        if (robotsServiceClient != null) {
            try {
                robotsServiceClient.close();
            } catch (Exception e) {
                LOG.warn("Error closing robotsServiceClient", e);
            }
        }
        if (redisResource != null) {
            try {
                redisResource.close();
            } catch (Exception e) {
                LOG.warn("Error closing Redis pool", e);
            }
        }
        if (db != null) {
            try {
                db.close();
            } catch (Exception e) {
                LOG.warn("Error closing db", e);
            }
        }

        LOG.info("FrontierService shutdown complete");
    }

    private Optional<String> redisPassword() {
        return Optional.ofNullable(settings.getRedisPassword())
                .map(String::trim)
                .filter(p -> !p.isEmpty());
    }

    private DefaultJedisClientConfig.Builder clientConfigBuilder() {
        var b = DefaultJedisClientConfig.builder();
        redisPassword().ifPresent(b::password);
        return b;
    }
}
