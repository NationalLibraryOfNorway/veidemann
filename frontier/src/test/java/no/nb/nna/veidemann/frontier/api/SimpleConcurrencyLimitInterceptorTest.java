package no.nb.nna.veidemann.frontier.api;

import io.grpc.Metadata;
import io.grpc.ServerCall;
import io.grpc.ServerCallHandler;
import io.grpc.Status;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;

import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.argThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class SimpleConcurrencyLimitInterceptorTest {

    @Test
    void rejectsAtTheLimitAndReleasesAtMostOnce() {
        SimpleConcurrencyLimitInterceptor interceptor = new SimpleConcurrencyLimitInterceptor(1);
        @SuppressWarnings("unchecked")
        ServerCallHandler<String, String> handler = mock(ServerCallHandler.class);
        @SuppressWarnings("unchecked")
        ServerCall.Listener<String> delegate = mock(ServerCall.Listener.class);
        when(handler.startCall(any(), any())).thenReturn(delegate);

        ServerCall<String, String> first = mockCall();
        ServerCall.Listener<String> firstListener = interceptor.interceptCall(first, new Metadata(), handler);
        ArgumentCaptor<ServerCall<String, String>> wrappedCall = serverCallCaptor();
        verify(handler).startCall(wrappedCall.capture(), any());

        ServerCall<String, String> rejected = mockCall();
        interceptor.interceptCall(rejected, new Metadata(), handler);
        verify(rejected).close(
                argThat(status -> status.getCode() == Status.Code.RESOURCE_EXHAUSTED),
                any(Metadata.class));

        wrappedCall.getValue().close(Status.OK, new Metadata());
        firstListener.onCancel();

        interceptor.interceptCall(mockCall(), new Metadata(), handler);
        ServerCall<String, String> stillRejected = mockCall();
        interceptor.interceptCall(stillRejected, new Metadata(), handler);

        verify(handler, times(2)).startCall(any(), any());
        verify(stillRejected).close(
                argThat(status -> status.getCode() == Status.Code.RESOURCE_EXHAUSTED),
                any(Metadata.class));
    }

    @Test
    void requiresAPositiveLimit() {
        assertThatThrownBy(() -> new SimpleConcurrencyLimitInterceptor(0))
                .isInstanceOf(IllegalArgumentException.class)
                .hasMessageContaining("greater than zero");
    }

    @Test
    void releasesThePermitWhenStartingTheCallFails() {
        SimpleConcurrencyLimitInterceptor interceptor = new SimpleConcurrencyLimitInterceptor(1);
        @SuppressWarnings("unchecked")
        ServerCallHandler<String, String> failingHandler = mock(ServerCallHandler.class);
        when(failingHandler.startCall(any(), any())).thenThrow(new IllegalStateException("failed to start"));

        assertThatThrownBy(() -> interceptor.interceptCall(mockCall(), new Metadata(), failingHandler))
                .isInstanceOf(IllegalStateException.class);

        @SuppressWarnings("unchecked")
        ServerCallHandler<String, String> workingHandler = mock(ServerCallHandler.class);
        @SuppressWarnings("unchecked")
        ServerCall.Listener<String> workingDelegate = mock(ServerCall.Listener.class);
        when(workingHandler.startCall(any(), any())).thenReturn(workingDelegate);
        interceptor.interceptCall(mockCall(), new Metadata(), workingHandler);
        verify(workingHandler).startCall(any(), any());
    }

    @SuppressWarnings("unchecked")
    private static ServerCall<String, String> mockCall() {
        return mock(ServerCall.class);
    }

    @SuppressWarnings({"unchecked", "rawtypes"})
    private static ArgumentCaptor<ServerCall<String, String>> serverCallCaptor() {
        return (ArgumentCaptor) ArgumentCaptor.forClass(ServerCall.class);
    }
}
