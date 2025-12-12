package no.nb.nna.veidemann.frontier.db.script;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_PREFIX;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CRAWL_EXECUTION_ID_COUNT_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.QUEUE_COUNT_TOTAL_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.REMOVE_URI_QUEUE_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.UCHG;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.UEID;

import java.util.List;

import com.google.common.collect.ImmutableList;

public class UriRemoveScript extends RedisJob<Long> {
    private final LuaScript uriRemoveScript;

    public UriRemoveScript() {
        super("uriRemoveScript");
        this.uriRemoveScript = new LuaScript("uri_remove.lua");
    }

    public long run(JedisContext ctx,
            String uriId,
            String chgId,
            String eid,
            long sequence,
            long fetchTime,
            boolean deleteUri) {
        return execute(ctx, jedis -> {
            if (uriId == null || uriId.isEmpty()) {
                throw new IllegalArgumentException("Missing uriId");
            }

            String ueIdKey = UEID + chgId + ":" + eid;
            String ueIdVal = String.format("%4d:%d:%s",
                    sequence,
                    fetchTime,
                    uriId);
            String uchgKey = UCHG + chgId;
            String chgKey = CHG_PREFIX + chgId;

            String removeQueue = deleteUri ? uriId : "";

            List<String> keys = ImmutableList.of(
                    ueIdKey,
                    uchgKey,
                    chgKey,
                    CRAWL_EXECUTION_ID_COUNT_KEY,
                    QUEUE_COUNT_TOTAL_KEY,
                    REMOVE_URI_QUEUE_KEY);

            List<String> args = ImmutableList.of(
                    ueIdVal,
                    eid,
                    removeQueue);

            return (Long) uriRemoveScript.runString(jedis, keys, args);
        });
    }
}
