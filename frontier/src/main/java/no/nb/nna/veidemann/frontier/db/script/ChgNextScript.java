package no.nb.nna.veidemann.frontier.db.script;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_BUSY_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_PREFIX;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_READY_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_WAIT_KEY;

import java.util.List;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.google.common.collect.ImmutableList;
import com.google.common.primitives.Longs;

import no.nb.nna.veidemann.api.frontier.v1.CrawlHostGroup;
import redis.clients.jedis.resps.Tuple;

public class ChgNextScript extends RedisJob<CrawlHostGroup> {
    private static final Logger LOG = LoggerFactory.getLogger(ChgNextScript.class);
    private static final int DEFAULT_WAIT_FOR_READY_TIMEOUT = 5;

    private final LuaScript chgNextScript;
    private int waitForReadyTimeout = DEFAULT_WAIT_FOR_READY_TIMEOUT;

    public ChgNextScript() {
        super("chgNextScript");
        this.chgNextScript = new LuaScript("chg_next.lua");
    }

    public ChgNextScript withWaitForReadyTimeout(int waitForReadyTimeout) {
        this.waitForReadyTimeout = waitForReadyTimeout;
        return this;
    }

    public CrawlHostGroup run(JedisContext ctx, long busyTimeout) {
        return execute(ctx, jedis -> {
            List<String> res = jedis.blpop(waitForReadyTimeout, CHG_READY_KEY);
            if (res == null) {
                if (LOG.isTraceEnabled()) {
                    long now = System.currentTimeMillis();
                    for (Tuple t : jedis.zrangeWithScores(CHG_WAIT_KEY, 0, 0)) {
                        LOG.trace("No ready CHG. Idle CHG {} waiting for {} ms",
                                t.getElement(), (long) t.getScore() - now);
                    }
                    for (Tuple t : jedis.zrangeWithScores(CHG_BUSY_KEY, 0, 0)) {
                        LOG.trace("No ready CHG. Busy CHG {} will be released in {} ms if fetch is too slow",
                                t.getElement(), (long) t.getScore() - now);
                    }
                }
                return null;
            } else {
                LOG.trace("BLPOP {} {} => {}", waitForReadyTimeout, CHG_READY_KEY, res);
            }

            String chgId = res.get(1);
            String chgKey = CHG_PREFIX + chgId;
            List<String> keys = ImmutableList.of(CHG_BUSY_KEY, chgKey);
            List<String> args = ImmutableList.of(chgId, String.valueOf(System.currentTimeMillis() + busyTimeout));
            String result = (String) chgNextScript.runString(jedis, keys, args);

            CrawlHostGroup.Builder chg = CrawlHostGroup.newBuilder().setId(chgId);
            Long count = Longs.tryParse(result);
            if (count != null) {
                chg.setQueuedUriCount(count);
            }
            return chg.build();
        });
    }
}
