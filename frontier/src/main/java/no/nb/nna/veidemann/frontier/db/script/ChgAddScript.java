package no.nb.nna.veidemann.frontier.db.script;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_PREFIX;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_WAIT_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CRAWL_EXECUTION_ID_COUNT_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.QUEUE_COUNT_TOTAL_KEY;

import java.util.List;

import com.google.common.collect.ImmutableList;
import com.google.protobuf.Timestamp;
import com.google.protobuf.util.Timestamps;

public class ChgAddScript extends RedisJob<Long> {
    private final LuaScript chgAddScript;

    public ChgAddScript() {
        super("chgAddScript");
        this.chgAddScript = new LuaScript("chg_add.lua");
    }

    /**
     * Add URI to queue for a given CrawlHostGroup.
     *
     * @param busyTimeout if CHG is set to busy, this is the timeout
     * @return number of URIs in queue for this CrawlHostGroup
     */
    public long run(JedisContext ctx,
            String chgId,
            String crawlExecutionId,
            Timestamp earliestFetchTimestamp,
            long busyTimeout) {
        return execute(ctx, jedis -> {
            String chgKey = CHG_PREFIX + chgId;
            long readyTime = Timestamps.toMillis(earliestFetchTimestamp);
            String readyTimeString = Long.toString(readyTime);

            List<String> chgKeys = ImmutableList.of(
                    chgKey,
                    CHG_WAIT_KEY,
                    CRAWL_EXECUTION_ID_COUNT_KEY,
                    QUEUE_COUNT_TOTAL_KEY);

            List<String> chgArgs = ImmutableList.of(
                    readyTimeString,
                    crawlExecutionId,
                    chgId);

            return (Long) chgAddScript.runString(jedis, chgKeys, chgArgs);
        });
    }
}
