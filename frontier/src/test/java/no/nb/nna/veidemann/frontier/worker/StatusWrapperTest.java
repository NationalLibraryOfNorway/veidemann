package no.nb.nna.veidemann.frontier.worker;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.inOrder;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import com.google.protobuf.Timestamp;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus.State;
import no.nb.nna.veidemann.api.frontier.v1.JobExecutionStatus;
import no.nb.nna.veidemann.commons.db.CrawlExecutionStatusUpdate;
import no.nb.nna.veidemann.commons.db.FrontierAdapter;
import no.nb.nna.veidemann.frontier.db.CrawlQueueManager;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;

class StatusWrapperTest {
    private final Frontier frontier = mock(Frontier.class);
    private final FrontierAdapter adapter = mock(FrontierAdapter.class);
    private final CrawlQueueManager queue = mock(CrawlQueueManager.class);

    @BeforeEach
    void setUp() {
        when(frontier.getFrontierAdapter()).thenReturn(adapter);
        when(frontier.getCrawlQueueManager()).thenReturn(queue);
    }

    @Test
    void repairSkipsAggregateWhileAnyPersistedChildIsRunning() throws Exception {
        when(adapter.hasRunningCrawlExecutions("job")).thenReturn(true);

        assertThat(StatusWrapper.repairJobExecution(frontier, "job")).isFalse();

        verify(adapter, never()).getJobExecutionAggregate(anyString());
        verify(adapter, never()).saveJobExecutionStatus(any());
        verify(queue, never()).removeRedisJobExecution(anyString());
    }

    @Test
    void repairStillRejectsActiveChildrenFoundDuringReconstruction() throws Exception {
        when(adapter.getJobExecutionAggregate("job")).thenReturn(JobExecutionStatus.newBuilder()
                .putExecutionsState(State.FETCHING.name(), 1).build());

        assertThat(StatusWrapper.repairJobExecution(frontier, "job")).isFalse();

        verify(adapter).hasRunningCrawlExecutions("job");
        verify(adapter, never()).saveJobExecutionStatus(any());
    }

    @Test
    void repairDoesNotFinalizeAnEmptyJob() throws Exception {
        when(adapter.getJobExecutionAggregate("job")).thenReturn(JobExecutionStatus.getDefaultInstance());

        assertThat(StatusWrapper.repairJobExecution(frontier, "job")).isFalse();

        verify(adapter, never()).saveJobExecutionStatus(any());
    }

    @Test
    void repairSavesAuthoritativeCountersBeforeRemovingRedisState() throws Exception {
        JobExecutionStatus aggregate = finishedAggregate();
        when(adapter.getJobExecutionAggregate("job")).thenReturn(aggregate);
        when(adapter.getJobExecutionStatus("job")).thenReturn(runningJob());

        assertThat(StatusWrapper.repairJobExecution(frontier, "job")).isTrue();

        var saved = ArgumentCaptor.forClass(JobExecutionStatus.class);
        var order = inOrder(adapter, queue);
        order.verify(adapter).hasRunningCrawlExecutions("job");
        order.verify(adapter).getJobExecutionAggregate("job");
        order.verify(adapter).saveJobExecutionStatus(saved.capture());
        order.verify(queue).removeRedisJobExecution("job");
        assertThat(saved.getValue().getState()).isEqualTo(JobExecutionStatus.State.FINISHED);
        assertThat(saved.getValue().hasEndTime()).isTrue();
        assertThat(saved.getValue().getDocumentsCrawled()).isEqualTo(37);
        assertThat(saved.getValue().getExecutionsStateMap()).isEqualTo(aggregate.getExecutionsStateMap());
    }

    @Test
    void missingRedisAggregateQueuesRepairWithoutReconstructingInline() throws Exception {
        completeExecution();

        verify(queue).scheduleJobExecutionFinalization("job");
        verify(adapter, never()).hasRunningCrawlExecutions(anyString());
        verify(adapter, never()).getJobExecutionAggregate(anyString());
        verify(adapter, never()).saveJobExecutionStatus(any());
    }

    @Test
    void normalCompletionKeepsUsingTheAvailableRedisAggregate() throws Exception {
        when(queue.getTempJobExecutionStatus("job")).thenReturn(finishedAggregate());
        when(adapter.getJobExecutionStatus("job")).thenReturn(runningJob());

        completeExecution();

        verify(adapter).saveJobExecutionStatus(any());
        verify(queue).removeRedisJobExecution("job");
        verify(queue, never()).scheduleJobExecutionFinalization(anyString());
        verify(adapter, never()).hasRunningCrawlExecutions(anyString());
        verify(adapter, never()).getJobExecutionAggregate(anyString());
    }

    private void completeExecution() throws Exception {
        CrawlExecutionStatus previous = CrawlExecutionStatus.newBuilder()
                .setId("execution").setJobExecutionId("job").setState(State.FETCHING).build();
        CrawlExecutionStatus current = previous.toBuilder().setState(State.FINISHED)
                .setEndTime(Timestamp.newBuilder().setSeconds(1)).build();
        when(adapter.updateCrawlExecutionStatus(any())).thenReturn(new CrawlExecutionStatusUpdate(previous, current));
        when(queue.updateJobExecutionStatus(eq("job"), any(), any(), any())).thenReturn(false);

        StatusWrapper.getStatusWrapper(frontier, previous).setEndState(State.FINISHED).saveStatus();
    }

    private JobExecutionStatus finishedAggregate() {
        return JobExecutionStatus.newBuilder().setId("job").setDocumentsCrawled(37)
                .putExecutionsState(State.FINISHED.name(), 2).build();
    }

    private JobExecutionStatus runningJob() {
        return JobExecutionStatus.newBuilder().setId("job").setState(JobExecutionStatus.State.RUNNING).build();
    }
}
