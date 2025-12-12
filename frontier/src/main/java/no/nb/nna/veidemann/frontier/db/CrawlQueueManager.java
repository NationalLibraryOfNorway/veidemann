package no.nb.nna.veidemann.frontier.db;

import static no.nb.nna.veidemann.db.ProtoUtils.rethinkToProto;

import java.time.Instant;
import java.time.OffsetDateTime;
import java.util.Arrays;
import java.util.List;
import java.util.Map;
import java.util.Objects;
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
import no.nb.nna.veidemann.commons.db.DbConnectionException;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.commons.db.DbQueryException;
import no.nb.nna.veidemann.commons.db.FutureOptional;
import no.nb.nna.veidemann.db.ProtoUtils;
import no.nb.nna.veidemann.db.RethinkDbConnection;
import no.nb.nna.veidemann.db.Tables;
import no.nb.nna.veidemann.frontier.db.script.ChgAddScript;
import no.nb.nna.veidemann.frontier.db.script.ChgBusyTimeoutScript;
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
import no.nb.nna.veidemann.frontier.db.script.RedisJob.JedisContext;
import no.nb.nna.veidemann.frontier.db.script.UriAddScript;
import no.nb.nna.veidemann.frontier.db.script.UriRemoveScript;
import no.nb.nna.veidemann.frontier.db.script.UriUpdateScript;
import no.nb.nna.veidemann.frontier.worker.Frontier;
import no.nb.nna.veidemann.frontier.worker.PreFetchHandler;
import no.nb.nna.veidemann.frontier.worker.QueuedUriWrapper;
import redis.clients.jedis.Jedis;
import redis.clients.jedis.JedisPool;
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
    private static final Logger LOG = LoggerFactory.getLogger(CrawlQueueManager.class);

    public static final String UEID = "UEID:";
    public static final String UCHG = "UCHG:";
    public static final String JOB_EXECUTION_PREFIX = "JEID:";
    public static final String CRAWL_EXECUTION_ID_COUNT_KEY = "EIDC";
    public static final String QUEUE_COUNT_TOTAL_KEY = "QCT";
    public static final String REMOVE_URI_QUEUE_KEY = "REMURI";
    public static final String URI_ALREADY_INCLUDED_PREFIX = "AINC:";

    static final RethinkDB r = RethinkDB.r;
    public static final long RESCHEDULE_DELAY = 1000;

    private final RethinkDbConnection conn;
    private final JedisPool jedisPool;
    final UriAddScript uriAddScript;
    final UriRemoveScript uriRemoveScript;
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
    final JobExecutionGetScript jobExecutionGetScript;
    final JobExecutionUpdateScript jobExecutionUpdateScript;

    private final Frontier frontier;
    private final CrawlQueueWorker crawlQueueWorker;
    private final TimeoutSupplier<PreFetchHandler> nextFetchSupplier;

    // must be volatile, accessed from TimeoutSupplier worker threads
    private volatile boolean shouldRun = true;

    public CrawlQueueManager(Frontier frontier, RethinkDbConnection conn, JedisPool jedisPool) {
        this.frontier = frontier;
        this.conn = conn;
        this.jedisPool = jedisPool;
        uriAddScript = new UriAddScript();
        uriRemoveScript = new UriRemoveScript();
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
        jobExecutionGetScript = new JobExecutionGetScript();
        jobExecutionUpdateScript = new JobExecutionUpdateScript();

        this.crawlQueueWorker = new CrawlQueueWorker(frontier, jedisPool);

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

            Map<String, Object> rMap = ProtoUtils.protoToRethink(qUri);

            // Ensure that the URI we are about to add is not present in remove queue.
            try (JedisContext ctx = JedisContext.forPool(jedisPool)) {
                ctx.getJedis().lrem(REMOVE_URI_QUEUE_KEY, 0, qUri.getId());
            }

            Map<String, Object> response = conn.exec(
                    "db-saveQueuedUri",
                    r.table(Tables.URI_QUEUE.name)
                            .insert(rMap)
                            .optArg("durability", "soft")
                            .optArg("conflict", "replace")
                            .optArg("return_changes", "always"));

            @SuppressWarnings("unchecked")
            List<Map<String, Map>> changes = (List<Map<String, Map>>) response.get("changes");
            Map<String, Object> newDoc = changes.get(0).get("new_val");
            qUri = rethinkToProto(newDoc, QueuedUri.class);

            try (JedisContext ctx = JedisContext.forPool(jedisPool)) {
                uriAddScript.run(ctx, qUri);
                chgAddScript.run(
                        ctx,
                        qUri.getCrawlHostGroupId(),
                        qUri.getExecutionId(),
                        qUri.getEarliestFetchTimeStamp(),
                        frontier.getSettings().getBusyTimeout().toMillis());
            }

            return qUri;
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
            QueuedUri u;
            try {
                u = getNextQueuedUriToFetch(); // may throw DbException
            } catch (DbException e) {
                LOG.warn("Prefetch DB error while getting next URI to fetch", e);
                return null;
            }

            if (u == null) {
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
                handler = new PreFetchHandler(u, frontier);
            } catch (DbException e) {
                LOG.warn("Failed to construct PreFetchHandler for URI {}", u.getUri(), e);
                return null;
            } catch (RuntimeException e) {
                LOG.error("Bug constructing PreFetchHandler for URI {}", u.getUri(), e);
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

    private QueuedUri getNextQueuedUriToFetch() throws DbException {
        try (JedisContext jedisContext = JedisContext.forPool(jedisPool)) {
            CrawlHostGroup chg = getNextReadyCrawlHostGroup(jedisContext);
            if (chg == null) {
                return null;
            }

            String chgId = chg.getId();
            LOG.trace("Found Crawl Host Group ({})", chgId);

            FutureOptional<QueuedUri> foqu = getNextFetchableQueuedUriForCrawlHostGroup(jedisContext, chg, conn);

            if (foqu.isPresent()) {
                QueuedUri u = foqu.get();
                LOG.debug("Found Queued URI: {}, crawlHostGroup: {}", u.getUri(), u.getCrawlHostGroupId());
                return u;
            } else if (foqu.isMaybeInFuture()) {
                LOG.trace("Queued URI might be available at: {}", foqu.getWhen());

                long delay = (RESCHEDULE_DELAY + foqu.getDelayMs()) / 2;
                releaseCrawlHostGroup(jedisContext, chgId, chg.getSessionToken(), delay, false);
            } else {
                LOG.warn("No Queued URI found for CHG {}, waiting {}ms before retry", chgId, RESCHEDULE_DELAY);
                releaseCrawlHostGroup(jedisContext, chgId, chg.getSessionToken(), RESCHEDULE_DELAY, false);
            }

            return null;
        }
    }

    public void updateCrawlHostGroup(CrawlHostGroup chg) {
        try (JedisContext ctx = JedisContext.forPool(jedisPool)) {
            chgUpdateScript.run(ctx, chg);
        }
    }

    public CrawlHostGroup getCrawlHostGroup(String chgId) {
        try (JedisContext ctx = JedisContext.forPool(jedisPool)) {
            return chgGetScript.run(ctx, chgId);
        }
    }

    public CrawlHostGroup getCrawlHostGroupForSessionToken(String sessionToken) {
        if (sessionToken == null || sessionToken.isBlank()) {
            return null;
        }
        try (JedisContext ctx = JedisContext.forPool(jedisPool)) {
            String chgId = ctx.getJedis().hget(SESSION_TO_CHG_KEY, sessionToken);
            if (chgId == null) {
                return null;
            }
            return chgGetScript.run(ctx, chgId);
        }
    }

    public long deleteQueuedUrisForExecution(String executionId) throws DbException {
        try (JedisContext ctx = JedisContext.forPool(jedisPool)) {
            return deleteQueuedUrisForExecution(ctx, executionId);
        }
    }

    public long deleteQueuedUrisForExecution(JedisContext ctx, String executionId) throws DbException {
        long deleted = 0;
        ScanParams scanParams = new ScanParams().match(UEID + "*:" + executionId);
        ScanResult<String> queues = ctx.getJedis().scan("0", scanParams);
        while (true) {
            for (String queue : queues.getResult()) {
                String[] queueParts = queue.split(":");
                String chgp = queueParts[1] + ":" + queueParts[2];

                ScanResult<Tuple> uris = new ScanResult<>("0", null);
                do {
                    uris = ctx.getJedis().zscan(queue, uris.getCursor());
                    for (Tuple uri : uris.getResult()) {
                        String[] uriParts = uri.getElement().split(":", 3);
                        String uriId = uriParts[2];
                        long sequence = Longs.tryParse(uriParts[0].trim());
                        long fetchTime = Longs.tryParse(uriParts[1].trim());

                        deleted += removeQUri(ctx, uriId, chgp, executionId, sequence, fetchTime, true);
                    }
                } while (!uris.isCompleteIteration());
            }
            if (queues.isCompleteIteration()) {
                break;
            }
            queues = ctx.getJedis().scan(queues.getCursor(), scanParams);
        }
        return deleted;
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
        try (Jedis jedis = jedisPool.getResource()) {
            return jedis.sadd(URI_ALREADY_INCLUDED_PREFIX + jobExecutionId, uriHash) == 1;
        }
    }

    /**
     * Resets the stats and already included datastructures for a JobExecution.
     *
     * @param jobExecutionId job execution id
     */
    public void removeRedisJobExecution(String jobExecutionId) {
        try (Jedis jedis = jedisPool.getResource()) {
            jedis.del(URI_ALREADY_INCLUDED_PREFIX + jobExecutionId);
            jedis.del(JOB_EXECUTION_PREFIX + jobExecutionId);
        }
    }

    public static String uriHash(String uri) {
        return Hashing.sha256().hashUnencodedChars(uri).toString();
    }

    FutureOptional<QueuedUri> getNextFetchableQueuedUriForCrawlHostGroup(
            JedisContext ctx,
            CrawlHostGroup crawlHostGroup,
            RethinkDbConnection conn) throws DbException {

        NextUriScriptResult res = nextUriScript.run(ctx, crawlHostGroup);
        if (res.future != null) {
            return res.future;
        }

        Map<String, Object> obj = conn.exec(
                "db-getNextQueuedUriToFetch",
                r.table(Tables.URI_QUEUE.name).get(res.id));

        if (obj != null) {
            return FutureOptional.of(rethinkToProto(obj, QueuedUri.class));
        } else {
            LOG.warn("Db inconsistency: Could not find queued uri: {}, CHG: {}", res.id, res.chgId);
            removeQUri(ctx, res.id, res.chgId, res.eid, res.sequence, res.fetchTime, false);
            return FutureOptional.empty();
        }
    }

    public QueuedUri getQueuedUri(String uriId) throws DbException {
        return conn.executeGet(
                "db-getQueuedUri",
                r.table(Tables.URI_QUEUE.name).get(uriId),
                QueuedUri.class);
    }

    public long countByCrawlExecution(String executionId) {
        try (Jedis jedis = jedisPool.getResource()) {
            String c = jedis.hget(CRAWL_EXECUTION_ID_COUNT_KEY, executionId);
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
    }

    public long countByCrawlHostGroup(CrawlHostGroup chg) {
        try (JedisContext ctx = JedisContext.forPool(jedisPool)) {
            return countChgScript.run(ctx, chg);
        }
    }

    public long queueCountTotal() {
        try (Jedis jedis = jedisPool.getResource()) {
            String c = jedis.get(QUEUE_COUNT_TOTAL_KEY);
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
    }

    public long busyCrawlHostGroupCount() {
        try (Jedis jedis = jedisPool.getResource()) {
            return jedis.zcard(CHG_BUSY_KEY);
        }
    }

    public void updateQueuedUri(QueuedUriWrapper queuedUriWrapper, Timestamp oldEarliestFetchTimestamp) {
        try (JedisContext ctx = JedisContext.forPool(jedisPool)) {
            uriUpdateScript.run(ctx, queuedUriWrapper, oldEarliestFetchTimestamp);
        }
    }

    private long removeQUri(
            JedisContext ctx,
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

    public boolean removeTmpCrawlHostGroup(QueuedUri qUri, String tmpChgId, boolean deleteUri) {
        return removeQUri(qUri, tmpChgId, deleteUri);
    }

    public boolean removeQUri(QueuedUriWrapper qUri) {
        QueuedUri toBeRemoved = qUri.getQueuedUriForRemoval();
        if (toBeRemoved.getId().isEmpty()) {
            return false;
        }
        return removeQUri(qUri.getQueuedUriForRemoval(), qUri.getCrawlHostGroupId(), true);
    }

    private boolean removeQUri(QueuedUri qUri, String chgId, boolean deleteUri) {
        if (LOG.isTraceEnabled()) {
            String stack = Arrays.stream(new RuntimeException().getStackTrace())
                    .filter(s -> s.getClassName().contains("no.nb.nna"))
                    .map(s -> String.format(
                            "%s:%s(%d)",
                            s.getClassName().substring(s.getClassName().lastIndexOf(".") + 1),
                            s.getMethodName(),
                            s.getLineNumber()))
                    .reduce("", (r, s) -> r.isEmpty() ? s : r + "<<" + s);
            LOG.trace("remUri: {}, Trace: {}", qUri.getId(), stack);
        }

        try (JedisContext ctx = JedisContext.forPool(jedisPool)) {
            long numRemoved = uriRemoveScript.run(
                    ctx,
                    qUri.getId(),
                    chgId,
                    qUri.getExecutionId(),
                    qUri.getSequence(),
                    qUri.getEarliestFetchTimeStamp().getSeconds(),
                    deleteUri);
            if (numRemoved != 1) {
                LOG.error("Queued uri id '{}' to be removed from Redis was not found", qUri.getId());
            }
        }
        return true;
    }

    public Long getBusyTimeout(String crawlHostGroupId) {
        try (JedisContext ctx = JedisContext.forPool(jedisPool)) {
            Double timeout = ctx.getJedis().zscore(CHG_BUSY_KEY, crawlHostGroupId);
            if (timeout == null) {
                return null;
            }
            return timeout.longValue();
        }
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
        try (JedisContext ctx = JedisContext.forPool(jedisPool)) {
            return updateBusyTimeout(ctx, crawlHostGroupId, sessionToken, timeoutMs);
        }
    }

    public boolean updateBusyTimeout(JedisContext ctx, String crawlHostGroupId, String sessionToken, Long timeoutMs) {
        Long resp = chgUpdateBusyTimeoutScript.run(ctx, crawlHostGroupId, sessionToken, timeoutMs);
        return resp != null;
    }

    private CrawlHostGroup getNextReadyCrawlHostGroup(JedisContext jedisContext) {
        try {
            long busyTimeout = frontier.getSettings().getBusyTimeout().toMillis();
            return getNextChgScript.run(jedisContext, busyTimeout);
        } catch (Exception e) {
            LOG.warn("Error while getting next ready CrawlHostGroup", e);
            return null;
        }
    }

    public void releaseCrawlHostGroup(CrawlHostGroup crawlHostGroup, long nextFetchDelayMs, boolean isTimeout) {
        try (JedisContext ctx = JedisContext.forPool(jedisPool)) {
            releaseCrawlHostGroup(ctx, crawlHostGroup.getId(), crawlHostGroup.getSessionToken(), nextFetchDelayMs,
                    isTimeout);
        }
    }

    public void releaseCrawlHostGroup(String crawlHostGroupId, long nextFetchDelayMs) {
        try (JedisContext ctx = JedisContext.forPool(jedisPool)) {
            LOG.debug("Releasing CrawlHostGroup: {}, with no sessionToken", crawlHostGroupId);
            releaseCrawlHostGroup(ctx, crawlHostGroupId, "", nextFetchDelayMs, false);
        }
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
            JedisContext ctx,
            String crawlHostGroupId,
            String sessionToken,
            long nextFetchDelayMs,
            boolean isTimeout) {

        releaseChgScript.run(ctx, crawlHostGroupId, sessionToken, nextFetchDelayMs, isTimeout);
    }

    public void scheduleCrawlExecutionTimeout(String ceid, OffsetDateTime timeout) {
        try (Jedis jedis = jedisPool.getResource()) {
            jedis.zadd(
                    CRAWL_EXECUTION_RUNNING_KEY,
                    timeout.toInstant().toEpochMilli(),
                    ceid);
        }
    }

    public void removeCrawlExecutionFromTimeoutSchedule(String executionId) {
        try (Jedis jedis = jedisPool.getResource()) {
            jedis.zrem(CRAWL_EXECUTION_RUNNING_KEY, executionId);
        }
    }

    public JobExecutionStatus getTempJobExecutionStatus(String jobExecutionId) {
        try (JedisContext ctx = JedisContext.forPool(jedisPool)) {
            return getTempJobExecutionStatus(ctx, jobExecutionId);
        }
    }

    public JobExecutionStatus getTempJobExecutionStatus(JedisContext ctx, String jobExecutionId) {
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

        try (JedisContext ctx = JedisContext.forPool(jedisPool)) {
            return jobExecutionUpdateScript.run(ctx, jobExecutionId, oldState, newState, change);
        }
    }

    public void pause(boolean pause) {
        nextFetchSupplier.pause(pause);
    }

    @Override
    public void close() throws InterruptedException {
        shouldRun = false;
        crawlQueueWorker.close();
        nextFetchSupplier.close();
    }
}
