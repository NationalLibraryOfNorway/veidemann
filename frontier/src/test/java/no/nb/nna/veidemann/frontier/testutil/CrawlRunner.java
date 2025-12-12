/*
 * Copyright 2019 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package no.nb.nna.veidemann.frontier.testutil;

import static com.rethinkdb.RethinkDB.r;
import static org.assertj.core.api.Assertions.assertThat;
import static org.awaitility.Awaitility.await;

import java.time.Duration;
import java.util.Arrays;
import java.util.Collection;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.concurrent.CompletionService;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorCompletionService;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.stream.Collectors;

import org.assertj.core.description.Description;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.google.common.util.concurrent.SettableFuture;
import com.rethinkdb.net.Cursor;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import no.nb.nna.veidemann.api.config.v1.ConfigObject;
import no.nb.nna.veidemann.api.config.v1.ConfigRef;
import no.nb.nna.veidemann.api.config.v1.CrawlLimitsConfig;
import no.nb.nna.veidemann.api.config.v1.Kind;
import no.nb.nna.veidemann.api.config.v1.PolitenessConfig.RobotsPolicy;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionId;
import no.nb.nna.veidemann.api.frontier.v1.CrawlSeedRequest;
import no.nb.nna.veidemann.api.frontier.v1.CrawlSeedRequest.Builder;
import no.nb.nna.veidemann.api.frontier.v1.FrontierGrpc;
import no.nb.nna.veidemann.api.frontier.v1.JobExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.JobExecutionStatus.State;
import no.nb.nna.veidemann.commons.db.ConfigAdapter;
import no.nb.nna.veidemann.commons.db.DbConnectionException;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.commons.db.DbQueryException;
import no.nb.nna.veidemann.commons.db.DbService;
import no.nb.nna.veidemann.commons.db.ExecutionsAdapter;
import no.nb.nna.veidemann.commons.util.ApiTools;
import no.nb.nna.veidemann.db.RethinkDbConnection;
import no.nb.nna.veidemann.db.Tables;
import no.nb.nna.veidemann.db.initializer.RethinkDbInitializer;
import no.nb.nna.veidemann.frontier.settings.Settings;
import redis.clients.jedis.Jedis;
import redis.clients.jedis.JedisPool;

public class CrawlRunner implements AutoCloseable {
    private static final Logger LOG = LoggerFactory.getLogger(CrawlRunner.class);

    private final ConfigAdapter configAdapter;
    private final ExecutionsAdapter executionsAdapter;
    private final RethinkDbConnection conn;
    private final ManagedChannel frontierChannel;
    private final FrontierGrpc.FrontierBlockingStub frontierStub;
    private final RethinkDbData rethinkDbData;
    private final JedisPool jedisPool;
    private final ExecutorService submitSeedExecutor = Executors.newFixedThreadPool(8);

    public CrawlRunner(Settings settings, RethinkDbData rethinkDbData, JedisPool jedisPool) {
        DbService dbService = DbService.getInstance();
        this.configAdapter = dbService.getConfigAdapter();
        this.executionsAdapter = dbService.getExecutionsAdapter();
        this.conn = ((RethinkDbInitializer) dbService.getDbInitializer()).getDbConnection();

        this.frontierChannel = ManagedChannelBuilder
                .forAddress("localhost", settings.getApiPort())
                .usePlaintext()
                .build();
        this.frontierStub = FrontierGrpc.newBlockingStub(frontierChannel).withWaitForReady();
        this.rethinkDbData = rethinkDbData;
        this.jedisPool = jedisPool;
    }

    public ConfigObject genJob(String name) throws DbException {
        return genJob(name, CrawlLimitsConfig.getDefaultInstance(), 1.0);
    }

    public ConfigObject genJob(String name, CrawlLimitsConfig limits, double priority) throws DbException {
        ConfigObject.Builder defaultCrawlHostGroupConfig = configAdapter.getConfigObject(
                ConfigRef.newBuilder()
                        .setKind(Kind.crawlHostGroupConfig)
                        .setId("chg-default")
                        .build())
                .toBuilder();

        defaultCrawlHostGroupConfig.getCrawlHostGroupConfigBuilder()
                .setMinTimeBetweenPageLoadMs(1)
                .setMaxTimeBetweenPageLoadMs(1)
                .setDelayFactor(.1f)
                .setMaxRetries(3)
                .setRetryDelaySeconds(1);

        configAdapter.saveConfigObject(defaultCrawlHostGroupConfig.build());

        ConfigObject.Builder politenessBuilder = ConfigObject.newBuilder()
                .setApiVersion("v1")
                .setKind(Kind.politenessConfig);
        politenessBuilder.getMetaBuilder().setName("stress");
        politenessBuilder.getPolitenessConfigBuilder()
                .setRobotsPolicy(RobotsPolicy.OBEY_ROBOTS);
        ConfigObject politeness = configAdapter.saveConfigObject(politenessBuilder.build());

        ConfigObject.Builder browserConfigBuilder = ConfigObject.newBuilder()
                .setApiVersion("v1")
                .setKind(Kind.browserConfig);
        browserConfigBuilder.getMetaBuilder().setName("stress");
        ConfigObject browserConfig = configAdapter.saveConfigObject(browserConfigBuilder.build());

        ConfigObject.Builder collectionBuilder = ConfigObject.newBuilder()
                .setApiVersion("v1")
                .setKind(Kind.collection);
        collectionBuilder.getMetaBuilder().setName("stress");
        ConfigObject collection = configAdapter.saveConfigObject(collectionBuilder.build());

        ConfigObject.Builder crawlConfigBuilder = ConfigObject.newBuilder()
                .setApiVersion("v1")
                .setKind(Kind.crawlConfig);
        crawlConfigBuilder.getMetaBuilder().setName("stress");
        crawlConfigBuilder.getCrawlConfigBuilder()
                .setPriorityWeight(priority)
                .setPolitenessRef(ApiTools.refForConfig(politeness))
                .setBrowserConfigRef(ApiTools.refForConfig(browserConfig))
                .setCollectionRef(ApiTools.refForConfig(collection));
        ConfigObject crawlConfig = configAdapter.saveConfigObject(crawlConfigBuilder.build());

        ConfigObject.Builder scopeScriptBuilder = ConfigObject.newBuilder()
                .setApiVersion("v1")
                .setKind(Kind.browserScript);
        scopeScriptBuilder.getMetaBuilder().setName("stress");
        ConfigObject scopeScript = configAdapter.saveConfigObject(scopeScriptBuilder.build());

        ConfigObject.Builder crawlJobBuilder = ConfigObject.newBuilder()
                .setApiVersion("v1")
                .setKind(Kind.crawlJob);
        crawlJobBuilder.getMetaBuilder().setName(name);
        crawlJobBuilder.getCrawlJobBuilder()
                .setCrawlConfigRef(ApiTools.refForConfig(crawlConfig))
                .setScopeScriptRef(ApiTools.refForConfig(scopeScript))
                .setLimits(limits);

        return configAdapter.saveConfigObject(crawlJobBuilder.build());
    }

    public List<SeedAndExecutions> genSeeds(int count, String hostPrefix, ConfigObject... jobs) throws DbException {
        return genSeeds(0, count, hostPrefix, jobs);
    }

    public List<SeedAndExecutions> genSeeds(int offset, int count, String hostPrefix, ConfigObject... jobs)
            throws DbException {
        LOG.info("Generating {} seeds with prefix '{}'", count, hostPrefix);

        Set<ConfigRef> jobRefs = Arrays.stream(jobs)
                .map(ApiTools::refForConfig)
                .collect(Collectors.toSet());

        CompletionService<Void> generateSeedService = new ExecutorCompletionService<>(submitSeedExecutor);
        SeedAndExecutions[] seeds = new SeedAndExecutions[count];

        for (int i = 0; i < count; i++) {
            final int idx = i;
            final String name = String.format("%s-%06d", hostPrefix, i + offset);
            final String url = String.format("http://%s-%06d.com", hostPrefix, i + offset);

            generateSeedService.submit(() -> {
                try {
                    ConfigObject.Builder entityBuilder = ConfigObject.newBuilder()
                            .setApiVersion("v1")
                            .setKind(Kind.crawlEntity);
                    entityBuilder.getMetaBuilder().setName(name);
                    ConfigObject entity = configAdapter.saveConfigObject(entityBuilder.build());

                    ConfigObject.Builder seedBuilder = ConfigObject.newBuilder()
                            .setApiVersion("v1")
                            .setKind(Kind.seed);
                    seedBuilder.getMetaBuilder().setName(url);
                    seedBuilder.getSeedBuilder()
                            .setEntityRef(ApiTools.refForConfig(entity))
                            .addAllJobRef(jobRefs);

                    ConfigObject seed = configAdapter.saveConfigObject(seedBuilder.build());
                    seeds[idx] = new SeedAndExecutions(seed, jobRefs);
                } catch (Exception e) {
                    LOG.error("Failed generating seed {}", name, e);
                }
                return null;
            });
        }

        for (int i = 0; i < count; i++) {
            try {
                generateSeedService.take();
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                LOG.warn("Seed generation interrupted", e);
                break;
            }
        }
        return Arrays.asList(seeds);
    }

    public RunningCrawl runCrawl(final ConfigObject crawlJob, final List<SeedAndExecutions> seeds) throws DbException {
        LOG.info("Submitting {} seeds to job '{}'", seeds.size(), crawlJob.getMeta().getName());
        JobExecutionStatus jes = executionsAdapter.createJobExecutionStatus(crawlJob.getId());

        RunningCrawl runningCrawl = new RunningCrawl();
        runningCrawl.remainingSeeds = new CountDownLatch(seeds.size());

        try {
            for (SeedAndExecutions seed : seeds) {
                submitSeedExecutor.submit(() -> {
                    Objects.requireNonNull(seed, "seed");
                    Objects.requireNonNull(crawlJob, "crawlJob");
                    Objects.requireNonNull(jes, "jobExecutionStatus");
                    Objects.requireNonNull(jes.getId(), "jobExecutionId");
                    Objects.requireNonNull(seed.seed, "seed.configObject");

                    Builder requestBuilder = CrawlSeedRequest.newBuilder()
                            .setJob(crawlJob)
                            .setSeed(seed.seed)
                            .setJobExecutionId(jes.getId());

                    CrawlExecutionId ceid = frontierStub.crawlSeed(requestBuilder.build());
                    seed.crawlExecutions.get(crawlJob.getId()).set(ceid);
                    runningCrawl.remainingSeeds.countDown();
                });
            }
        } catch (Exception e) {
            LOG.error("Error while submitting seeds for crawl job {}", crawlJob.getMeta().getName(), e);
        }

        runningCrawl.jobName = crawlJob.getMeta().getName();
        runningCrawl.jes = jes;
        return runningCrawl;
    }

    public void awaitCrawlFinished(RunningCrawl... runningCrawls) {
        awaitCrawlFinished(30, TimeUnit.SECONDS, runningCrawls);
    }

    public Duration awaitCrawlFinished(long timeout, TimeUnit unit, RunningCrawl... runningCrawls) {
        AtomicInteger emptyChgKeysCount = new AtomicInteger(0);

        await()
                .pollDelay(1, TimeUnit.SECONDS)
                .pollInterval(1, TimeUnit.SECONDS)
                .atMost(timeout, unit)
                .until(() -> {
                    try (Jedis jedis = jedisPool.getResource()) {
                        Set<String> chgKeys = jedis.keys("chg*");
                        if (chgKeys.isEmpty()) {
                            emptyChgKeysCount.incrementAndGet();
                        }

                        List<RunningCrawl> statuses = Arrays.stream(runningCrawls)
                                .map(j -> {
                                    try {
                                        j.jes = executionsAdapter.getJobExecutionStatus(j.jes.getId());
                                        return j;
                                    } catch (DbException ex) {
                                        throw new RuntimeException(ex);
                                    }
                                })
                                .filter(j -> j.jes.getState() == State.RUNNING)
                                .peek(j -> {
                                    if (LOG.isTraceEnabled()) {
                                        LOG.trace(
                                                "Job '{}' {}, Executions: CREATED={}, FETCHING={}, SLEEPING={}, "
                                                        + "FINISHED={}, ABORTED_TIMEOUT={}, ABORTED_SIZE={}, "
                                                        + "ABORTED_MANUAL={}, FAILED={}",
                                                j.jobName, j.jes.getState(),
                                                j.jes.getExecutionsStateMap().getOrDefault("CREATED", 0),
                                                j.jes.getExecutionsStateMap().getOrDefault("FETCHING", 0),
                                                j.jes.getExecutionsStateMap().getOrDefault("SLEEPING", 0),
                                                j.jes.getExecutionsStateMap().getOrDefault("FINISHED", 0),
                                                j.jes.getExecutionsStateMap().getOrDefault("ABORTED_TIMEOUT", 0),
                                                j.jes.getExecutionsStateMap().getOrDefault("ABORTED_SIZE", 0),
                                                j.jes.getExecutionsStateMap().getOrDefault("ABORTED_MANUAL", 0),
                                                j.jes.getExecutionsStateMap().getOrDefault("FAILED", 0));
                                    }
                                })
                                .collect(Collectors.toList());

                        boolean allDone = statuses.stream().allMatch(j -> j.jes.getState() != State.RUNNING);

                        if (allDone
                                && rethinkDbData.getQueuedUris().isEmpty()
                                && jedis.keys("*").size() <= 1) {
                            return true;
                        }

                        if (statuses.stream().anyMatch(j -> j.jes.getState() == State.RUNNING)) {
                            Description desc = new Description() {
                                @Override
                                public String value() {
                                    StringBuilder sb = new StringBuilder();
                                    try (Jedis jedisInner = jedisPool.getResource()) {
                                        sb.append(String.format(
                                                "Crawl is not finished, but redis chg keys are missing.%n"
                                                        + "Remaining REDIS keys: %s%nQueue count total: %s",
                                                jedisInner.keys("*"),
                                                jedisInner.get("QCT")));
                                        Cursor c = conn.exec("db-getQueuedUris", r.table(Tables.URI_QUEUE.name));
                                        c.forEach(v -> sb.append("\nURI in RethinkDB queue: ").append(v));
                                    } catch (DbConnectionException | DbQueryException ex) {
                                        LOG.error("Error querying queue state", ex);
                                    }
                                    return sb.toString();
                                }
                            };
                            assertThat(emptyChgKeysCount)
                                    .as(desc)
                                    .withFailMessage("")
                                    .hasValueLessThan(3);
                        }

                        LOG.debug("Still running: {}", statuses.size());
                        return false;
                    }
                });

        // Kept signature; could be used by callers later.
        return null;
    }

    @Override
    public void close() throws Exception {
        submitSeedExecutor.shutdownNow();
        frontierChannel.shutdownNow().awaitTermination(5, TimeUnit.SECONDS);
    }

    public static class SeedAndExecutions {
        final ConfigObject seed;
        final Map<String, SettableFuture<CrawlExecutionId>> crawlExecutions = new HashMap<>();

        public SeedAndExecutions(ConfigObject seed, Collection<ConfigRef> jobRefs) {
            this.seed = seed;
            for (ConfigRef ref : jobRefs) {
                crawlExecutions.put(ref.getId(), SettableFuture.create());
            }
        }

        public ConfigObject getSeed() {
            return seed;
        }

        public SettableFuture<CrawlExecutionId> getCrawlExecution(ConfigObject job) {
            return crawlExecutions.get(job.getId());
        }
    }

    public static class RunningCrawl {
        String jobName;
        JobExecutionStatus jes;
        CountDownLatch remainingSeeds;

        public JobExecutionStatus getStatus() {
            return jes;
        }

        public long awaitAllSeedsSubmitted(long timeout, TimeUnit unit) throws InterruptedException {
            remainingSeeds.await(timeout, unit);
            return remainingSeeds.getCount();
        }
    }
}
