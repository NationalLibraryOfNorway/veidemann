package no.nb.nna.veidemann.frontier.db;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_TIMEOUT_KEY;

import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.function.Supplier;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.google.common.util.concurrent.ThreadFactoryBuilder;

import no.nb.nna.veidemann.api.commons.v1.Error;
import no.nb.nna.veidemann.api.frontier.v1.CrawlHostGroup;
import no.nb.nna.veidemann.commons.ExtraStatusCodes;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.commons.db.DbService;
import no.nb.nna.veidemann.frontier.db.script.RedisJob.JedisContext;
import no.nb.nna.veidemann.frontier.worker.Frontier;
import no.nb.nna.veidemann.frontier.worker.PostFetchHandler;
import redis.clients.jedis.Jedis;

public class CrawlQueueWorker implements AutoCloseable {
    private static final Logger LOG = LoggerFactory.getLogger(CrawlQueueWorker.class);

    private final Frontier frontier;
    private final Supplier<Jedis> jedisSupplier;
    private final ScheduledExecutorService executor;

    public CrawlQueueWorker(Frontier frontier, Supplier<Jedis> jedisSupplier) {
        this.frontier = frontier;
        this.jedisSupplier = jedisSupplier;
        this.executor = Executors.newScheduledThreadPool(
                2,
                new ThreadFactoryBuilder()
                        .setNameFormat("CrawlQueueWorker-%d")
                        .setUncaughtExceptionHandler((t, e) ->
                                LOG.error("Uncaught exception in {}", t.getName(), e))
                        .build()
        );

        executor.scheduleWithFixedDelay(this::runFetchTimeoutWorker, 1200, 500, TimeUnit.MILLISECONDS);
        executor.scheduleAtFixedRate(this::runCheckPaused, 3, 3, TimeUnit.SECONDS);
    }

    private void runFetchTimeoutWorker() {
        Error err = ExtraStatusCodes.RUNTIME_EXCEPTION.toFetchError("Timeout waiting for Harvester");

        try (JedisContext ctx = JedisContext.forSupplier(jedisSupplier)) {
            var jedis = ctx.getJedis();
            String chgId;
            while ((chgId = jedis.lpop(CHG_TIMEOUT_KEY)) != null) {
                processTimedOutChg(ctx, chgId, err);
            }
        } catch (Exception e) {
            LOG.warn("Error in fetchTimeoutWorker", e);
        }
    }

    private void processTimedOutChg(JedisContext ctx, String chgId, Error err) {
        PostFetchHandler postFetchHandler = null;
        try {
            CrawlHostGroup chg = frontier.getCrawlQueueManager().getCrawlHostGroup(chgId);
            if (chg.getCurrentUriId().isEmpty()) {
                frontier.getCrawlQueueManager()
                        .releaseCrawlHostGroup(ctx, chgId, chg.getSessionToken(), 0, true);
                return;
            }

            postFetchHandler = new PostFetchHandler(chg, frontier, false);
            postFetchHandler.postFetchFailure(err);
        } catch (Exception e) {
            LOG.warn("Error while processing timed out chg {}", chgId, e);
        } finally {
            if (postFetchHandler != null) {
                try {
                    postFetchHandler.postFetchFinally(true);
                } catch (Exception e) {
                    LOG.warn("Error in postFetchFinally for chg {}", chgId, e);
                }
            }
        }
    }

    private void runCheckPaused() {
        try {
            boolean desiredPaused = DbService.getInstance()
                    .getExecutionsAdapter()
                    .getDesiredPausedState();
            frontier.getCrawlQueueManager().pause(desiredPaused);
        } catch (DbException e) {
            LOG.warn("Could not read pause state", e);
        } catch (Exception e) {
            LOG.warn("Unexpected error in checkPaused", e);
        }
    }

    @Override
    public void close() throws InterruptedException {
        LOG.debug("Closing CrawlQueueWorker");
        executor.shutdown();
        if (!executor.awaitTermination(15, TimeUnit.SECONDS)) {
            LOG.warn("CrawlQueueWorker did not terminate in 15 seconds; forcing shutdownNow");
            executor.shutdownNow();
            if (!executor.awaitTermination(5, TimeUnit.SECONDS)) {
                LOG.error("CrawlQueueWorker still not terminated after shutdownNow");
            }
        }
        LOG.debug("CrawlQueueWorker closed");
    }
}

