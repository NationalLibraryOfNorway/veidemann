package no.nb.nna.veidemann.frontier.api;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import java.util.List;
import java.util.Map;

import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;

import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.grpc.stub.StreamObserver;
import no.nb.nna.veidemann.api.frontier.v1.ExecutionIds;
import no.nb.nna.veidemann.api.frontier.v1.QueueCountsResponse;
import no.nb.nna.veidemann.frontier.db.CrawlQueueManager;
import no.nb.nna.veidemann.frontier.worker.Frontier;

class FrontierGrpcServiceQueueCountTest {

    @Test
    void returnsDeduplicatedBatchCountsForJobExecutions() {
        Frontier frontier = mock(Frontier.class);
        CrawlQueueManager crawlQueueManager = mock(CrawlQueueManager.class);
        when(frontier.getCrawlQueueManager()).thenReturn(crawlQueueManager);
        when(crawlQueueManager.countByJobExecutions(List.of("job-1", "job-2")))
                .thenReturn(Map.of("job-1", 4L, "job-2", 0L));
        FrontierGrpcService service = new FrontierGrpcService(frontier);
        @SuppressWarnings("unchecked")
        StreamObserver<QueueCountsResponse> observer = mock(StreamObserver.class);

        service.queueCountsForJobExecutions(
                ExecutionIds.newBuilder().addId("job-1").addId("job-1").addId("job-2").build(),
                observer);

        ArgumentCaptor<QueueCountsResponse> response = ArgumentCaptor.forClass(QueueCountsResponse.class);
        verify(observer).onNext(response.capture());
        assertThat(response.getValue().getCountsMap()).containsExactlyInAnyOrderEntriesOf(
                Map.of("job-1", 4L, "job-2", 0L));
        verify(observer).onCompleted();
    }

    @Test
    void rejectsBlankAndOversizedBatchRequests() {
        Frontier frontier = mock(Frontier.class);
        CrawlQueueManager crawlQueueManager = mock(CrawlQueueManager.class);
        when(frontier.getCrawlQueueManager()).thenReturn(crawlQueueManager);
        FrontierGrpcService service = new FrontierGrpcService(frontier);

        for (ExecutionIds request : List.of(
                ExecutionIds.newBuilder().addId(" ").build(),
                ExecutionIds.newBuilder().addAllId(
                        java.util.stream.IntStream.range(0, 101).mapToObj(Integer::toString).toList()).build())) {
            @SuppressWarnings("unchecked")
            StreamObserver<QueueCountsResponse> observer = mock(StreamObserver.class);
            service.queueCountsForCrawlExecutions(request, observer);

            ArgumentCaptor<Throwable> error = ArgumentCaptor.forClass(Throwable.class);
            verify(observer).onError(error.capture());
            assertThat(((StatusRuntimeException) error.getValue()).getStatus().getCode())
                    .isEqualTo(Status.Code.INVALID_ARGUMENT);
        }
        verify(crawlQueueManager, never()).countByCrawlExecutions(org.mockito.ArgumentMatchers.anyList());
    }
}
