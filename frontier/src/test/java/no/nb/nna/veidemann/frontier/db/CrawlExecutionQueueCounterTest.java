package no.nb.nna.veidemann.frontier.db;

import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;
import redis.clients.jedis.Jedis;

import java.util.Arrays;
import java.util.List;
import java.util.function.Supplier;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class CrawlExecutionQueueCounterTest {

    @Test
    void aggregatesUniqueNonBlankIdsWithOneHmget() {
        Jedis jedis = mock(Jedis.class);
        @SuppressWarnings("unchecked")
        Supplier<Jedis> supplier = mock(Supplier.class);
        when(supplier.get()).thenReturn(jedis);
        when(jedis.hmget(eq(CrawlQueueManager.CRAWL_EXECUTION_ID_COUNT_KEY), any(String[].class)))
                .thenReturn(Arrays.asList("4", "invalid", null));
        CrawlExecutionQueueCounter counter = new CrawlExecutionQueueCounter(supplier);

        long count = counter.count(List.of("first", "", "first", "invalid", "missing"));

        assertThat(count).isEqualTo(4L);
        ArgumentCaptor<String[]> ids = ArgumentCaptor.forClass(String[].class);
        verify(jedis).hmget(eq(CrawlQueueManager.CRAWL_EXECUTION_ID_COUNT_KEY), ids.capture());
        assertThat(ids.getValue()).containsExactly("first", "invalid", "missing");
    }

    @Test
    void returnsZeroWithoutRedisForAnEmptyEffectiveRequest() {
        @SuppressWarnings("unchecked")
        Supplier<Jedis> supplier = mock(Supplier.class);
        CrawlExecutionQueueCounter counter = new CrawlExecutionQueueCounter(supplier);

        assertThat(counter.count(List.of("", "  "))).isZero();
        verify(supplier, never()).get();
    }
}
