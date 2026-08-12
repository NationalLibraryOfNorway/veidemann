package no.nb.nna.veidemann.frontier.api;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;

import io.grpc.stub.StreamObserver;
import no.nb.nna.veidemann.api.frontier.v1.CountResponse;
import no.nb.nna.veidemann.api.frontier.v1.JobExecutionId;
import no.nb.nna.veidemann.frontier.db.CrawlQueueManager;
import no.nb.nna.veidemann.frontier.worker.Frontier;

class FrontierGrpcServiceQueueCountTest {

    @Test
    void returnsQueueCountForJobExecution() {
        Frontier frontier = mock(Frontier.class);
        CrawlQueueManager crawlQueueManager = mock(CrawlQueueManager.class);
        when(frontier.getCrawlQueueManager()).thenReturn(crawlQueueManager);
        when(crawlQueueManager.countByJobExecution("job-execution")).thenReturn(42L);
        FrontierGrpcService service = new FrontierGrpcService(frontier);
        @SuppressWarnings("unchecked")
        StreamObserver<CountResponse> observer = mock(StreamObserver.class);

        service.queueCountForJobExecution(
                JobExecutionId.newBuilder().setId("job-execution").build(),
                observer);

        ArgumentCaptor<CountResponse> response = ArgumentCaptor.forClass(CountResponse.class);
        verify(observer).onNext(response.capture());
        assertThat(response.getValue().getCount()).isEqualTo(42L);
        verify(observer).onCompleted();
    }
}
