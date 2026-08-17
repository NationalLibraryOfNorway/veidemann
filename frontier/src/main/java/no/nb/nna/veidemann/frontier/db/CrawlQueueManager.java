package no.nb.nna.veidemann.frontier.db;

import java.time.Instant;
import java.time.OffsetDateTime;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.concurrent.TimeUnit;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.slf4j.MDC;

import com.google.common.hash.Hashing;
import com.google.common.primitives.Longs;
import com.google.protobuf.Timestamp;
import com.rethinkdb.RethinkDB;

import io.opentracing.Scope;
import io.opentracing.Span;
import io.opentracing.tag.Tags;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus.State;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatusChangeOrBuilder;
import no.nb.nna.veidemann.api.frontier.v1.CrawlHostGroup;
import no.nb.nna.veidemann.api.frontier.v1.JobExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.PageHarvestSpec;
import no.nb.nna.veidemann.api.frontier.v1.QueuedUri;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.commons.db.FrontierAdapter;
import no.nb.nna.veidemann.commons.db.FutureOptional;
import no.nb.nna.veidemann.db.ProtoUtils;
import no.nb.nna.veidemann.frontier.db.script.ChgAddScript;
import no.nb.nna.veidemann.frontier.db.script.ChgBusyTimeoutScript;
import no.nb.nna.veidemann.frontier.db.script.ChgCleanupIfEmptyScript;
import no.nb.nna.veidemann.frontier.db.script.ChgGetScript;
import no.nb.nna.veidemann.frontier.db.script.ChgNextScript;
import no.nb.nna.veidemann.frontier.db.script.ChgQueueCountScript;
import no.nb.nna.veidemann.frontier.db.script.ChgReleaseScript;
import no.nb.nna.veidemann.frontier.db.script.ChgUpdateBusyTimeoutScript;
import no.nb.nna.veidemann.frontier.db.script.ChgUpdateScript;
import no.nb.nna.veidemann.frontier.db.script.JobExecutionGetScript;
import no.nb.nna.veidemann.frontier.db.script.JobExecutionUpdateScript;
import no.nb.nna.veidemann.frontier.db.script.NextUriScript;
import no.nb.nna.veidemann.frontier.db.script.NextUriScript.NextUriScriptResult;
import no.nb.nna.veidemann.frontier.db.script.RedisJob.RedisContext;
import no.nb.nna.veidemann.frontier.db.script.UriAddScript;
import no.nb.nna.veidemann.frontier.db.script.UriMoveScript;
import no.nb.nna.veidemann.frontier.db.script.UriRemoveScript;
import no.nb.nna.veidemann.frontier.db.script.UriUpdateScript;
import no.nb.nna.veidemann.frontier.worker.CrawlExecutionHelpers;
import no.nb.nna.veidemann.frontier.worker.CrawlExecutionNotActiveException;
import no.nb.nna.veidemann.frontier.worker.Frontier;
import no.nb.nna.veidemann.frontier.worker.PreFetchHandler;
import no.nb.nna.veidemann.frontier.worker.QueuedUriWrapper;
import redis.clients.jedis.UnifiedJedis;
import redis.clients.jedis.params.ScanParams;
import redis.clients.jedis.resps.ScanResult;
import redis.clients.jedis.resps.Tuple;

public class CrawlQueueManager implements AutoCloseable {
    public static final String CHG_BUSY_KEY = "chg_busy{chg}";
    public static final String CHG_READY_KEY = "chg_ready{chg}";
    public static final String CHG_WAIT_KEY = "chg_wait{chg}";
    public static final String CHG_TIMEOUT_KEY = "chg_timeout{chg}";
    public static final String CHG_PREFIX = "CHG{chg}:";
    public static final String SESSION_TO_CHG_KEY = "chg_session{chg}";
    public static final String CRAWL_EXECUTION_RUNNING_KEY = "ceid_running";
    public static final String CRAWL_EXECUTION_TIMEOUT_KEY = "ceid_timeout"; // unused, but kept for compat
    public static final String CRAWL_EXECUTION_FINALIZE_KEY = "CEFINALIZE";
    public static final String JOB_EXECUTION_FINALIZE_KEY = "JEFINALIZE";
    private static final Logger LOG = LoggerFactory.getLogger(CrawlQueueManager.class);

    public static final String UEID = "UEID:";
    public static final String UCHG = "UCHG:";
    public static final String JOB_EXECUTION_PREFIX = "JEID:";
    public static final String CRAWL_EXECUTION_ID_COUNT_KEY = "EIDC";
    public static final String CRAWL_EXECUTION_JOB_EXECUTION_KEY = "EIDJ";
    public static final String JOB_EXECUTION_ID_COUNT_KEY = "JEIDC";
    public static final String QUEUE_COUNT_TOTAL_KEY = "QCT";
    public static final String REMOVE_URI_QUEUE_KEY = "REMURI";
    public static final String URI_ALREADY_INCLUDED_PREFIX = "AINC:";

    static final RethinkDB r = RethinkDB.r;
    public static final long RESCHEDULE_DELAY = 1000;

    private final FrontierAdapter frontierAdapter;
    private final UnifiedJedis redisClient;
    final UriAddScript uriAddScript;
    final UriRemoveScript uriRemoveScript;
    final UriMoveScript uriMoveScript;
    final UriUpdateScript uriUpdateScript;
    final NextUriScript nextUriScript;
    final ChgAddScript chgAddScript;
    final ChgNextScript getNextChgScript;
    final ChgReleaseScript releaseChgScript;
    final ChgQueueCountScript countChgScript;
    final ChgUpdateBusyTimeoutScript chgUpdateBusyTimeoutScript;
    final ChgUpdateScript chgUpdateScript;
    final ChgGetScript chgGetScript;
    final ChgBusyTimeoutScript chgBusyTimeoutScript;
    final ChgCleanupIfEmptyScript chgCleanupIfEmptyScript;
    final JobExecutionGetScript jobExecutionGetScript;
    final JobExecutionUpdateScript jobExecutionUpdateScript;
    final JobExecutionQueueCounter jobExecutionQueueCounter;

    private final Frontier frontier;
    private final CrawlQueueWorker crawlQueueWorker;
    private final CrawlExecutionReconciler executionReconciler;
    private final TimeoutSupplier<PreFetchHandler> nextFetchSupplier;

    // must be volatile, accessed from TimeoutSupplier worker threads
    private volatile boolean shouldRun = true;

    public CrawlQueueManager(Frontier frontier, FrontierAdapter frontierAdapter, UnifiedJedis redisClient) {
        this.frontier = frontier;
        this.frontierAdapter = frontierAdapter;
        this.redisClient = redisClient;
        uriAddScript = new UriAddScript();
        uriRemoveScript = new UriRemoveScript();
        uriMoveScript = new UriMoveScript();
        uriUpdateScript = new UriUpdateScript();
        nextUriScript = new NextUriScript();
        chgAddScript = new ChgAddScript();
        getNextChgScript = new ChgNextScript();
        releaseChgScript = new ChgReleaseScript();
        countChgScript = new ChgQueueCountScript();
        chgUpdateBusyTimeoutScript = new ChgUpdateBusyTimeoutScript();
        chgUpdateScript = new ChgUpdateScript();
        chgGetScript = new ChgGetScript();
        chgBusyTimeoutScript = new ChgBusyTimeoutScript();
        chgCleanupIfEmptyScript = new ChgCleanupIfEmptyScript();
        jobExecutionGetScript = new JobExecutionGetScript();
        jobExecutionUpdateScript = new JobExecutionUpdateScript();
        jobExecutionQueueCounter = new JobExecutionQueueCounter(redisClient);

        this.crawlQueueWorker = new CrawlQueueWorker(frontier, redisClient);

        // Prefetch queue: capacity 64, 15s timeout, 6 worker threads
        this.nextFetchSupplier = new TimeoutSupplier<>(
                64,
                15,
                TimeUnit.SECONDS,
                6,
                this::getPrefetchHandler,
                handler -> {
                    if (handler == null || handler.getQueuedUri() == null) {
                        return;
                    }
                    String chgId = handler.getQueuedUri().getCrawlHostGroupId();
                    LOG.debug("Timeout waiting for harvester, rescheduling CHG {}", chgId);
                    // conservative: release CHG, no direct URI manipulation here
                    releaseCrawlHostGroup(chgId, RESCHEDULE_DELAY);
                });
        this.executionReconciler = new CrawlExecutionReconciler(frontier, redisClient);
    }

    public QueuedUri addToCrawlHostGroup(QueuedUri qUri) throws DbException {
        MDC.put("eid", qUri.getExecutionId());
        MDC.put("uri", qUri.getUri());
        try {
            Objects.requireNonNull(qUri.getCrawlHostGroupId(), "CrawlHostGroupId cannot be null");
            Objects.requireNonNull(qUri.getPolitenessRef().getId(), "PolitenessId cannot be null");
            if (qUri.getSequence() <= 0L) {
                throw new IllegalArgumentException("Sequence must be a positive number");
            }

            if (!qUri.hasEarliestFetchTimeStamp()) {
                qUri = qUri.toBuilder()
                        .setEarliestFetchTimeStamp(ProtoUtils.getNowTs())
                        .build();
            }
            // Ensure that the URI we are about to add is not present in remove queue.
            RedisContext ctx = RedisContext.forClient(redisClient);
            ctx.getClient().lrem(REMOVE_URI_QUEUE_KEY, 0, qUri.getId());

            qUri = frontierAdapter.saveQueuedUri(qUri);

            uriAddScript.run(ctx, qUri);
            chgAddScript.run(
                    ctx,
                    qUri.getCrawlHostGroupId(),
                    qUri.getExecutionId(),
                    qUri.getJobExecutionId(),
                    qUri.getEarliestFetchTimeStamp(),
                    frontier.getSettings().getBusyTimeout().toMillis());

            // Close the window where desiredState changes while the URI is being
            // persisted. The reconciler repairs the remaining process-crash window
            // from the Redis execution counters.
            if (!CrawlExecutionHelpers.isExecutionActive(frontier, qUri.getExecutionId())) {
                throw new CrawlExecutionNotActiveException(qUri.getExecutionId());
            }

            return qUri;
        } catch (CrawlExecutionNotActiveException e) {
            throw e;
        } catch (DbException e) {
            // domain-level DB errors: log and propagate as-is
            LOG.warn("Failed to add URI {} to CrawlHostGroup {}", qUri.getUri(), qUri.getCrawlHostGroupId(), e);
            throw e;
        } catch (RuntimeException e) {
            // programming / unexpected errors: log and rethrow
            LOG.error("Unexpected error adding URI {} to CrawlHostGroup {}", qUri.getUri(), qUri.getCrawlHostGroupId(),
                    e);
            throw e;
        } finally {
            MDC.remove("eid");
            MDC.remove("uri");
        }
    }

    /**
     * Called by harvester: get next page to fetch.
     * Delegates to TimeoutSupplier which prefetches URIs.
     */
    public PageHarvestSpec getNextToFetch() {
        PreFetchHandler handler;
        try {
            handler = nextFetchSupplier.get(1, TimeUnit.SECONDS);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            return null;
        }

        if (handler == null) {
            return null;
        }

        try {
            PageHarvestSpec spec = handler.getHarvestSpec();

            // After we have a session token and CHG is updated, extend busy timeout
            long newTimeoutMs = Instant.now()
                    .plus(frontier.getSettings().getBusyTimeout())
                    .toEpochMilli();

            boolean updated = updateBusyTimeout(
                    spec.getQueuedUri().getCrawlHostGroupId(),
                    spec.getSessionToken(),
                    newTimeoutMs);

            if (!updated) {
                LOG.debug("CHG {} not busy when refreshing timeout after getHarvestSpec",
                        spec.getQueuedUri().getCrawlHostGroupId());
            }

            return spec;
        } catch (CrawlExecutionNotActiveException e) {
            LOG.debug("Discarding prefetched URI because its crawl execution is no longer active");
            try {
                releaseCrawlHostGroup(handler.getQueuedUri().getCrawlHostGroupId(), 0);
            } catch (RuntimeException releaseError) {
                LOG.debug("Prefetched crawl-host group was already released during abort cleanup");
            }
            return null;
        } catch (DbException e) {
            LOG.warn("Failed to build PageHarvestSpec for queued URI", e);
            return null;
        }
    }

    /**
     * Supplier function for TimeoutSupplier: single attempt to produce a
     * PreFetchHandler.
     * No internal infinite loop; TimeoutSupplier handles looping and back-pressure.
     */
    private PreFetchHandler getPrefetchHandler() {
        if (!shouldRun) {
            return null;
        }

        Span span = frontier.getTracer()
                .buildSpan("Prefetch")
                .withTag(Tags.COMPONENT, "Frontier")
                .withTag(Tags.SPAN_KIND, Tags.SPAN_KIND_SERVER)
                .start();

        try (Scope scope = frontier.getTracer().scopeManager().activate(span)) {
            QueueLease lease;
            try {
                lease = getNextQueuedUriToFetch(); // may throw DbException
            } catch (DbException e) {
                LOG.warn("Prefetch DB error while getting next URI to fetch", e);
                return null;
            }

            if (lease == null) {
                // No work right now; avoid tight spin in supplier threads
                try {
                    Thread.sleep(10);
                } catch (InterruptedException ie) {
                    Thread.currentThread().interrupt();
                }
                return null;
            }

            PreFetchHandler handler;
            try {
                handler = new PreFetchHandler(lease, frontier);
            } catch (DbException e) {
                LOG.warn("Failed to construct PreFetchHandler for URI {}", lease.uriId(), e);
                return null;
            } catch (RuntimeException e) {
                LOG.error("Bug constructing PreFetchHandler for URI {}", lease.uriId(), e);
                return null;
            }

            try {
                if (handler.preFetch()) {
                    // Preconditions ok, side-effects done, handler ready
                    return handler;
                } else {
                    // DENIED/RETRY/etc; PreFetchHandler already did the necessary cleanup /
                    // reschedule
                    return null;
                }
            } catch (DbException e) {
                LOG.warn("preFetch failed for URI {}", handler.getQueuedUri().getUri(), e);
                return null;
            }
        } finally {
            span.finish();
        }
    }

    private QueueLease getNextQueuedUriToFetch() throws DbException {
        RedisContext ctx = redisContext();
        CrawlHostGroup chg = getNextReadyCrawlHostGroup(ctx);
        if (chg == null) {
            return null;
        }

        String chgId = chg.getId();
        LOG.trace("Found Crawl Host Group ({})", chgId);

        FutureOptional<QueueLease> foqu = getNextFetchableQueuedUriForCrawlHostGroup(ctx, chg);

        if (foqu.isPresent()) {
            QueueLease lease = foqu.get();
            LOG.debug("Found Queued URI: {}, crawlHostGroup: {}", lease.uriId(), lease.crawlHostGroupId());
            return lease;
        } else if (foqu.isMaybeInFuture()) {
            LOG.trace("Queued URI might be available at: {}", foqu.getWhen());

            long delay = (RESCHEDULE_DELAY + foqu.getDelayMs()) / 2;
            releaseCrawlHostGroup(ctx, chgId, chg.getSessionToken(), delay, false);
        } else {
            LOG.warn("No Queued URI found for CHG {}, waiting {}ms before retry", chgId, RESCHEDULE_DELAY);
            releaseCrawlHostGroup(ctx, chgId, chg.getSessionToken(), RESCHEDULE_DELAY, false);
        }

        return null;
    }

    public void updateCrawlHostGroup(CrawlHostGroup chg) {
        chgUpdateScript.run(redisContext(), chg);
    }

    public CrawlHostGroup getCrawlHostGroup(String chgId) {
        return chgGetScript.run(redisContext(), chgId);
    }

    public CrawlHostGroup getCrawlHostGroupForSessionToken(String sessionToken) {
        if (sessionToken == null || sessionToken.isBlank()) {
            return null;
        }
        RedisContext ctx = redisContext();
        String chgId = ctx.getClient().hget(SESSION_TO_CHG_KEY, sessionToken);
        if (chgId == null) {
            return null;
        }
        return chgGetScript.run(ctx, chgId);
    }

    public long deleteQueuedUrisForExecution(String executionId) throws DbException {
        return deleteQueuedUrisForExecution(redisContext(), executionId, false).deleted();
    }

    public long deleteQueuedUrisForExecution(RedisContext ctx, String executionId) throws DbException {
        return deleteQueuedUrisForExecution(ctx, executionId, false).deleted();
    }

    public QueueCleanupResult deletePendingUrisForExecution(String executionId) throws DbException {
        return deleteQueuedUrisForExecution(redisContext(), executionId, true);
    }

    private QueueCleanupResult deleteQueuedUrisForExecution(
            RedisContext ctx,
            String executionId,
            boolean preserveActiveFetch) throws DbException {
        long deleted = 0;
        long preserved = 0;
        Set<String> affectedChgIds = new LinkedHashSet<>();
        ScanParams scanParams = new ScanParams().match(UEID + "*:" + executionId);
        ScanResult<String> queues = ctx.getClient().scan("0", scanParams);
        while (true) {
            for (String queue : queues.getResult()) {
                String chgId = crawlHostGroupIdFromExecutionQueue(queue, executionId);
                affectedChgIds.add(chgId);

                ScanResult<Tuple> uris = new ScanResult<>("0", null);
                do {
                    uris = ctx.getClient().zscan(queue, uris.getCursor());
                    for (Tuple uri : uris.getResult()) {
                        String[] uriParts = uri.getElement().split(":", 3);
                        String uriId = uriParts[2];
                        long sequence = Longs.tryParse(uriParts[0].trim());
                        long fetchTime = Longs.tryParse(uriParts[1].trim());

                        long removed = uriRemoveScript.run(
                                ctx,
                                uriId,
                                chgId,
                                executionId,
                                sequence,
                                fetchTime,
                                true,
                                preserveActiveFetch);
                        if (removed > 0) {
                            deleted += removed;
                        } else if (removed < 0) {
                            preserved++;
                        }
                    }
                } while (!uris.isCompleteIteration());
            }
            if (queues.isCompleteIteration()) {
                break;
            }
            queues = ctx.getClient().scan(queues.getCursor(), scanParams);
        }
        affectedChgIds.forEach(chgId -> chgCleanupIfEmptyScript.run(ctx, chgId));
        return new QueueCleanupResult(deleted, preserved);
    }

    static String crawlHostGroupIdFromExecutionQueue(String queue, String executionId) {
        String executionSuffix = ":" + executionId;
        if (!queue.startsWith(UEID) || !queue.endsWith(executionSuffix)) {
            throw new IllegalArgumentException(
                    "Invalid crawl execution queue key '" + queue + "' for execution " + executionId);
        }

        // UEID:<chgId>:<executionId>. Do not split on ':' because a crawl-host
        // group ID may itself contain it. The old parser also accidentally included
        // executionId in chgId, causing UriRemoveScript to append it a second time.
        return queue.substring(UEID.length(), queue.length() - executionSuffix.length());
    }

    /**
     * Atomically checks if a uri is already included in queue for a JobExecution
     * and adds the uri
     * to the datastructure such that the next call to this function with the same
     * QueuedUri will always return false.
     *
     * @param qu the uri to check
     * @return true if the uri is not seen for the JobExecution
     */
    public boolean uriNotIncludedInQueue(QueuedUriWrapper qu) {
        String jobExecutionId = qu.getJobExecutionId();
        String uriHash = uriHash(qu.getIncludedCheckUri());
        return redisClient.sadd(URI_ALREADY_INCLUDED_PREFIX + jobExecutionId, uriHash) == 1;
    }

    /**
     * Resets the stats and already included datastructures for a JobExecution.
     *
     * @param jobExecutionId job execution id
     */
    public void removeRedisJobExecution(String jobExecutionId) {
        redisClient.del(URI_ALREADY_INCLUDED_PREFIX + jobExecutionId);
        redisClient.del(JOB_EXECUTION_PREFIX + jobExecutionId);
        redisClient.hdel(JOB_EXECUTION_ID_COUNT_KEY, jobExecutionId);
        redisClient.zrem(JOB_EXECUTION_FINALIZE_KEY, jobExecutionId);
    }

    public static String uriHash(String uri) {
        return Hashing.sha256().hashUnencodedChars(uri).toString();
    }

    FutureOptional<QueueLease> getNextFetchableQueuedUriForCrawlHostGroup(
            RedisContext ctx,
            CrawlHostGroup crawlHostGroup) throws DbException {

        NextUriScriptResult res = nextUriScript.run(ctx, crawlHostGroup);
        if (res.future != null) {
            return res.future.isMaybeInFuture()
                    ? FutureOptional.emptyUntil(res.future.getWhen())
                    : FutureOptional.empty();
        }

        QueuedUri obj = frontierAdapter.getQueuedUri(res.id);

        if (obj != null) {
            QueueLease lease = new QueueLease(
                    res.id,
                    res.eid,
                    obj.getJobExecutionId(),
                    res.chgId,
                    res.sequence,
                    res.fetchTime);
            if (!res.chgId.equals(obj.getCrawlHostGroupId())) {
                LOG.info("Repairing interrupted queue move for URI {} from {} to {}",
                        res.id, res.chgId, obj.getCrawlHostGroupId());
                moveLease(ctx, lease, obj);
                releaseCrawlHostGroup(ctx, crawlHostGroup.getId(), crawlHostGroup.getSessionToken(), 0, false);
                return FutureOptional.empty();
            }
            return FutureOptional.of(lease);
        } else {
            LOG.warn("Db inconsistency: Could not find queued uri: {}, CHG: {}", res.id, res.chgId);
            removeQUri(ctx, res.id, res.chgId, res.eid, res.sequence, res.fetchTime, false);
            return FutureOptional.empty();
        }
    }

    public QueuedUri getQueuedUri(String uriId) throws DbException {
        return frontierAdapter.getQueuedUri(uriId);
    }

    public long countByCrawlExecution(String executionId) {
        String c = redisClient.hget(CRAWL_EXECUTION_ID_COUNT_KEY, executionId);
        if (c == null) {
            return 0L;
        }
        Long parsed = Longs.tryParse(c);
        if (parsed == null) {
            LOG.warn("Invalid crawl execution count '{}' for executionId {}", c, executionId);
            return 0L;
        }
        return parsed;
    }

    public Map<String, Long> countByCrawlExecutions(List<String> executionIds) {
        Map<String, Long> counts = new LinkedHashMap<>();
        if (executionIds.isEmpty()) {
            return counts;
        }

        List<String> values = redisClient.hmget(CRAWL_EXECUTION_ID_COUNT_KEY, executionIds.toArray(String[]::new));
        for (int index = 0; index < executionIds.size(); index++) {
            String executionId = executionIds.get(index);
            String value = values.get(index);
            Long parsed = value == null ? null : Longs.tryParse(value);
            if (parsed == null) {
                if (value != null) {
                    LOG.warn("Invalid crawl execution count '{}' for executionId {}", value, executionId);
                }
                parsed = 0L;
            }
            counts.put(executionId, parsed);
        }
        return counts;
    }

    public Map<String, Long> countByJobExecutions(List<String> jobExecutionIds) {
        return jobExecutionQueueCounter.counts(jobExecutionIds);
    }

    public long countByCrawlHostGroup(CrawlHostGroup chg) {
        return countChgScript.run(redisContext(), chg);
    }

    public long queueCountTotal() {
        String c = redisClient.get(QUEUE_COUNT_TOTAL_KEY);
        if (c == null) {
            return 0L;
        }
        Long parsed = Longs.tryParse(c);
        if (parsed == null) {
            LOG.warn("Invalid total queue count '{}'", c);
            return 0L;
        }
        return parsed;
    }

    public long busyCrawlHostGroupCount() {
        return redisClient.zcard(CHG_BUSY_KEY);
    }

    public void updateQueuedUri(QueuedUriWrapper queuedUriWrapper, Timestamp oldEarliestFetchTimestamp) {
        uriUpdateScript.run(redisContext(), queuedUriWrapper, oldEarliestFetchTimestamp);
    }

    private long removeQUri(
            RedisContext ctx,
            String id,
            String chgId,
            String eid,
            long sequence,
            long fetchTime,
            boolean deleteUri) {

        long numRemoved = uriRemoveScript.run(ctx, id, chgId, eid, sequence, fetchTime, deleteUri);
        if (numRemoved != 1) {
            LOG.error("Queued uri id '{}' to be removed from Redis was not found", id);
        }
        return numRemoved;
    }

    public boolean completeLease(QueueLease lease) {
        return removeQUri(lease, true);
    }

    public long moveLease(QueueLease source, QueuedUri target) {
        return moveLease(redisContext(), source, target);
    }

    private long moveLease(RedisContext ctx, QueueLease source, QueuedUri target) {
        if (source.crawlHostGroupId().equals(target.getCrawlHostGroupId())) {
            return 0L;
        }
        return uriMoveScript.run(ctx, source, target);
    }

    private boolean removeQUri(QueueLease lease, boolean deleteUri) {
        if (LOG.isTraceEnabled()) {
            LOG.trace("Completing queued URI lease {}", lease.uriId());
        }

        long numRemoved = uriRemoveScript.run(
                redisContext(),
                lease.uriId(),
                lease.crawlHostGroupId(),
                lease.executionId(),
                lease.sequence(),
                lease.fetchTime(),
                deleteUri);
        if (numRemoved != 1) {
            LOG.warn("Queued URI lease '{}' was already absent", lease.uriId());
        }
        return numRemoved == 1;
    }

    public void scheduleExecutionFinalization(String executionId) {
        redisClient.zadd(CRAWL_EXECUTION_FINALIZE_KEY, System.currentTimeMillis(), executionId);
    }

    public Long getBusyTimeout(String crawlHostGroupId) {
        Double timeout = redisClient.zscore(CHG_BUSY_KEY, crawlHostGroupId);
        if (timeout == null) {
            return null;
        }
        return timeout.longValue();
    }

    /**
     * Update timeout for busy CHG.
     * <p>
     * Timeout is only updated if CHG is already in busy state. If CHG was not busy,
     * nothing is done and the return value is false.
     *
     * @param crawlHostGroupId the CHG to update
     * @param timeoutMs        the new timeout value
     * @return true if CHG was busy
     */
    public boolean updateBusyTimeout(String crawlHostGroupId, String sessionToken, Long timeoutMs) {
        return updateBusyTimeout(redisContext(), crawlHostGroupId, sessionToken, timeoutMs);
    }

    public boolean updateBusyTimeout(RedisContext ctx, String crawlHostGroupId, String sessionToken, Long timeoutMs) {
        Long resp = chgUpdateBusyTimeoutScript.run(ctx, crawlHostGroupId, sessionToken, timeoutMs);
        return resp != null;
    }

    private CrawlHostGroup getNextReadyCrawlHostGroup(RedisContext jedisContext) {
        try {
            long busyTimeout = frontier.getSettings().getBusyTimeout().toMillis();
            return getNextChgScript.run(jedisContext, busyTimeout);
        } catch (Exception e) {
            LOG.warn("Error while getting next ready CrawlHostGroup", e);
            return null;
        }
    }

    public void releaseCrawlHostGroup(CrawlHostGroup crawlHostGroup, long nextFetchDelayMs, boolean isTimeout) {
        releaseCrawlHostGroup(redisContext(), crawlHostGroup.getId(), crawlHostGroup.getSessionToken(),
                nextFetchDelayMs, isTimeout);
    }

    public void releaseCrawlHostGroup(String crawlHostGroupId, long nextFetchDelayMs) {
        LOG.debug("Releasing CrawlHostGroup: {}, with no sessionToken", crawlHostGroupId);
        releaseCrawlHostGroup(redisContext(), crawlHostGroupId, "", nextFetchDelayMs, false);
    }

    /**
     * Release a busy CrawlHostGroup.
     * <p>
     * Moves CHG from busy queue to wait queue and removes the session token. If CHG
     * should be released because of timeout
     * while waiting for harvester, then the isTimeout parameter should be set to
     * true. In this situation the CHG is
     * already removed from busy queue and the Lua script can take that into
     * account.
     */
    public void releaseCrawlHostGroup(
            RedisContext ctx,
            String crawlHostGroupId,
            String sessionToken,
            long nextFetchDelayMs,
            boolean isTimeout) {

        releaseChgScript.run(ctx, crawlHostGroupId, sessionToken, nextFetchDelayMs, isTimeout);
    }

    public void scheduleCrawlExecutionTimeout(String ceid, OffsetDateTime timeout) {
        redisClient.zadd(
                CRAWL_EXECUTION_RUNNING_KEY,
                timeout.toInstant().toEpochMilli(),
                ceid);
    }

    public void removeCrawlExecutionFromTimeoutSchedule(String executionId) {
        redisClient.zrem(CRAWL_EXECUTION_RUNNING_KEY, executionId);
    }

    public JobExecutionStatus getTempJobExecutionStatus(String jobExecutionId) {
        return getTempJobExecutionStatus(redisContext(), jobExecutionId);
    }

    public JobExecutionStatus getTempJobExecutionStatus(RedisContext ctx, String jobExecutionId) {
        return jobExecutionGetScript.run(ctx, jobExecutionId);
    }

    /**
     * @param jobExecutionId job execution id
     * @param oldState       old state
     * @param newState       new state
     * @param change         status change info
     * @return true if job is running
     */
    public Boolean updateJobExecutionStatus(
            String jobExecutionId,
            State oldState,
            State newState,
            CrawlExecutionStatusChangeOrBuilder change) {

        return jobExecutionUpdateScript.run(redisContext(), jobExecutionId, oldState, newState, change);
    }

    private RedisContext redisContext() {
        return RedisContext.forClient(redisClient);
    }

    public void pause(boolean pause) {
        nextFetchSupplier.pause(pause);
    }

    @Override
    public void close() throws InterruptedException {
        shouldRun = false;
        executionReconciler.close();
        crawlQueueWorker.close();
        nextFetchSupplier.close();
    }
}
