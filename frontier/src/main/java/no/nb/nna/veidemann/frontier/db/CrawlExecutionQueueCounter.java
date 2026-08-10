package no.nb.nna.veidemann.frontier.db;

import com.google.common.primitives.Longs;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import redis.clients.jedis.Jedis;

import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;
import java.util.function.Supplier;

final class CrawlExecutionQueueCounter {
    private static final Logger LOG = LoggerFactory.getLogger(CrawlExecutionQueueCounter.class);

    private final Supplier<Jedis> jedisSupplier;

    CrawlExecutionQueueCounter(Supplier<Jedis> jedisSupplier) {
        this.jedisSupplier = jedisSupplier;
    }

    long count(List<String> executionIds) {
        Set<String> uniqueIds = new LinkedHashSet<>();
        for (String executionId : executionIds) {
            if (executionId != null && !executionId.isBlank()) {
                uniqueIds.add(executionId);
            }
        }
        if (uniqueIds.isEmpty()) {
            return 0L;
        }

        try (Jedis jedis = jedisSupplier.get()) {
            List<String> counts = jedis.hmget(
                    CrawlQueueManager.CRAWL_EXECUTION_ID_COUNT_KEY,
                    uniqueIds.toArray(String[]::new));
            long total = 0L;
            int index = 0;
            for (String executionId : uniqueIds) {
                String count = counts.get(index++);
                if (count == null) {
                    continue;
                }
                Long parsed = Longs.tryParse(count);
                if (parsed == null) {
                    LOG.warn("Invalid crawl execution count '{}' for executionId {}", count, executionId);
                    continue;
                }
                total += parsed;
            }
            return total;
        }
    }
}
