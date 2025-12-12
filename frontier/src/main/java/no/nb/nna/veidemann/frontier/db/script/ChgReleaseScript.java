package no.nb.nna.veidemann.frontier.db.script;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_BUSY_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_WAIT_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.SESSION_TO_CHG_KEY;

import java.util.List;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.google.common.collect.ImmutableList;

import no.nb.nna.veidemann.frontier.db.CrawlQueueManager;
import redis.clients.jedis.exceptions.JedisDataException;

public class ChgReleaseScript extends RedisJob<Long> {
    private static final Logger LOG = LoggerFactory.getLogger(ChgReleaseScript.class);

    private final LuaScript chgReleaseScript;

    public ChgReleaseScript() {
        super("chgReleaseScript");
        this.chgReleaseScript = new LuaScript("chg_release.lua");
    }

    /**
     * Release a busy CrawlHostGroup.
     *
     * Moves CHG from busy queue to wait queue and removes the session token. If CHG
     * should be released
     * because of timeout while waiting for harvester, then isTimeout must be true.
     * In that case the CHG
     * is already removed from busy queue and the Lua script handles that.
     */
    public Long run(JedisContext ctx,
            String crawlHostGroupId,
            String sessionToken,
            long nextFetchDelayMs,
            boolean isTimeout) {

        if (nextFetchDelayMs < 10) {
            nextFetchDelayMs = 10;
        }
        String chgKey = CrawlQueueManager.CHG_PREFIX + crawlHostGroupId;
        long waitTime = System.currentTimeMillis() + nextFetchDelayMs;

        List<String> keys = ImmutableList.of(
                CHG_BUSY_KEY,
                CHG_WAIT_KEY,
                chgKey,
                SESSION_TO_CHG_KEY);

        List<String> args = ImmutableList.of(
                String.valueOf(waitTime),
                crawlHostGroupId,
                sessionToken,
                String.valueOf(isTimeout));

        return execute(ctx, jedis -> {
            try {
                String result = (String) chgReleaseScript.runString(jedis, keys, args);
                return Long.parseLong(result);
            } catch (JedisDataException e) {
                LOG.warn("CHG release script error for {}: {}", crawlHostGroupId, e.getMessage());
                return 0L;
            }
        });
    }
}
