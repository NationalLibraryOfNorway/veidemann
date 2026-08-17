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

package no.nb.nna.veidemann.frontier.worker;

import com.google.protobuf.Timestamp;
import com.google.protobuf.util.Durations;
import com.google.protobuf.util.Timestamps;
import no.nb.nna.veidemann.api.frontier.v1.CrawlHostGroup;
import no.nb.nna.veidemann.api.frontier.v1.QueuedUri;
import no.nb.nna.veidemann.commons.db.DbConnectionException;
import no.nb.nna.veidemann.commons.db.DbQueryException;
import no.nb.nna.veidemann.db.ProtoUtils;
import no.nb.nna.veidemann.frontier.db.QueueLease;
import no.nb.nna.veidemann.frontier.db.script.ChgAddScript;
import no.nb.nna.veidemann.frontier.db.script.ChgCleanupIfEmptyScript;
import no.nb.nna.veidemann.frontier.db.script.ChgDelayedQueueScript;
import no.nb.nna.veidemann.frontier.db.script.ChgGetScript;
import no.nb.nna.veidemann.frontier.db.script.ChgNextScript;
import no.nb.nna.veidemann.frontier.db.script.ChgQueueCountScript;
import no.nb.nna.veidemann.frontier.db.script.ChgReleaseScript;
import no.nb.nna.veidemann.frontier.db.script.ChgUpdateBusyTimeoutScript;
import no.nb.nna.veidemann.frontier.db.script.ChgUpdateScript;
import no.nb.nna.veidemann.frontier.db.script.JobExecutionGetScript;
import no.nb.nna.veidemann.frontier.db.script.JobExecutionUpdateScript;
import no.nb.nna.veidemann.frontier.db.script.NextUriScript;
import no.nb.nna.veidemann.frontier.db.script.RedisJob.RedisContext;
import no.nb.nna.veidemann.frontier.db.script.UriAddScript;
import no.nb.nna.veidemann.frontier.db.script.UriMoveScript;
import no.nb.nna.veidemann.frontier.db.script.UriRemoveScript;
import no.nb.nna.veidemann.frontier.testutil.RedisData;
import no.nb.nna.veidemann.frontier.testutil.SkipUntilFilter;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.output.BaseConsumer;
import org.testcontainers.containers.output.OutputFrame;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;
import redis.clients.jedis.ConnectionPoolConfig;
import redis.clients.jedis.DefaultJedisClientConfig;
import redis.clients.jedis.RedisClient;
import redis.clients.jedis.UnifiedJedis;

import java.time.OffsetDateTime;
import java.time.temporal.ChronoUnit;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_READY_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_BUSY_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_PREFIX;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_WAIT_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CRAWL_EXECUTION_FINALIZE_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CRAWL_EXECUTION_ID_COUNT_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.JOB_EXECUTION_ID_COUNT_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.QUEUE_COUNT_TOTAL_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.SESSION_TO_CHG_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.UEID;
import static no.nb.nna.veidemann.frontier.testutil.FrontierAssertions.assertThat;
import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.within;

@Tag("integration")
@Tag("redis")
@Testcontainers
public class CrawlHostGroupTest {
    @SuppressWarnings("resource")
    @Container
    public static GenericContainer<?> redis = new GenericContainer<>(DockerImageName.parse("redis:8-alpine"))
            .withExposedPorts(6379);

    UnifiedJedis redisClient;
    RedisData redisData;

    @BeforeEach
    public void cleanDb() throws DbQueryException, DbConnectionException {
        String redisHost = redis.getHost();
        Integer redisPort = redis.getFirstMappedPort();

        ConnectionPoolConfig poolConfig = new ConnectionPoolConfig();
        poolConfig.setMaxTotal(24);
        redisClient = RedisClient.builder()
                .hostAndPort(redisHost, redisPort)
                .clientConfig(DefaultJedisClientConfig.builder().build())
                .poolConfig(poolConfig)
                .build();
        redisData = new RedisData(redisClient);
    }

    @AfterEach
    public void shutdown() {
        redisClient.flushDB();
        redisClient.close();
    }

    @Test
    public void testChgAddScript() throws Exception {
        {
            RedisContext ctx = RedisContext.forClient(redisClient);
            String chgId1 = "myChgId";
            String chgId2 = "mySecondChgId";
            String eId1 = "myCrawlExecutionId";
            String eId2 = "mySecondCrawlExecutionId";
            String eId3 = "myThirdCrawlExecutionId";
            String jobExecutionId1 = "myJobExecutionId";
            String jobExecutionId2 = "mySecondJobExecutionId";
            Timestamp earliestFetchTimestamp1 = ProtoUtils.odtToTs(OffsetDateTime.parse("2107-12-03T10:15:30+01:00"));
            Timestamp earliestFetchTimestamp2 = ProtoUtils.odtToTs(OffsetDateTime.parse("2107-12-13T10:15:30+01:00"));

            ChgAddScript chgAddScript = new ChgAddScript();

            chgAddScript.run(ctx, chgId1, eId1, jobExecutionId1, earliestFetchTimestamp1, 1000);
            assertThat(redisData)
                    .hasQueueTotalCount(1)
                    .crawlExecutionQueueCounts().hasNumberOfElements(1).hasQueueCount(eId1, 1);
            assertThat(redisData).jobExecutionQueueCounts()
                    .hasNumberOfElements(1).hasQueueCount(jobExecutionId1, 1);
            assertThat(redisData).crawlExecutionJobExecutions()
                    .hasNumberOfElements(1).mapsTo(eId1, jobExecutionId1);
            assertThat(redisData)
                    .crawlHostGroups().hasNumberOfElements(1).id(chgId1).hasQueueCount(1);
            assertThat(redisData).waitQueue()
                    .hasNumberOfElements(1)
                    .element(0).hasTimestamp(earliestFetchTimestamp1).hasValue(chgId1);
            assertThat(redisData).busyQueue().hasNumberOfElements(0);
            assertThat(redisData).readyQueue().hasNumberOfElements(0);


            chgAddScript.run(ctx, chgId1, eId1, jobExecutionId1, earliestFetchTimestamp2, 1000);
            assertThat(redisData)
                    .hasQueueTotalCount(2)
                    .crawlExecutionQueueCounts().hasNumberOfElements(1).hasQueueCount(eId1, 2);
            assertThat(redisData).jobExecutionQueueCounts()
                    .hasNumberOfElements(1).hasQueueCount(jobExecutionId1, 2);
            assertThat(redisData)
                    .crawlHostGroups().hasNumberOfElements(1).id(chgId1).hasQueueCount(2);
            assertThat(redisData).waitQueue()
                    .hasNumberOfElements(1)
                    .element(0).hasTimestamp(earliestFetchTimestamp1).hasValue(chgId1);
            assertThat(redisData).busyQueue().hasNumberOfElements(0);
            assertThat(redisData).readyQueue().hasNumberOfElements(0);


            chgAddScript.run(ctx, chgId1, eId2, jobExecutionId2, earliestFetchTimestamp2, 1000);
            assertThat(redisData)
                    .hasQueueTotalCount(3)
                    .crawlExecutionQueueCounts().hasNumberOfElements(2).hasQueueCount(eId1, 2).hasQueueCount(eId2, 1);
            assertThat(redisData).jobExecutionQueueCounts()
                    .hasNumberOfElements(2)
                    .hasQueueCount(jobExecutionId1, 2)
                    .hasQueueCount(jobExecutionId2, 1);
            assertThat(redisData)
                    .crawlHostGroups().hasNumberOfElements(1).id(chgId1).hasQueueCount(3);
            assertThat(redisData).waitQueue()
                    .hasNumberOfElements(1)
                    .element(0).hasTimestamp(earliestFetchTimestamp1).hasValue(chgId1);
            assertThat(redisData).busyQueue().hasNumberOfElements(0);
            assertThat(redisData).readyQueue().hasNumberOfElements(0);


            chgAddScript.run(ctx, chgId2, eId2, jobExecutionId2, earliestFetchTimestamp2, 1000);
            assertThat(redisData)
                    .hasQueueTotalCount(4)
                    .crawlExecutionQueueCounts().hasNumberOfElements(2).hasQueueCount(eId1, 2).hasQueueCount(eId2, 2);
            assertThat(redisData).jobExecutionQueueCounts()
                    .hasNumberOfElements(2)
                    .hasQueueCount(jobExecutionId1, 2)
                    .hasQueueCount(jobExecutionId2, 2);
            assertThat(redisData)
                    .crawlHostGroups().hasNumberOfElements(2)
                    .id(chgId1).hasQueueCount(3)
                    .id(chgId2).hasQueueCount(1);
            assertThat(redisData).waitQueue()
                    .hasNumberOfElements(2)
                    .element(0).hasTimestamp(earliestFetchTimestamp1).hasValue(chgId1)
                    .element(1).hasTimestamp(earliestFetchTimestamp2).hasValue(chgId2);
            assertThat(redisData).busyQueue().hasNumberOfElements(0);
            assertThat(redisData).readyQueue().hasNumberOfElements(0);

            chgAddScript.run(ctx, chgId2, eId3, jobExecutionId1, earliestFetchTimestamp2, 1000);
            assertThat(redisData).jobExecutionQueueCounts()
                    .hasNumberOfElements(2)
                    .hasQueueCount(jobExecutionId1, 3)
                    .hasQueueCount(jobExecutionId2, 2);
            assertThat(redisData).crawlExecutionJobExecutions()
                    .hasNumberOfElements(3)
                    .mapsTo(eId1, jobExecutionId1)
                    .mapsTo(eId2, jobExecutionId2)
                    .mapsTo(eId3, jobExecutionId1);
        }
    }

    @Test
    public void uriRemovalUpdatesAndCleansUpJobExecutionCounters() throws Exception {
        {
            RedisContext ctx = RedisContext.forClient(redisClient);
            Timestamp fetchTime = ProtoUtils.getNowTs();
            QueuedUri queuedUri = QueuedUri.newBuilder()
                    .setId("uriId")
                    .setCrawlHostGroupId("crawlHostGroupId")
                    .setExecutionId("crawlExecutionId")
                    .setJobExecutionId("jobExecutionId")
                    .setSequence(1)
                    .setEarliestFetchTimeStamp(fetchTime)
                    .setPriorityWeight(1)
                    .setDiscoveryPath("parent")
                    .build();

            new UriAddScript().run(ctx, queuedUri);
            new ChgAddScript().run(ctx,
                    queuedUri.getCrawlHostGroupId(),
                    queuedUri.getExecutionId(),
                    queuedUri.getJobExecutionId(),
                    fetchTime,
                    1000);

            assertThat(redisData).jobExecutionQueueCounts()
                    .hasNumberOfElements(1).hasQueueCount(queuedUri.getJobExecutionId(), 1);
            assertThat(redisData).crawlExecutionJobExecutions()
                    .hasNumberOfElements(1)
                    .mapsTo(queuedUri.getExecutionId(), queuedUri.getJobExecutionId());

            UriRemoveScript uriRemoveScript = new UriRemoveScript();
            long removed = uriRemoveScript.run(ctx,
                    queuedUri.getId(),
                    queuedUri.getCrawlHostGroupId(),
                    queuedUri.getExecutionId(),
                    queuedUri.getSequence(),
                    fetchTime.getSeconds(),
                    false);

            assertThat(removed).isEqualTo(1);
            assertThat(redisData).hasQueueTotalCount(0);
            assertThat(redisData).crawlExecutionQueueCounts().hasNumberOfElements(0);
            assertThat(redisData).jobExecutionQueueCounts().hasNumberOfElements(0);
            assertThat(redisData).crawlExecutionJobExecutions().hasNumberOfElements(0);

            long removedAgain = uriRemoveScript.run(ctx,
                    queuedUri.getId(),
                    queuedUri.getCrawlHostGroupId(),
                    queuedUri.getExecutionId(),
                    queuedUri.getSequence(),
                    fetchTime.getSeconds(),
                    false);

            assertThat(removedAgain).isZero();
            assertThat(redisData).hasQueueTotalCount(0);
            assertThat(redisData).jobExecutionQueueCounts().hasNumberOfElements(0);
        }
    }

    @Test
    public void uriMoveIsAtomicIdempotentAndDoesNotChangeExecutionCounts() {
        RedisContext ctx = RedisContext.forClient(redisClient);
        Timestamp fetchTime = ProtoUtils.getNowTs();
        QueuedUri source = QueuedUri.newBuilder()
                .setId("uriId")
                .setCrawlHostGroupId("temporaryGroup")
                .setExecutionId("crawlExecutionId")
                .setJobExecutionId("jobExecutionId")
                .setSequence(1)
                .setEarliestFetchTimeStamp(fetchTime)
                .setPriorityWeight(1)
                .build();
        QueuedUri target = source.toBuilder()
                .setCrawlHostGroupId("resolvedGroup")
                .setUnresolved(false)
                .build();

        new UriAddScript().run(ctx, source);
        new ChgAddScript().run(ctx,
                source.getCrawlHostGroupId(),
                source.getExecutionId(),
                source.getJobExecutionId(),
                fetchTime,
                1000);

        UriMoveScript move = new UriMoveScript();
        assertThat(move.run(ctx, QueueLease.from(source), target)).isEqualTo(1);
        assertThat(move.run(ctx, QueueLease.from(source), target)).isZero();

        assertThat(redisClient.get(QUEUE_COUNT_TOTAL_KEY)).isEqualTo("1");
        assertThat(redisClient.hget(CRAWL_EXECUTION_ID_COUNT_KEY, source.getExecutionId())).isEqualTo("1");
        assertThat(redisClient.hget(JOB_EXECUTION_ID_COUNT_KEY, source.getJobExecutionId())).isEqualTo("1");
        assertThat(redisClient.exists(UEID + source.getCrawlHostGroupId() + ":" + source.getExecutionId()))
                .isFalse();
        assertThat(redisClient.zcard(UEID + target.getCrawlHostGroupId() + ":" + target.getExecutionId()))
                .isOne();
        assertThat(redisClient.hget(CHG_PREFIX + source.getCrawlHostGroupId(), "qc")).isEqualTo("0");
        assertThat(redisClient.hget(CHG_PREFIX + target.getCrawlHostGroupId(), "qc")).isEqualTo("1");
        assertThat(redisClient.zscore(CHG_WAIT_KEY, target.getCrawlHostGroupId())).isNotNull();
    }

    @Test
    public void finalUriRemovalSignalsFinalizationAndNewWorkCancelsStaleSignal() {
        RedisContext ctx = RedisContext.forClient(redisClient);
        Timestamp fetchTime = ProtoUtils.getNowTs();
        QueuedUri queuedUri = QueuedUri.newBuilder()
                .setId("uriId")
                .setCrawlHostGroupId("crawlHostGroupId")
                .setExecutionId("crawlExecutionId")
                .setJobExecutionId("jobExecutionId")
                .setSequence(1)
                .setEarliestFetchTimeStamp(fetchTime)
                .setPriorityWeight(1)
                .setDiscoveryPath("parent")
                .build();

        UriAddScript addUri = new UriAddScript();
        ChgAddScript addGroup = new ChgAddScript();
        addUri.run(ctx, queuedUri);
        addGroup.run(ctx, queuedUri.getCrawlHostGroupId(), queuedUri.getExecutionId(),
                queuedUri.getJobExecutionId(), fetchTime, 1000);
        assertThat(new UriRemoveScript().run(ctx,
                queuedUri.getId(), queuedUri.getCrawlHostGroupId(), queuedUri.getExecutionId(),
                queuedUri.getSequence(), fetchTime.getSeconds(), false)).isOne();
        assertThat(redisClient.zscore(CRAWL_EXECUTION_FINALIZE_KEY, queuedUri.getExecutionId())).isNotNull();

        QueuedUri replacement = queuedUri.toBuilder().setId("replacementUri").build();
        addUri.run(ctx, replacement);
        addGroup.run(ctx, replacement.getCrawlHostGroupId(), replacement.getExecutionId(),
                replacement.getJobExecutionId(), fetchTime, 1000);
        assertThat(redisClient.zscore(CRAWL_EXECUTION_FINALIZE_KEY, queuedUri.getExecutionId())).isNull();
        assertThat(redisClient.hget(CRAWL_EXECUTION_ID_COUNT_KEY, queuedUri.getExecutionId())).isEqualTo("1");
    }

    @Test
    public void abortCleanupPreservesActiveFetchAndPrunesEmptyHostGroup() {
        RedisContext ctx = RedisContext.forClient(redisClient);
        Timestamp fetchTime = ProtoUtils.getNowTs();
        QueuedUri queuedUri = QueuedUri.newBuilder()
                .setId("uriId")
                .setCrawlHostGroupId("crawlHostGroupId")
                .setExecutionId("crawlExecutionId")
                .setJobExecutionId("jobExecutionId")
                .setSequence(1)
                .setEarliestFetchTimeStamp(fetchTime)
                .setPriorityWeight(1)
                .setDiscoveryPath("parent")
                .build();

        new UriAddScript().run(ctx, queuedUri);
        new ChgAddScript().run(ctx,
                queuedUri.getCrawlHostGroupId(),
                queuedUri.getExecutionId(),
                queuedUri.getJobExecutionId(),
                fetchTime,
                1000);
        redisClient.zadd(CHG_BUSY_KEY, System.currentTimeMillis() + 1000, queuedUri.getCrawlHostGroupId());
        redisClient.hset(CHG_PREFIX + queuedUri.getCrawlHostGroupId(), "st", "session");
        redisClient.hset(SESSION_TO_CHG_KEY, "session", queuedUri.getCrawlHostGroupId());

        UriRemoveScript uriRemove = new UriRemoveScript();
        // The CHG is reserved before prefetch has stored its current URI.
        long preserved = uriRemove.run(
                ctx,
                queuedUri.getId(),
                queuedUri.getCrawlHostGroupId(),
                queuedUri.getExecutionId(),
                queuedUri.getSequence(),
                fetchTime.getSeconds(),
                true,
                true);
        assertThat(preserved).isEqualTo(-1);
        assertThat(redisData).hasQueueTotalCount(1)
                .crawlExecutionQueueCounts().hasQueueCount(queuedUri.getExecutionId(), 1);

        redisClient.hset(CHG_PREFIX + queuedUri.getCrawlHostGroupId(), "u", queuedUri.getId());
        preserved = uriRemove.run(
                ctx,
                queuedUri.getId(),
                queuedUri.getCrawlHostGroupId(),
                queuedUri.getExecutionId(),
                queuedUri.getSequence(),
                fetchTime.getSeconds(),
                true,
                true);
        assertThat(preserved).isEqualTo(-1);

        long removed = uriRemove.run(
                ctx,
                queuedUri.getId(),
                queuedUri.getCrawlHostGroupId(),
                queuedUri.getExecutionId(),
                queuedUri.getSequence(),
                fetchTime.getSeconds(),
                true,
                false);
        assertThat(removed).isEqualTo(1);
        assertThat(new ChgCleanupIfEmptyScript().run(ctx, queuedUri.getCrawlHostGroupId())).isTrue();
        assertThat(redisData).hasQueueTotalCount(0).crawlHostGroups().hasNumberOfElements(0);
        assertThat(redisData).waitQueue().hasNumberOfElements(0);
        assertThat(redisData).busyQueue().hasNumberOfElements(0);
        assertThat(redisData).sessionTokens().hasNumberOfElements(0);
    }

    @Test
    public void abortCleanupKeepsSharedHostGroupScheduled() {
        RedisContext ctx = RedisContext.forClient(redisClient);
        Timestamp fetchTime = ProtoUtils.getNowTs();
        QueuedUri first = QueuedUri.newBuilder()
                .setId("firstUri")
                .setCrawlHostGroupId("sharedHostGroup")
                .setExecutionId("abortedExecution")
                .setJobExecutionId("jobExecution")
                .setSequence(1)
                .setEarliestFetchTimeStamp(fetchTime)
                .setPriorityWeight(1)
                .setDiscoveryPath("parent")
                .build();
        QueuedUri second = first.toBuilder()
                .setId("secondUri")
                .setExecutionId("runningExecution")
                .build();

        UriAddScript uriAdd = new UriAddScript();
        ChgAddScript chgAdd = new ChgAddScript();
        uriAdd.run(ctx, first);
        chgAdd.run(ctx, first.getCrawlHostGroupId(), first.getExecutionId(),
                first.getJobExecutionId(), fetchTime, 1000);
        uriAdd.run(ctx, second);
        chgAdd.run(ctx, second.getCrawlHostGroupId(), second.getExecutionId(),
                second.getJobExecutionId(), fetchTime, 1000);

        long removed = new UriRemoveScript().run(
                ctx,
                first.getId(),
                first.getCrawlHostGroupId(),
                first.getExecutionId(),
                first.getSequence(),
                fetchTime.getSeconds(),
                true,
                true);
        assertThat(removed).isEqualTo(1);
        assertThat(new ChgCleanupIfEmptyScript().run(ctx, first.getCrawlHostGroupId())).isFalse();

        assertThat(redisData).hasQueueTotalCount(1)
                .crawlHostGroups().hasNumberOfElements(1)
                .id(first.getCrawlHostGroupId()).hasQueueCount(1);
        assertThat(redisData).crawlExecutionQueueCounts()
                .hasNumberOfElements(1)
                .hasQueueCount(second.getExecutionId(), 1);
        assertThat(redisData).waitQueue().hasNumberOfElements(1);
    }

    @Test
    public void testChgDelayedQueueScript() throws Exception {
        {
            RedisContext ctx = RedisContext.forClient(redisClient);
            String chgId1 = "myChgId";
            String chgId2 = "mySecondChgId";
            String eId1 = "myCrawlExecutionId";
            String eId2 = "mySecondCrawlExecutionId";
            Timestamp earliestFetchTimestamp1 = ProtoUtils.getNowTs();
            Timestamp earliestFetchTimestamp2 = ProtoUtils.odtToTs(OffsetDateTime.parse("2107-12-13T10:15:30+01:00"));

            ChgAddScript chgAddScript = new ChgAddScript();
            ChgDelayedQueueScript chgDelayedQueueScript = new ChgDelayedQueueScript();

            // Add some CrawlHostGroups
            chgAddScript.run(ctx, chgId1, eId1, "jobExecutionId", earliestFetchTimestamp1, 1000);
            chgAddScript.run(ctx, chgId1, eId1, "jobExecutionId", earliestFetchTimestamp2, 1000);
            chgAddScript.run(ctx, chgId1, eId2, "jobExecutionId", earliestFetchTimestamp2, 1000);
            chgAddScript.run(ctx, chgId2, eId2, "jobExecutionId", earliestFetchTimestamp2, 1000);

            // Check expected state
            assertThat(redisData)
                    .hasQueueTotalCount(4)
                    .crawlExecutionQueueCounts().hasNumberOfElements(2).hasQueueCount(eId1, 2).hasQueueCount(eId2, 2);
            assertThat(redisData)
                    .crawlHostGroups().hasNumberOfElements(2)
                    .id(chgId1).hasQueueCount(3)
                    .id(chgId2).hasQueueCount(1);
            assertThat(redisData).waitQueue()
                    .hasNumberOfElements(2)
                    .element(0).hasTimestamp(earliestFetchTimestamp1).hasValue(chgId1)
                    .element(1).hasTimestamp(earliestFetchTimestamp2).hasValue(chgId2);
            assertThat(redisData).busyQueue().hasNumberOfElements(0);
            assertThat(redisData).readyQueue().hasNumberOfElements(0);

            // Call DelayedQueue script
            Long moved = chgDelayedQueueScript.run(ctx, CHG_WAIT_KEY, CHG_READY_KEY);

            // Check expected state after move
            assertThat(moved)
                    .withFailMessage("Expected number of moved CrawlHostGroups from wait to ready to be <%d>, but was <%d>", 1, moved)
                    .isEqualTo(1);
            assertThat(redisData)
                    .hasQueueTotalCount(4)
                    .crawlExecutionQueueCounts().hasNumberOfElements(2).hasQueueCount(eId1, 2).hasQueueCount(eId2, 2);
            assertThat(redisData)
                    .crawlHostGroups().hasNumberOfElements(2)
                    .id(chgId1).hasQueueCount(3)
                    .id(chgId2).hasQueueCount(1);
            assertThat(redisData).waitQueue()
                    .hasNumberOfElements(1)
                    .element(0).hasTimestamp(earliestFetchTimestamp2).hasValue(chgId2);
            assertThat(redisData).busyQueue().hasNumberOfElements(0);
            assertThat(redisData).readyQueue().hasNumberOfElements(1)
                    .containsExactly(chgId1);
        }
    }

    class ToStdOutConsumer extends BaseConsumer<ToStdOutConsumer> {
        @Override
        public void accept(OutputFrame outputFrame) {
            System.out.println(outputFrame.getUtf8String());
        }
    }

    @Test
    public void testChgNextScript() throws Exception {
        redis.followOutput(new SkipUntilFilter("Ready to accept connections", new ToStdOutConsumer()));

        {
            RedisContext ctx = RedisContext.forClient(redisClient);
            String chgId1 = "myFirstChgId";
            String chgId2 = "mySecondChgId";
            String chgId3 = "myThirdChgId";
            String eId1 = "myFirstCrawlExecutionId";
            String eId2 = "mySecondCrawlExecutionId";
            String eId3 = "myThirdCrawlExecutionId";
            String eId4 = "myFourthCrawlExecutionId";
            Timestamp earliestFetchTimestamp1 = ProtoUtils.getNowTs();
            Timestamp earliestFetchTimestamp2 = Timestamps.add(earliestFetchTimestamp1, Durations.fromSeconds(1));

            ChgAddScript chgAddScript = new ChgAddScript();
            ChgDelayedQueueScript chgDelayedQueueScript = new ChgDelayedQueueScript();
            ChgNextScript chgNextScript = new ChgNextScript().withWaitForReadyTimeout(1);
            ChgUpdateScript chgUpdateScript = new ChgUpdateScript();
            ChgGetScript chgGetScript = new ChgGetScript();
            ChgReleaseScript chgReleaseScript = new ChgReleaseScript();

            // Add some CrawlHostGroups and move to ready
            chgAddScript.run(ctx, chgId1, eId1, "jobExecutionId", earliestFetchTimestamp1, 1000);
            chgAddScript.run(ctx, chgId1, eId1, "jobExecutionId", earliestFetchTimestamp1, 1000);
            chgAddScript.run(ctx, chgId1, eId2, "jobExecutionId", earliestFetchTimestamp2, 1000);
            chgAddScript.run(ctx, chgId2, eId3, "jobExecutionId", earliestFetchTimestamp1, 1000);
            chgAddScript.run(ctx, chgId3, eId4, "jobExecutionId", earliestFetchTimestamp2, 1000);
            Long moved = chgDelayedQueueScript.run(ctx, CHG_WAIT_KEY, CHG_READY_KEY);

            // Check expected state
            Long expectedMoved = 2L;
            assertThat(moved)
                    .withFailMessage("Expected number of moved CrawlHostGroups from wait to ready to be <%d>, but was <%d>", expectedMoved, moved)
                    .isEqualTo(expectedMoved);
            assertThat(redisData)
                    .hasQueueTotalCount(5)
                    .crawlExecutionQueueCounts().hasNumberOfElements(4)
                    .hasQueueCount(eId1, 2)
                    .hasQueueCount(eId2, 1)
                    .hasQueueCount(eId3, 1)
                    .hasQueueCount(eId4, 1);
            assertThat(redisData)
                    .crawlHostGroups().hasNumberOfElements(3)
                    .id(chgId1).hasQueueCount(3)
                    .id(chgId2).hasQueueCount(1)
                    .id(chgId3).hasQueueCount(1);
            assertThat(redisData).waitQueue()
                    .hasNumberOfElements(1)
                    .element(0).hasTimestamp(earliestFetchTimestamp2).hasValue(chgId3);
            assertThat(redisData).busyQueue().hasNumberOfElements(0);
            assertThat(redisData).readyQueue().hasNumberOfElements(2)
                    .containsExactly(chgId1, chgId2);

            CrawlHostGroup chg1 = chgNextScript.run(ctx, 2000);
            assertThat(chg1).hasQueueCount(3);

            assertThat(redisData)
                    .hasQueueTotalCount(5)
                    .crawlExecutionQueueCounts().hasNumberOfElements(4)
                    .hasQueueCount(eId1, 2)
                    .hasQueueCount(eId2, 1)
                    .hasQueueCount(eId3, 1)
                    .hasQueueCount(eId4, 1);
            assertThat(redisData)
                    .crawlHostGroups().hasNumberOfElements(3)
                    .id(chgId1).hasQueueCount(3)
                    .id(chgId2).hasQueueCount(1)
                    .id(chgId3).hasQueueCount(1);
            assertThat(redisData).waitQueue()
                    .hasNumberOfElements(1)
                    .element(0).hasTimestamp(earliestFetchTimestamp2).hasValue(chgId3);
            assertThat(redisData).busyQueue().hasNumberOfElements(1).element(0)
                    .hasTimestampCloseTo(Timestamps.add(ProtoUtils.getNowTs(), Durations.fromSeconds(2)), within(200, ChronoUnit.MILLIS))
                    .hasValue(chgId1);
            assertThat(redisData).readyQueue().hasNumberOfElements(1)
                    .containsExactly(chgId2);

            CrawlHostGroup chg2 = chgNextScript.run(ctx, 2000);
            assertThat(chg2).hasQueueCount(1);
            assertThat(redisData).busyQueue().hasNumberOfElements(2)
                    .element(0)
                    .hasTimestampCloseTo(Timestamps.add(ProtoUtils.getNowTs(), Durations.fromSeconds(2)), within(200, ChronoUnit.MILLIS))
                    .hasValue(chgId1)
                    .element(1)
                    .hasTimestampCloseTo(Timestamps.add(ProtoUtils.getNowTs(), Durations.fromSeconds(2)), within(200, ChronoUnit.MILLIS))
                    .hasValue(chgId2);
            assertThat(redisData).readyQueue().hasNumberOfElements(0);

            CrawlHostGroup chg3 = chgNextScript.run(ctx, 2000);
            assertThat(chg3).isNull();
//            System.out.println(redis.getLogs());
//            System.out.println("LOGS\n" + toStringConsumer.toUtf8String());
            Timestamp fetchStartTimestamp = ProtoUtils.getNowTs();
            chg1 = chg1.toBuilder()
                    .setCurrentUriId("uri1")
                    .setFetchStartTimeStamp(fetchStartTimestamp)
                    .setSessionToken("sess1")
                    .setRetryDelaySeconds(1)
                    .setMaxRetries(3)
                    .setMinTimeBetweenPageLoadMs(10)
                    .setMaxTimeBetweenPageLoadMs(1000)
                    .setDelayFactor(1.5f)
                    .build();
            chgUpdateScript.run(ctx, chg1);

            assertThat(redisData)
                    .hasQueueTotalCount(5)
                    .crawlExecutionQueueCounts().hasNumberOfElements(4)
                    .hasQueueCount(eId1, 2)
                    .hasQueueCount(eId2, 1)
                    .hasQueueCount(eId3, 1)
                    .hasQueueCount(eId4, 1);
            assertThat(redisData)
                    .crawlHostGroups().hasNumberOfElements(3)
                    .id(chgId1)
                    .hasQueueCount(3)
                    .hasPolitenessValues(10, 1000, 3, 1, 1.5f)
                    .hasSessionToken("sess1")
                    .hasCurrentUriId("uri1")
                    .hasFetchStartTimeStamp(fetchStartTimestamp)
                    .id(chgId2).hasQueueCount(1)
                    .id(chgId3).hasQueueCount(1);
            assertThat(redisData).waitQueue()
                    .hasNumberOfElements(1)
                    .element(0).hasTimestamp(earliestFetchTimestamp2).hasValue(chgId3);
            assertThat(redisData).busyQueue().hasNumberOfElements(2)
                    .element(0)
                    .hasValue(chgId1)
                    .hasTimestampCloseTo(Timestamps.add(ProtoUtils.getNowTs(), Durations.fromSeconds(1)), within(2000, ChronoUnit.MILLIS))
                    .element(1)
                    .hasValue(chgId2)
                    .hasTimestampCloseTo(Timestamps.add(ProtoUtils.getNowTs(), Durations.fromSeconds(1)), within(2000, ChronoUnit.MILLIS));
            assertThat(redisData).readyQueue().hasNumberOfElements(0);
            assertThat(redisData).sessionTokens().hasNumberOfElements(1).hasCrawlHostId("sess1", chgId1);

            CrawlHostGroup result = chgGetScript.run(ctx, chgId1);
            assertThat(result).isEqualTo(chg1);

            chgReleaseScript.run(ctx, chgId1, "sess1", 1999, false);
            assertThat(redisData)
                    .hasQueueTotalCount(5)
                    .crawlExecutionQueueCounts().hasNumberOfElements(4)
                    .hasQueueCount(eId1, 2)
                    .hasQueueCount(eId2, 1)
                    .hasQueueCount(eId3, 1)
                    .hasQueueCount(eId4, 1);
            assertThat(redisData)
                    .crawlHostGroups().hasNumberOfElements(3)
                    .id(chgId1)
                    .hasQueueCount(3)
                    .hasPolitenessValues(0, 0, 0, 0, 0)
                    .hasSessionToken("")
                    .hasCurrentUriId("")
                    .id(chgId2).hasQueueCount(1)
                    .id(chgId3).hasQueueCount(1);
            assertThat(redisData).waitQueue()
                    .hasNumberOfElements(2)
                    .element(0).hasValue(chgId3).hasTimestamp(earliestFetchTimestamp2)
                    .element(1).hasTimestampCloseTo(Timestamps.add(ProtoUtils.getNowTs(), Durations.fromSeconds(2)), within(200, ChronoUnit.MILLIS));
            assertThat(redisData).busyQueue().hasNumberOfElements(1)
//                    .element(0)
//                    .hasValue(chgId1)
//                    .hasTimestampCloseTo(Timestamps.add(ProtoUtils.getNowTs(), Durations.fromSeconds(1)), within(200, ChronoUnit.MILLIS))
                    .element(0)
                    .hasValue(chgId2)
                    .hasTimestampCloseTo(Timestamps.add(ProtoUtils.getNowTs(), Durations.fromSeconds(1)), within(2000, ChronoUnit.MILLIS));
            assertThat(redisData).readyQueue().hasNumberOfElements(0);
            assertThat(redisData).sessionTokens().hasNumberOfElements(0);
        }
    }

    public void forOtherTests() {
                Object[] scripts = {
                                new UriAddScript(),
                                new UriRemoveScript(),
                                new NextUriScript(),
                                new ChgAddScript(),
                                new ChgNextScript(),
                                new ChgReleaseScript(),
                                new ChgQueueCountScript(),
                                new ChgUpdateBusyTimeoutScript(),
                                new ChgUpdateScript(),
                                new ChgGetScript(),
                                new JobExecutionGetScript(),
                                new JobExecutionUpdateScript()
                };
                assertThat(scripts).hasSize(12);
    }
}
