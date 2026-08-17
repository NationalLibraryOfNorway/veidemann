package no.nb.nna.veidemann.frontier.db.script;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_PREFIX;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_WAIT_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CRAWL_EXECUTION_ID_COUNT_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CRAWL_EXECUTION_JOB_EXECUTION_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CRAWL_EXECUTION_FINALIZE_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.JOB_EXECUTION_ID_COUNT_KEY;
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
    public long run(RedisContext ctx,
            String chgId,
            String crawlExecutionId,
            String jobExecutionId,
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
                    JOB_EXECUTION_ID_COUNT_KEY,
                    CRAWL_EXECUTION_JOB_EXECUTION_KEY,
                    QUEUE_COUNT_TOTAL_KEY,
                    CRAWL_EXECUTION_FINALIZE_KEY);

            List<String> chgArgs = ImmutableList.of(
                    readyTimeString,
                    crawlExecutionId,
                    jobExecutionId,
                    chgId);

            return (Long) chgAddScript.runString(jedis, chgKeys, chgArgs);
        });
    }
}
