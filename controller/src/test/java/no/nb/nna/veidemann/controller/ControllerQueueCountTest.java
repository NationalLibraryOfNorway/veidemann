package no.nb.nna.veidemann.controller;

import com.google.common.util.concurrent.FutureCallback;
import io.grpc.Metadata;
import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.grpc.stub.StreamObserver;
import no.nb.nna.veidemann.api.frontier.v1.CountResponse;
import no.nb.nna.veidemann.api.frontier.v1.JobExecutionId;
import no.nb.nna.veidemann.commons.db.ConfigAdapter;
import no.nb.nna.veidemann.commons.db.ExecutionsAdapter;
import no.nb.nna.veidemann.controller.settings.Settings;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;

import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.doAnswer;
import static org.mockito.Mockito.mock;
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
    void forwardsJobExecutionCountResponses() {
        doAnswer(invocation -> {
            FutureCallback<CountResponse> callback = invocation.getArgument(1);
            callback.onSuccess(CountResponse.newBuilder().setCount(42).build());
            return null;
        }).when(frontierClient).queueCountForJobExecution(any(), any(), any());
        @SuppressWarnings("unchecked")
        StreamObserver<CountResponse> observer = mock(StreamObserver.class);

        controllerService.queueCountForJobExecution(
                JobExecutionId.newBuilder().setId("job-execution").build(), observer);

        ArgumentCaptor<CountResponse> response = ArgumentCaptor.forClass(CountResponse.class);
        verify(observer).onNext(response.capture());
        assertThat(response.getValue().getCount()).isEqualTo(42L);
        verify(observer).onCompleted();
    }

    @Test
    void forwardsAnEmptyJobExecutionId() {
        doAnswer(invocation -> {
            JobExecutionId request = invocation.getArgument(0);
            assertThat(request.getId()).isEmpty();
            FutureCallback<CountResponse> callback = invocation.getArgument(1);
            callback.onSuccess(CountResponse.getDefaultInstance());
            return null;
        }).when(frontierClient).queueCountForJobExecution(any(), any(), any());
        @SuppressWarnings("unchecked")
        StreamObserver<CountResponse> observer = mock(StreamObserver.class);

        controllerService.queueCountForJobExecution(JobExecutionId.getDefaultInstance(), observer);

        verify(observer).onNext(CountResponse.getDefaultInstance());
        verify(observer).onCompleted();
    }

    @Test
    void preservesResourceExhaustedStatus() {
        Metadata.Key<String> retryHint = Metadata.Key.of("retry-hint", Metadata.ASCII_STRING_MARSHALLER);
        Metadata trailers = new Metadata();
        trailers.put(retryHint, "later");
        doAnswer(invocation -> {
            FutureCallback<CountResponse> callback = invocation.getArgument(1);
            callback.onFailure(Status.RESOURCE_EXHAUSTED
                    .withDescription("concurrency limit reached")
                    .asRuntimeException(trailers));
            return null;
        }).when(frontierClient).queueCountForJobExecution(any(), any(), any());
        @SuppressWarnings("unchecked")
        StreamObserver<CountResponse> observer = mock(StreamObserver.class);

        controllerService.queueCountForJobExecution(JobExecutionId.getDefaultInstance(), observer);

        ArgumentCaptor<Throwable> error = ArgumentCaptor.forClass(Throwable.class);
        verify(observer).onError(error.capture());
        StatusRuntimeException forwarded = (StatusRuntimeException) error.getValue();
        assertThat(forwarded.getStatus().getCode()).isEqualTo(Status.Code.RESOURCE_EXHAUSTED);
        assertThat(forwarded.getStatus().getDescription()).isEqualTo("concurrency limit reached");
        assertThat(forwarded.getTrailers().get(retryHint)).isEqualTo("later");
    }
}
