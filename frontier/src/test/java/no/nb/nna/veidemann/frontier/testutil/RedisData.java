package no.nb.nna.veidemann.frontier.testutil;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_BUSY_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_PREFIX;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_READY_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_WAIT_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.SESSION_TO_CHG_KEY;

import java.util.List;
import java.util.Map;
import java.util.function.Supplier;
import java.util.stream.Collectors;

import no.nb.nna.veidemann.api.frontier.v1.CrawlHostGroup;
import no.nb.nna.veidemann.frontier.db.script.CrawlHostGroupCodec;
import redis.clients.jedis.Jedis;
import redis.clients.jedis.resps.Tuple;

public class RedisData {
    final Supplier<Jedis> jedisSupplier;

    public RedisData(Supplier<Jedis> jedisSupplier) {
        this.jedisSupplier = jedisSupplier;
    }

    public long getQueueCountTotal() {
        try (Jedis jedis = jedisSupplier.get()) {
            String val = jedis.get("QCT");
            return Long.parseLong(val);
        } catch (NumberFormatException e) {
            return 0L;
        }
    }

    public Map<String, Long> getCrawlExecutionCounts() {
        try (Jedis jedis = jedisSupplier.get()) {
            return jedis.hgetAll("EIDC").entrySet().stream()
                    .collect(Collectors.toUnmodifiableMap(e -> e.getKey(), e -> Long.parseLong(e.getValue())));
        }
    }

    public Map<String, CrawlHostGroup> getCrawlHostGroups() {
        try (Jedis jedis = jedisSupplier.get()) {
            return jedis.keys(CHG_PREFIX + "*").stream()
                    .map(k -> CrawlHostGroupCodec.decode(k.substring(CHG_PREFIX.length()), jedis.hgetAll(k)))
                    .collect(Collectors.toUnmodifiableMap(chg -> chg.getId(), chg -> chg));
        }
    }

    public List<Tuple> getWaitQueue() {
        try (Jedis jedis = jedisSupplier.get()) {
            return jedis.zrangeWithScores(CHG_WAIT_KEY, 0, -1);
        }
    }

    public List<Tuple> getBusyQueue() {
        try (Jedis jedis = jedisSupplier.get()) {
            return jedis.zrangeWithScores(CHG_BUSY_KEY, 0, -1);
        }
    }

    public List<String> getReadyQueue() {
        try (Jedis jedis = jedisSupplier.get()) {
            return jedis.lrange(CHG_READY_KEY, 0, -1);
        }
    }

    public Map<String, String> getSessionTokens() {
        try (Jedis jedis = jedisSupplier.get()) {
            return jedis.hgetAll(SESSION_TO_CHG_KEY);
        }
    }
}
