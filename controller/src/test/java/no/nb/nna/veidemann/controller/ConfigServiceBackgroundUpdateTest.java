package no.nb.nna.veidemann.controller;

import io.grpc.Context;
import io.grpc.Status;
import io.grpc.stub.StreamObserver;
import no.nb.nna.veidemann.api.config.v1.Kind;
import no.nb.nna.veidemann.api.config.v1.UpdateRequest;
import no.nb.nna.veidemann.api.config.v1.UpdateResponse;
import no.nb.nna.veidemann.api.config.v1.UpdateTaskAccepted;
import no.nb.nna.veidemann.commons.auth.EmailContextKey;
import no.nb.nna.veidemann.commons.db.ConfigAdapter;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;

import java.util.concurrent.CountDownLatch;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class ConfigServiceBackgroundUpdateTest {
    private ConfigAdapter db;
    private ConfigService service;

    @BeforeEach
    void setUp() {
        db = mock(ConfigAdapter.class);
        service = new ConfigService(db, mock(ScopeServiceClient.class),
                Executors.newSingleThreadExecutor());
    }

    @AfterEach
    void tearDown() {
        service.close();
    }

    @Test
    void acceptsBeforeUpdateCompletesAndPreservesSubmittingUser() throws Exception {
        CountDownLatch started = new CountDownLatch(1);
        CountDownLatch release = new CountDownLatch(1);
        CountDownLatch completed = new CountDownLatch(1);
        AtomicReference<String> backgroundUser = new AtomicReference<>();
        when(db.updateConfigObjects(any())).thenAnswer(invocation -> {
            backgroundUser.set(EmailContextKey.email());
            started.countDown();
            assertThat(release.await(5, TimeUnit.SECONDS)).isTrue();
            completed.countDown();
            return UpdateResponse.newBuilder().setUpdated(500_000).build();
        });
        @SuppressWarnings("unchecked")
        StreamObserver<UpdateTaskAccepted> observer = mock(StreamObserver.class);

        Context.current().withValue(EmailContextKey.getKey(), "curator@example.org").run(() ->
                service.startUpdateConfigObjects(updateRequest(), observer));

        ArgumentCaptor<UpdateTaskAccepted> accepted = ArgumentCaptor.forClass(UpdateTaskAccepted.class);
        verify(observer).onNext(accepted.capture());
        verify(observer).onCompleted();
        verify(observer, never()).onError(any());
        assertThat(accepted.getValue().getTaskId()).isNotBlank();
        assertThat(started.await(5, TimeUnit.SECONDS)).isTrue();
        assertThat(backgroundUser).hasValue("curator@example.org");

        release.countDown();
        assertThat(completed.await(5, TimeUnit.SECONDS)).isTrue();
    }

    @Test
    void rejectsAConcurrentBackgroundUpdate() throws Exception {
        CountDownLatch started = new CountDownLatch(1);
        CountDownLatch release = new CountDownLatch(1);
        when(db.updateConfigObjects(any())).thenAnswer(invocation -> {
            started.countDown();
            assertThat(release.await(5, TimeUnit.SECONDS)).isTrue();
            return UpdateResponse.getDefaultInstance();
        });
        @SuppressWarnings("unchecked")
        StreamObserver<UpdateTaskAccepted> firstObserver = mock(StreamObserver.class);
        @SuppressWarnings("unchecked")
        StreamObserver<UpdateTaskAccepted> secondObserver = mock(StreamObserver.class);

        service.startUpdateConfigObjects(updateRequest(), firstObserver);
        assertThat(started.await(5, TimeUnit.SECONDS)).isTrue();
        service.startUpdateConfigObjects(updateRequest(), secondObserver);

        ArgumentCaptor<Throwable> error = ArgumentCaptor.forClass(Throwable.class);
        verify(secondObserver).onError(error.capture());
        assertThat(Status.fromThrowable(error.getValue()).getCode())
                .isEqualTo(Status.Code.RESOURCE_EXHAUSTED);
        verify(secondObserver, never()).onNext(any());
        verify(secondObserver, never()).onCompleted();

        release.countDown();
    }

    @Test
    void backgroundFailureDoesNotRespondAgain() throws Exception {
        CountDownLatch attempted = new CountDownLatch(1);
        when(db.updateConfigObjects(any())).thenAnswer(invocation -> {
            attempted.countDown();
            throw new IllegalStateException("database failed");
        });
        @SuppressWarnings("unchecked")
        StreamObserver<UpdateTaskAccepted> observer = mock(StreamObserver.class);

        service.startUpdateConfigObjects(updateRequest(), observer);

        verify(observer).onNext(any());
        verify(observer).onCompleted();
        assertThat(attempted.await(5, TimeUnit.SECONDS)).isTrue();
        verify(observer, never()).onError(any());
    }

    private UpdateRequest updateRequest() {
        return UpdateRequest.newBuilder()
                .setListRequest(no.nb.nna.veidemann.api.config.v1.ListRequest.newBuilder()
                        .setKind(Kind.seed))
                .build();
    }
}
