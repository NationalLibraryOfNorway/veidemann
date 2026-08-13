package no.nb.nna.veidemann.frontier.db;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import java.util.function.Supplier;
import java.util.List;
import java.util.Map;
import java.util.Arrays;

import org.junit.jupiter.api.Test;

import redis.clients.jedis.Jedis;

class JobExecutionQueueCounterTest {

    @Test
    void readsBatchCountsWithOneHmgetAndMapsMissingOrInvalidValuesToZero() {
        Jedis jedis = mock(Jedis.class);
        @SuppressWarnings("unchecked")
        Supplier<Jedis> supplier = mock(Supplier.class);
        when(supplier.get()).thenReturn(jedis);
        when(jedis.hmget(CrawlQueueManager.JOB_EXECUTION_ID_COUNT_KEY, "one", "missing", "invalid"))
                .thenReturn(Arrays.asList("12", null, "broken"));

        Map<String, Long> counts = new JobExecutionQueueCounter(supplier)
                .counts(List.of("one", "missing", "invalid"));

        assertThat(counts).containsExactly(
                Map.entry("one", 12L),
                Map.entry("missing", 0L),
                Map.entry("invalid", 0L));
        verify(jedis).hmget(CrawlQueueManager.JOB_EXECUTION_ID_COUNT_KEY, "one", "missing", "invalid");
    }
}
