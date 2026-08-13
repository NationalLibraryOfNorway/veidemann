package no.nb.nna.veidemann.frontier.db;

import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.function.Supplier;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.google.common.primitives.Longs;

import redis.clients.jedis.Jedis;

final class JobExecutionQueueCounter {
    private static final Logger LOG = LoggerFactory.getLogger(JobExecutionQueueCounter.class);

    private final Supplier<Jedis> jedisSupplier;

    JobExecutionQueueCounter(Supplier<Jedis> jedisSupplier) {
        this.jedisSupplier = jedisSupplier;
    }

    Map<String, Long> counts(List<String> jobExecutionIds) {
        Map<String, Long> counts = new LinkedHashMap<>();
        if (jobExecutionIds.isEmpty()) {
            return counts;
        }

        try (Jedis jedis = jedisSupplier.get()) {
            List<String> values = jedis.hmget(
                    CrawlQueueManager.JOB_EXECUTION_ID_COUNT_KEY,
                    jobExecutionIds.toArray(String[]::new));
            for (int index = 0; index < jobExecutionIds.size(); index++) {
                String jobExecutionId = jobExecutionIds.get(index);
                String value = values.get(index);
                Long parsed = value == null ? null : Longs.tryParse(value);
                if (parsed == null) {
                    if (value != null) {
                        LOG.warn("Invalid job execution count '{}' for jobExecutionId {}", value, jobExecutionId);
                    }
                    parsed = 0L;
                }
                counts.put(jobExecutionId, parsed);
            }
        }
        return counts;
    }
}
