package no.nb.nna.veidemann.frontier.db.script;

import com.google.common.collect.ImmutableList;

import java.util.List;

public class ChgDelayedQueueScript extends RedisJob<Long> {
    private final LuaScript chgDelayedQueueScript;

    public ChgDelayedQueueScript() {
        super("chgDelayedQueueScript");
        this.chgDelayedQueueScript = new LuaScript("chg_delayed_queue.lua");
    }

    public Long run(JedisContext ctx, String fromQueue, String toQueue) {
        return execute(ctx, jedis -> {
            List<String> keys = ImmutableList.of(fromQueue, toQueue);
            List<String> args = ImmutableList.of(String.valueOf(System.currentTimeMillis()));
            return (Long) chgDelayedQueueScript.runString(jedis, keys, args);
        });
    }
}
