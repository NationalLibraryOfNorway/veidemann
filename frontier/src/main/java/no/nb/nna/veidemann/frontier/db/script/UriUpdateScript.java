package no.nb.nna.veidemann.frontier.db.script;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.UEID;

import java.util.List;

import com.google.common.collect.ImmutableList;
import com.google.protobuf.Timestamp;

import no.nb.nna.veidemann.frontier.worker.QueuedUriWrapper;

public class UriUpdateScript extends RedisJob<Void> {
    private final LuaScript uriUpdateScript;

    public UriUpdateScript() {
        super("uriUpdateScript");
        this.uriUpdateScript = new LuaScript("uri_update.lua");
    }

    public void run(JedisContext ctx,
            QueuedUriWrapper qUri,
            Timestamp oldEarliestFetchTimestamp) {
        execute(ctx, jedis -> {
            String ueIdKey = UEID + qUri.getCrawlHostGroupId() + ":" + qUri.getExecutionId();

            String ueIdOldVal = String.format("%4d:%d:%s",
                    qUri.getQueuedUri().getSequence(),
                    oldEarliestFetchTimestamp.getSeconds(),
                    qUri.getQueuedUri().getId());

            String ueIdNewVal = String.format("%4d:%d:%s",
                    qUri.getQueuedUri().getSequence(),
                    qUri.getQueuedUri().getEarliestFetchTimeStamp().getSeconds(),
                    qUri.getQueuedUri().getId());

            List<String> keys = ImmutableList.of(ueIdKey);
            List<String> args = ImmutableList.of(ueIdOldVal, ueIdNewVal);
            uriUpdateScript.runString(jedis, keys, args);
            return null;
        });
    }
}
