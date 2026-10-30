package no.nb.nna.veidemann.frontier.db;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CRAWL_EXECUTION_FINALIZE_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.JOB_EXECUTION_FINALIZE_KEY;
import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyDouble;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.doAnswer;
import static org.mockito.Mockito.doReturn;
import static org.mockito.Mockito.doThrow;
import static org.mockito.Mockito.inOrder;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import java.time.Clock;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

import com.google.protobuf.Timestamp;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus.State;
import no.nb.nna.veidemann.api.frontier.v1.JobExecutionStatus;
import no.nb.nna.veidemann.commons.db.DbQueryException;
import no.nb.nna.veidemann.commons.db.ExecutionsAdapter;
import no.nb.nna.veidemann.commons.db.FrontierAdapter;
import no.nb.nna.veidemann.frontier.worker.Frontier;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;
import redis.clients.jedis.UnifiedJedis;
import redis.clients.jedis.params.ZAddParams;
import redis.clients.jedis.params.ZRangeParams;

class CrawlExecutionReconcilerTest {
    private final Frontier frontier = mock(Frontier.class);
    private final FrontierAdapter adapter = mock(FrontierAdapter.class);
    private final ExecutionsAdapter executions = mock(ExecutionsAdapter.class);
    private final CrawlQueueManager queue = mock(CrawlQueueManager.class);
    private final UnifiedJedis redis = mock(UnifiedJedis.class);
    private final Clock clock = mock(Clock.class);
    private final Map<String, Long> pendingJobs = new LinkedHashMap<>();
    private final Map<String, CrawlExecutionStatus> pendingExecutions = new LinkedHashMap<>();

    @BeforeEach
    void setUp() throws Exception {
        when(clock.millis()).thenReturn(1000L);
        when(frontier.getFrontierAdapter()).thenReturn(adapter);
        when(frontier.getExecutionsAdapter()).thenReturn(executions);
        when(frontier.getCrawlQueueManager()).thenReturn(queue);
        when(executions.getCrawlExecutionStatus(anyString()))
                .thenAnswer(call -> pendingExecutions.get(call.getArgument(0)));
        when(redis.zrange(eq(CRAWL_EXECUTION_FINALIZE_KEY), any(ZRangeParams.class)))
                .thenAnswer(call -> new ArrayList<>(pendingExecutions.keySet()));
        when(redis.zrange(eq(JOB_EXECUTION_FINALIZE_KEY), any(ZRangeParams.class)))
                .thenAnswer(call -> pendingJobs.entrySet().stream()
                        .filter(entry -> entry.getValue() <= clock.millis()).map(Map.Entry::getKey).toList());
        when(redis.zrem(eq(CRAWL_EXECUTION_FINALIZE_KEY), anyString())).thenAnswer(call ->
                pendingExecutions.remove(call.getArgument(1)) == null ? 0L : 1L);
        when(redis.zrem(eq(JOB_EXECUTION_FINALIZE_KEY), anyString())).thenAnswer(call ->
                pendingJobs.remove(call.getArgument(1)) == null ? 0L : 1L);
        when(redis.zadd(eq(JOB_EXECUTION_FINALIZE_KEY), anyDouble(), anyString(), eq(ZAddParams.zAddParams().xx())))
                .thenAnswer(call -> {
                    String jobId = call.getArgument(2);
                    pendingJobs.computeIfPresent(jobId, (key, old) -> ((Double) call.getArgument(1)).longValue());
                    return 0L;
                });
        doAnswer(call -> {
            pendingJobs.putIfAbsent(call.getArgument(0), clock.millis());
            return null;
        }).when(queue).scheduleJobExecutionFinalization(anyString());
    }

    @Test
    void coalescesTerminalChildrenAndKeepsTheCooldownWhileTheJobIsRunning() throws Exception {
        when(adapter.hasRunningCrawlExecutions("job")).thenReturn(true);
        for (int i = 0; i < 40; i++) {
            terminalExecution("execution-" + i);
        }
        try (Harness harness = reconciler()) {
            harness.tick().run();

            assertThat(pendingExecutions).isEmpty();
            assertThat(pendingJobs).containsExactly(Map.entry("job", 61000L));
            verify(adapter).hasRunningCrawlExecutions("job");
            verify(adapter, never()).getJobExecutionAggregate(anyString());
            var order = inOrder(queue, redis);
            order.verify(queue).scheduleJobExecutionFinalization("job");
            order.verify(redis).zrem(CRAWL_EXECUTION_FINALIZE_KEY, "execution-0");

            when(clock.millis()).thenReturn(2000L);
            terminalExecution("another-execution");
            harness.tick().run();
            assertThat(pendingJobs).containsExactly(Map.entry("job", 61000L));
            verify(adapter, times(1)).hasRunningCrawlExecutions("job");

            when(clock.millis()).thenReturn(61000L);
            harness.tick().run();
            verify(adapter, times(2)).hasRunningCrawlExecutions("job");
            verify(adapter, never()).getJobExecutionAggregate(anyString());
        }
    }

    @Test
    void defersQueryFailuresAndRecoversFromTheDurableRequestAfterRestart() throws Exception {
        pendingJobs.put("job", 0L);
        when(adapter.hasRunningCrawlExecutions("job")).thenThrow(new DbQueryException("disconnected"));
        try (Harness harness = reconciler()) {
            harness.tick().run();
            assertThat(pendingJobs).containsExactly(Map.entry("job", 61000L));
            harness.tick().run();
            verify(adapter).hasRunningCrawlExecutions("job");
        }

        when(clock.millis()).thenReturn(61000L);
        doReturn(false).when(adapter).hasRunningCrawlExecutions("job");
        when(adapter.getJobExecutionAggregate("job")).thenReturn(JobExecutionStatus.newBuilder()
                .setId("job").setDocumentsCrawled(37).putExecutionsState(State.FINISHED.name(), 2).build());
        when(adapter.getJobExecutionStatus("job")).thenReturn(JobExecutionStatus.newBuilder()
                .setId("job").setState(JobExecutionStatus.State.RUNNING).build());
        try (Harness restarted = reconciler()) {
            restarted.tick().run();

            assertThat(pendingJobs).isEmpty();
            verify(adapter).getJobExecutionAggregate("job");
            var saved = ArgumentCaptor.forClass(JobExecutionStatus.class);
            var order = inOrder(adapter, redis);
            order.verify(adapter).saveJobExecutionStatus(saved.capture());
            order.verify(redis).zrem(JOB_EXECUTION_FINALIZE_KEY, "job");
            assertThat(saved.getValue().getDocumentsCrawled()).isEqualTo(37);
            assertThat(saved.getValue().hasEndTime()).isTrue();
        }
    }

    @Test
    void keepsTheExecutionSignalUntilItsJobRepairRequestIsDurable() throws Exception {
        terminalExecution("execution");
        doThrow(new IllegalStateException("Redis unavailable"))
                .when(queue).scheduleJobExecutionFinalization("job");
        try (Harness harness = reconciler()) {
            harness.tick().run();

            assertThat(pendingExecutions).containsKey("execution");
            verify(redis, never()).zrem(CRAWL_EXECUTION_FINALIZE_KEY, "execution");
            verify(adapter, never()).getJobExecutionAggregate(anyString());
        }
    }

    @Test
    void deferralDoesNotRecreateARequestRemovedByNormalCompletion() throws Exception {
        pendingJobs.put("job", 0L);
        when(adapter.hasRunningCrawlExecutions("job")).thenAnswer(call -> {
            pendingJobs.remove("job");
            return true;
        });
        try (Harness harness = reconciler()) {
            harness.tick().run();

            assertThat(pendingJobs).isEmpty();
            verify(redis).zadd(JOB_EXECUTION_FINALIZE_KEY, 61000D, "job", ZAddParams.zAddParams().xx());
            verify(adapter, never()).getJobExecutionAggregate(anyString());
        }
    }

    private void terminalExecution(String id) {
        pendingExecutions.put(id, CrawlExecutionStatus.newBuilder().setId(id).setJobExecutionId("job")
                .setState(State.FINISHED).setEndTime(Timestamp.newBuilder().setSeconds(1)).build());
    }

    private Harness reconciler() {
        ScheduledExecutorService scheduler = mock(ScheduledExecutorService.class);
        CrawlExecutionReconciler reconciler = new CrawlExecutionReconciler(
                frontier, redis, clock, mock(ExecutorService.class), scheduler);
        var tick = ArgumentCaptor.forClass(Runnable.class);
        verify(scheduler).scheduleWithFixedDelay(tick.capture(), eq(1L), eq(1L), eq(TimeUnit.SECONDS));
        return new Harness(reconciler, tick.getValue());
    }

    private record Harness(CrawlExecutionReconciler reconciler, Runnable tick) implements AutoCloseable {
        @Override
        public void close() throws Exception {
            reconciler.close();
        }
    }
}
