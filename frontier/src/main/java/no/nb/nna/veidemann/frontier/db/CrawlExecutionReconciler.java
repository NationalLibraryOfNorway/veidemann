/*
 * Copyright 2026 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 */
package no.nb.nna.veidemann.frontier.db;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CRAWL_EXECUTION_FINALIZE_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CRAWL_EXECUTION_ID_COUNT_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.JOB_EXECUTION_FINALIZE_KEY;

import java.util.List;
import java.util.Map.Entry;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.google.common.util.concurrent.ThreadFactoryBuilder;

import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus.State;
import no.nb.nna.veidemann.commons.db.ChangeFeed;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.frontier.worker.CrawlExecutionHelpers;
import no.nb.nna.veidemann.frontier.worker.Frontier;
import no.nb.nna.veidemann.frontier.worker.StatusWrapper;
import redis.clients.jedis.UnifiedJedis;
import redis.clients.jedis.params.ScanParams;
import redis.clients.jedis.params.ZRangeParams;
import redis.clients.jedis.resps.ScanResult;

/** Reconciles abort requests and durable count-to-zero finalization signals. */
final class CrawlExecutionReconciler implements AutoCloseable {
    private static final Logger LOG = LoggerFactory.getLogger(CrawlExecutionReconciler.class);
    private static final int BATCH_SIZE = 100;

    private final Frontier frontier;
    private final UnifiedJedis redisClient;
    private final ExecutorService changeFeedExecutor;
    private final ScheduledExecutorService maintenanceExecutor;
    private volatile boolean closed;
    private volatile ChangeFeed<CrawlExecutionStatus> currentFeed;
    private String queueCountCursor = "0";

    CrawlExecutionReconciler(Frontier frontier, UnifiedJedis redisClient) {
        this.frontier = frontier;
        this.redisClient = redisClient;
        ThreadFactoryBuilder threads = new ThreadFactoryBuilder()
                .setNameFormat("CrawlExecutionReconciler-%d")
                .setUncaughtExceptionHandler((thread, error) ->
                        LOG.error("Uncaught exception in {}", thread.getName(), error));
        changeFeedExecutor = Executors.newSingleThreadExecutor(threads.build());
        maintenanceExecutor = Executors.newSingleThreadScheduledExecutor(threads.build());
        changeFeedExecutor.execute(this::watchAbortRequests);
        maintenanceExecutor.scheduleWithFixedDelay(this::processFinalizationSignals, 1, 1, TimeUnit.SECONDS);
        maintenanceExecutor.scheduleWithFixedDelay(this::runAntiEntropy, 1, 60, TimeUnit.SECONDS);
    }

    private void watchAbortRequests() {
        long retrySeconds = 1;
        while (!closed) {
            try (ChangeFeed<CrawlExecutionStatus> feed = frontier.getExecutionsAdapter()
                    .watchCrawlExecutionAbortRequests()) {
                currentFeed = feed;
                var changes = feed.stream().iterator();
                while (!closed && changes.hasNext()) {
                    retrySeconds = 1;
                    reconcileSafely(changes.next());
                }
            } catch (Exception e) {
                if (!closed) {
                    LOG.warn("Abort changefeed disconnected; reconnecting in {} seconds", retrySeconds, e);
                }
            } finally {
                currentFeed = null;
            }
            if (!closed) {
                try {
                    TimeUnit.SECONDS.sleep(retrySeconds);
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                    return;
                }
                retrySeconds = Math.min(60, retrySeconds * 2);
            }
        }
    }

    private void processFinalizationSignals() {
        try {
            List<String> executionIds = redisClient.zrange(
                    CRAWL_EXECUTION_FINALIZE_KEY,
                    ZRangeParams.zrangeByScoreParams(0, System.currentTimeMillis()).limit(0, BATCH_SIZE));
            executionIds.forEach(this::reconcileByIdSafely);

            List<String> jobExecutionIds = redisClient.zrange(
                    JOB_EXECUTION_FINALIZE_KEY,
                    ZRangeParams.zrangeByScoreParams(0, System.currentTimeMillis()).limit(0, BATCH_SIZE));
            jobExecutionIds.forEach(this::repairJobSafely);
        } catch (Exception e) {
            LOG.warn("Could not process execution finalization signals", e);
        }
    }

    private void repairJobSafely(String jobExecutionId) {
        try {
            if (StatusWrapper.repairJobExecution(frontier, jobExecutionId)) {
                redisClient.zrem(JOB_EXECUTION_FINALIZE_KEY, jobExecutionId);
            } else {
                redisClient.zadd(JOB_EXECUTION_FINALIZE_KEY,
                        System.currentTimeMillis() + TimeUnit.MINUTES.toMillis(1),
                        jobExecutionId);
            }
        } catch (Exception e) {
            LOG.warn("Could not repair job execution {}", jobExecutionId, e);
        }
    }

    private void runAntiEntropy() {
        try {
            ScanResult<Entry<String, String>> result = redisClient.hscan(
                    CRAWL_EXECUTION_ID_COUNT_KEY,
                    queueCountCursor,
                    new ScanParams().count(BATCH_SIZE));
            queueCountCursor = result.getCursor();
            for (Entry<String, String> entry : result.getResult()) {
                CrawlExecutionStatus status = frontier.getExecutionsAdapter()
                        .getCrawlExecutionStatus(entry.getKey());
                if (status != null
                        && (CrawlExecutionHelpers.isAbortState(status.getDesiredState())
                                || CrawlExecutionHelpers.isTerminalState(status.getState()))) {
                    reconcileSafely(status);
                }
            }
        } catch (Exception e) {
            LOG.warn("Could not run execution anti-entropy repair", e);
        }
    }

    private void reconcileByIdSafely(String executionId) {
        try {
            CrawlExecutionStatus status = frontier.getExecutionsAdapter()
                    .getCrawlExecutionStatus(executionId);
            if (status == null) {
                redisClient.zrem(CRAWL_EXECUTION_FINALIZE_KEY, executionId);
                return;
            }
            reconcile(status);
        } catch (Exception e) {
            LOG.warn("Could not reconcile crawl execution {}", executionId, e);
        }
    }

    private void reconcileSafely(CrawlExecutionStatus status) {
        try {
            reconcile(status);
        } catch (Exception e) {
            LOG.warn("Could not reconcile crawl execution {}", status.getId(), e);
        }
    }

    private void reconcile(CrawlExecutionStatus status) throws DbException {
        String executionId = status.getId();
        frontier.getCrawlQueueManager().removeCrawlExecutionFromTimeoutSchedule(executionId);

        if (CrawlExecutionHelpers.isTerminalState(status.getState())) {
            frontier.getCrawlQueueManager().deleteQueuedUrisForExecution(executionId);
            if (status.getDesiredState() == status.getState() || !status.hasEndTime()) {
                StatusWrapper.getStatusWrapper(frontier, status)
                        .setEndState(status.getState())
                        .saveStatus();
            }
            if (StatusWrapper.repairJobExecution(frontier, status.getJobExecutionId())) {
                redisClient.zrem(JOB_EXECUTION_FINALIZE_KEY, status.getJobExecutionId());
            } else {
                // Keep a durable, low-frequency repair trigger until the remaining
                // children become terminal. This covers a crash between the crawl
                // status write and its Redis job-aggregate update.
                redisClient.zadd(JOB_EXECUTION_FINALIZE_KEY,
                        System.currentTimeMillis(),
                        status.getJobExecutionId());
            }
            redisClient.zrem(CRAWL_EXECUTION_FINALIZE_KEY, executionId);
            return;
        }

        State desiredState = status.getDesiredState();
        if (CrawlExecutionHelpers.isAbortState(desiredState)) {
            QueueCleanupResult cleanup = frontier.getCrawlQueueManager()
                    .deletePendingUrisForExecution(executionId);
            if (cleanup.preservedActive() > 0) {
                redisClient.zadd(CRAWL_EXECUTION_FINALIZE_KEY,
                        System.currentTimeMillis() + 1000L,
                        executionId);
                return;
            }
            StatusWrapper.getStatusWrapper(frontier, status)
                    .setEndState(desiredState)
                    .saveStatus();
            redisClient.zrem(CRAWL_EXECUTION_FINALIZE_KEY, executionId);
            return;
        }

        if (frontier.getCrawlQueueManager().countByCrawlExecution(executionId) > 0) {
            redisClient.zrem(CRAWL_EXECUTION_FINALIZE_KEY, executionId);
            return;
        }

        StatusWrapper.getStatusWrapper(frontier, status)
                .setEndState(State.FINISHED)
                .saveStatus();
        redisClient.zrem(CRAWL_EXECUTION_FINALIZE_KEY, executionId);
    }

    @Override
    public void close() throws InterruptedException {
        closed = true;
        ChangeFeed<CrawlExecutionStatus> feed = currentFeed;
        if (feed != null) {
            feed.close();
        }
        changeFeedExecutor.shutdownNow();
        maintenanceExecutor.shutdown();
        changeFeedExecutor.awaitTermination(5, TimeUnit.SECONDS);
        maintenanceExecutor.awaitTermination(5, TimeUnit.SECONDS);
    }
}
