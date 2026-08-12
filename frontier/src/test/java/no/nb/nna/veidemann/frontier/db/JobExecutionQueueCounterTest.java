package no.nb.nna.veidemann.frontier.db;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import java.util.function.Supplier;

import org.junit.jupiter.api.Test;

import redis.clients.jedis.Jedis;

class JobExecutionQueueCounterTest {

    @Test
    void returnsStoredCount() {
        Jedis jedis = mock(Jedis.class);
        @SuppressWarnings("unchecked")
        Supplier<Jedis> supplier = mock(Supplier.class);
        when(supplier.get()).thenReturn(jedis);
        when(jedis.hget(CrawlQueueManager.JOB_EXECUTION_ID_COUNT_KEY, "job-execution")).thenReturn("42");

        long count = new JobExecutionQueueCounter(supplier).count("job-execution");

        assertThat(count).isEqualTo(42L);
    }

    @Test
    void returnsZeroForMissingOrInvalidCounts() {
        Jedis jedis = mock(Jedis.class);
        @SuppressWarnings("unchecked")
        Supplier<Jedis> supplier = mock(Supplier.class);
        when(supplier.get()).thenReturn(jedis);
        when(jedis.hget(CrawlQueueManager.JOB_EXECUTION_ID_COUNT_KEY, "invalid")).thenReturn("not-a-number");

        JobExecutionQueueCounter counter = new JobExecutionQueueCounter(supplier);

        assertThat(counter.count("missing")).isZero();
        assertThat(counter.count("invalid")).isZero();
    }

    @Test
    void returnsZeroWithoutRedisForBlankIds() {
        @SuppressWarnings("unchecked")
        Supplier<Jedis> supplier = mock(Supplier.class);
        JobExecutionQueueCounter counter = new JobExecutionQueueCounter(supplier);

        assertThat(counter.count("")).isZero();
        assertThat(counter.count("  ")).isZero();
        assertThat(counter.count(null)).isZero();
        verify(supplier, never()).get();
    }
}
