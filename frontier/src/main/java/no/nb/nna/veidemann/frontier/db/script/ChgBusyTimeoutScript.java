package no.nb.nna.veidemann.frontier.db.script;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_BUSY_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.SESSION_TO_CHG_KEY;

import java.util.List;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.google.common.collect.ImmutableList;

public class ChgBusyTimeoutScript extends RedisJob<List<String>> {
    private static final Logger LOG = LoggerFactory.getLogger(ChgBusyTimeoutScript.class);

    private final LuaScript chgBusyTimeoutScript;

    public ChgBusyTimeoutScript() {
        super("chgBusyTimeoutScript");
        this.chgBusyTimeoutScript = new LuaScript("chg_busy_timeout.lua");
    }

    /**
     * Move CrawlHostGroups which have timed out in busy state to ready state.
     *
     * @return list of CHG IDs moved from busy to ready
     */
    public List<String> run(JedisContext ctx) {
        return execute(ctx, jedis -> {
            List<String> keys = ImmutableList.of(CHG_BUSY_KEY, SESSION_TO_CHG_KEY);
            List<String> args = ImmutableList.of(String.valueOf(System.currentTimeMillis()));
            @SuppressWarnings("unchecked")
            List<String> result = (List<String>) chgBusyTimeoutScript.runString(jedis, keys, args);
            LOG.trace("ChgBusyTimeout result: {}", result);
            return result;
        });
    }
}
