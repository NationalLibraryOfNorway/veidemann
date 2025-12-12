package no.nb.nna.veidemann.frontier.db.script;

import java.util.Map;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import no.nb.nna.veidemann.api.frontier.v1.CrawlHostGroup;
import no.nb.nna.veidemann.frontier.db.CrawlQueueManager;

public class ChgGetScript extends RedisJob<CrawlHostGroup> {
    private static final Logger LOG = LoggerFactory.getLogger(ChgGetScript.class);

    public ChgGetScript() {
        super("chgGetScript");
    }

    public CrawlHostGroup run(JedisContext ctx, String crawlHostGroupId) {
        return execute(ctx, jedis -> {
            String key = CrawlQueueManager.CHG_PREFIX + crawlHostGroupId;
            Map<String, String> encoded = jedis.hgetAll(key);
            LOG.trace("HGETALL {}, RESULT: {}", key, encoded);
            return CrawlHostGroupCodec.decode(crawlHostGroupId, encoded);
        });
    }
}
