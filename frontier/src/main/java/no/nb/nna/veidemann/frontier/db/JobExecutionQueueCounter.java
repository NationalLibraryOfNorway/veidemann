package no.nb.nna.veidemann.frontier.db;

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

    long count(String jobExecutionId) {
        if (jobExecutionId == null || jobExecutionId.isBlank()) {
            return 0L;
        }

        try (Jedis jedis = jedisSupplier.get()) {
            String count = jedis.hget(CrawlQueueManager.JOB_EXECUTION_ID_COUNT_KEY, jobExecutionId);
            if (count == null) {
                return 0L;
            }
            Long parsed = Longs.tryParse(count);
            if (parsed == null) {
                LOG.warn("Invalid job execution count '{}' for jobExecutionId {}", count, jobExecutionId);
                return 0L;
            }
            return parsed;
        }
    }
}
