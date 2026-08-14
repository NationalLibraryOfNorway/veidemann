package no.nb.nna.veidemann.frontier.testutil;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_BUSY_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_PREFIX;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_READY_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_WAIT_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CRAWL_EXECUTION_JOB_EXECUTION_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.JOB_EXECUTION_ID_COUNT_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.SESSION_TO_CHG_KEY;

import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

import no.nb.nna.veidemann.api.frontier.v1.CrawlHostGroup;
import no.nb.nna.veidemann.frontier.db.script.CrawlHostGroupCodec;
import redis.clients.jedis.UnifiedJedis;
import redis.clients.jedis.resps.Tuple;

public class RedisData {
    final UnifiedJedis redisClient;

    public RedisData(UnifiedJedis redisClient) {
        this.redisClient = redisClient;
    }

    public long getQueueCountTotal() {
        try {
            String val = redisClient.get("QCT");
            return Long.parseLong(val);
        } catch (NumberFormatException e) {
            return 0L;
        }
    }

    public Map<String, Long> getCrawlExecutionCounts() {
        return redisClient.hgetAll("EIDC").entrySet().stream()
                .collect(Collectors.toUnmodifiableMap(e -> e.getKey(), e -> Long.parseLong(e.getValue())));
    }

    public Map<String, Long> getJobExecutionCounts() {
        return redisClient.hgetAll(JOB_EXECUTION_ID_COUNT_KEY).entrySet().stream()
                .collect(Collectors.toUnmodifiableMap(e -> e.getKey(), e -> Long.parseLong(e.getValue())));
    }

    public Map<String, String> getCrawlExecutionJobExecutions() {
        return redisClient.hgetAll(CRAWL_EXECUTION_JOB_EXECUTION_KEY);
    }

    public Map<String, CrawlHostGroup> getCrawlHostGroups() {
        return redisClient.keys(CHG_PREFIX + "*").stream()
                .map(k -> CrawlHostGroupCodec.decode(k.substring(CHG_PREFIX.length()), redisClient.hgetAll(k)))
                .collect(Collectors.toUnmodifiableMap(chg -> chg.getId(), chg -> chg));
    }

    public List<Tuple> getWaitQueue() {
        return redisClient.zrangeWithScores(CHG_WAIT_KEY, 0, -1);
    }

    public List<Tuple> getBusyQueue() {
        return redisClient.zrangeWithScores(CHG_BUSY_KEY, 0, -1);
    }

    public List<String> getReadyQueue() {
        return redisClient.lrange(CHG_READY_KEY, 0, -1);
    }

    public Map<String, String> getSessionTokens() {
        return redisClient.hgetAll(SESSION_TO_CHG_KEY);
    }
}
