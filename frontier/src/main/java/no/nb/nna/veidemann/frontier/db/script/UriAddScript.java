package no.nb.nna.veidemann.frontier.db.script;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.UCHG;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.UEID;

import java.util.List;
import java.util.Locale;

import com.google.common.collect.ImmutableList;

import no.nb.nna.veidemann.api.frontier.v1.QueuedUri;

public class UriAddScript extends RedisJob<Boolean> {
    private final LuaScript uriAddScript;

    public UriAddScript() {
        super("uriAddScript");
        this.uriAddScript = new LuaScript("uri_add.lua");
    }

    /**
     * Add URI to queue.
     */
    public void run(JedisContext ctx, QueuedUri qUri) {
        execute(ctx, jedis -> {
            String chgId = qUri.getCrawlHostGroupId();

            String ueIdKey = String.format("%s%s:%s",
                    UEID,
                    chgId,
                    qUri.getExecutionId());
            String ueIdVal = String.format("%4d:%d:%s",
                    qUri.getSequence(),
                    qUri.getEarliestFetchTimeStamp().getSeconds(),
                    qUri.getId());

            String uchgKey = UCHG + chgId;
            double priorityWeight = qUri.getPriorityWeight();

            // If this is a seed, up the priority
            if (qUri.getDiscoveryPath().isEmpty()) {
                priorityWeight += 100d;
            }
            String weight = String.format(Locale.ENGLISH, "%1.2f", priorityWeight);

            String eid = qUri.getExecutionId();
            List<String> keys = ImmutableList.of(ueIdKey, uchgKey);
            List<String> args = ImmutableList.of(ueIdVal, weight, eid);

            uriAddScript.runString(jedis, keys, args);
            return null;
        });
    }
}
