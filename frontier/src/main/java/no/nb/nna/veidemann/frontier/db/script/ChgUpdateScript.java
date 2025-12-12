package no.nb.nna.veidemann.frontier.db.script;

import java.util.Map;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import no.nb.nna.veidemann.api.frontier.v1.CrawlHostGroup;
import no.nb.nna.veidemann.frontier.db.CrawlQueueManager;

public class ChgUpdateScript extends RedisJob<Void> {
    private static final Logger LOG = LoggerFactory.getLogger(ChgUpdateScript.class);

    public ChgUpdateScript() {
        super("chgUpdate");
    }

    public void run(JedisContext ctx, CrawlHostGroup crawlHostGroup) {
        execute(ctx, jedis -> {
            String chgKey = CrawlQueueManager.CHG_PREFIX + crawlHostGroup.getId();
            Map<String, String> encoded = CrawlHostGroupCodec.encodeMap(crawlHostGroup);
            jedis.hset(chgKey, encoded);
            if (!crawlHostGroup.getSessionToken().isEmpty()) {
                jedis.hset(CrawlQueueManager.SESSION_TO_CHG_KEY,
                        crawlHostGroup.getSessionToken(),
                        crawlHostGroup.getId());
            }
            LOG.trace("Updated CHG {}: {}", crawlHostGroup.getId(), encoded);
            return null;
        });
    }
}
