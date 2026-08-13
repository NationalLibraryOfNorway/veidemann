package no.nb.nna.veidemann.controller;

import com.google.common.util.concurrent.FutureCallback;
import io.grpc.Metadata;
import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.grpc.stub.StreamObserver;
import no.nb.nna.veidemann.api.frontier.v1.ExecutionIds;
import no.nb.nna.veidemann.api.frontier.v1.QueueCountsResponse;
import no.nb.nna.veidemann.commons.db.ConfigAdapter;
import no.nb.nna.veidemann.commons.db.ExecutionsAdapter;
import no.nb.nna.veidemann.controller.settings.Settings;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;

import java.util.List;
import java.util.Objects;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.doAnswer;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;

class ControllerQueueCountTest {
    private FrontierClient frontierClient;
    private ControllerService controllerService;

    @BeforeEach
    void setUp() {
        frontierClient = mock(FrontierClient.class);
        JobExecutionUtil.addFrontierClient("url", frontierClient);
        controllerService = new ControllerService(
                new Settings(), mock(ConfigAdapter.class), mock(ExecutionsAdapter.class), List.of());
    }

    @Test
    void forwardsBatchResponses() {
        QueueCountsResponse expected = QueueCountsResponse.newBuilder()
                .putCounts("job-1", 8L)
                .putCounts("job-2", 0L)
                .build();
        doAnswer(invocation -> {
            FutureCallback<QueueCountsResponse> callback = invocation.getArgument(1);
            callback.onSuccess(expected);
            return null;
        }).when(frontierClient).queueCountsForJobExecutions(any(), any(), any());
        @SuppressWarnings("unchecked")
        StreamObserver<QueueCountsResponse> observer = mock(StreamObserver.class);

        controllerService.queueCountsForJobExecutions(
                ExecutionIds.newBuilder().addId("job-1").addId("job-2").build(), observer);

        verify(observer).onNext(expected);
        verify(observer).onCompleted();
    }

    @Test
    void preservesBatchFailureStatusAndTrailers() {
        Metadata.Key<String> retryHint = Metadata.Key.of("retry-hint", Metadata.ASCII_STRING_MARSHALLER);
        Metadata trailers = new Metadata();
        trailers.put(retryHint, "later");
        doAnswer(invocation -> {
            FutureCallback<QueueCountsResponse> callback = invocation.getArgument(1);
            callback.onFailure(Status.RESOURCE_EXHAUSTED.withDescription("busy").asRuntimeException(trailers));
            return null;
        }).when(frontierClient).queueCountsForCrawlExecutions(any(), any(), any());
        @SuppressWarnings("unchecked")
        StreamObserver<QueueCountsResponse> observer = mock(StreamObserver.class);

        controllerService.queueCountsForCrawlExecutions(
                ExecutionIds.newBuilder().addId("crawl-1").build(), observer);

        ArgumentCaptor<Throwable> error = ArgumentCaptor.forClass(Throwable.class);
        verify(observer).onError(error.capture());
        StatusRuntimeException forwarded = (StatusRuntimeException) error.getValue();
        assertThat(forwarded.getStatus().getCode()).isEqualTo(Status.Code.RESOURCE_EXHAUSTED);
        assertThat(forwarded.getStatus().getDescription()).isEqualTo("busy");
        assertThat(Objects.requireNonNull(forwarded.getTrailers()).get(retryHint)).isEqualTo("later");
    }

    @Test
    void rejectsOversizedBatchBeforeForwarding() {
        ExecutionIds request = ExecutionIds.newBuilder().addAllId(
                java.util.stream.IntStream.range(0, 101).mapToObj(Integer::toString).toList()).build();
        @SuppressWarnings("unchecked")
        StreamObserver<QueueCountsResponse> observer = mock(StreamObserver.class);

        controllerService.queueCountsForJobExecutions(request, observer);

        ArgumentCaptor<Throwable> error = ArgumentCaptor.forClass(Throwable.class);
        verify(observer).onError(error.capture());
        assertThat(((StatusRuntimeException) error.getValue()).getStatus().getCode())
                .isEqualTo(Status.Code.INVALID_ARGUMENT);
        verify(frontierClient, never()).queueCountsForJobExecutions(any(), any(), any());
    }
}
