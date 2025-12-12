package no.nb.nna.veidemann.frontier.db.script;

import java.util.List;

import com.google.common.collect.ImmutableList;

import no.nb.nna.veidemann.frontier.db.CrawlQueueManager;

public class ChgUpdateBusyTimeoutScript extends RedisJob<Long> {
    private final LuaScript chgUpdateBusyTimeoutScript;

    public ChgUpdateBusyTimeoutScript() {
        super("chgUpdateBusyTimeoutScript");
        this.chgUpdateBusyTimeoutScript = new LuaScript("chg_update_busy_timeout.lua");
    }

    public Long run(JedisContext ctx, String crawlHostGroupId, String sessionToken, Long timeoutTimeMs) {
        String chgKey = CrawlQueueManager.CHG_PREFIX + crawlHostGroupId;

        return execute(ctx, jedis -> {
            List<String> keys = ImmutableList.of(CrawlQueueManager.CHG_BUSY_KEY, chgKey);
            List<String> args = ImmutableList.of(
                    String.valueOf(timeoutTimeMs),
                    crawlHostGroupId,
                    sessionToken);
            return (Long) chgUpdateBusyTimeoutScript.runString(jedis, keys, args);
        });
    }
}
