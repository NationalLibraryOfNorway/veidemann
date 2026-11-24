// Copyright 2026 National Library of Norway.
// SPDX-License-Identifier: Apache-2.0

package no.nb.nna.veidemann.frontier.worker;

import static org.assertj.core.api.Assertions.assertThat;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.MoreExecutors;
import com.google.common.util.concurrent.SettableFuture;
import io.grpc.Server;
import io.grpc.ServerBuilder;
import io.grpc.Status;
import io.grpc.stub.StreamObserver;
import java.time.Duration;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.ForkJoinPool;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import no.nb.nna.veidemann.api.config.v1.ConfigObject;
import no.nb.nna.veidemann.api.config.v1.ConfigRef;
import no.nb.nna.veidemann.api.frontier.v1.QueuedUri;
import no.nb.nna.veidemann.api.robotsevaluator.v1.IsAllowedReply;
import no.nb.nna.veidemann.api.robotsevaluator.v1.IsAllowedRequest;
import no.nb.nna.veidemann.api.robotsevaluator.v1.RobotsEvaluatorGrpc;
import org.junit.jupiter.api.Test;

class RobotsServiceClientTest {
    @Test
    void boundsEachCheckAndReleasesPendingDecision() throws Exception {
        AtomicBoolean stalled = new AtomicBoolean();
        Server server = ServerBuilder.forPort(0).addService(new RobotsEvaluatorGrpc.RobotsEvaluatorImplBase() {
            @Override
            public void isAllowed(IsAllowedRequest req, StreamObserver<IsAllowedReply> reply) {
                switch (req.getUri()) {
                    case "deny":
                        reply.onNext(IsAllowedReply.newBuilder().setIsAllowed(false).build());
                        reply.onCompleted();
                        break;
                    case "unavailable":
                        reply.onError(Status.UNAVAILABLE.asRuntimeException());
                        break;
                    case "stall":
                        // Leave this call pending until its client deadline.
                        stalled.set(true);
                        break;
                    default:
                        reply.onError(Status.INVALID_ARGUMENT.asRuntimeException());
                }
            }
        }).build().start();

        Frontier frontier = mock(Frontier.class);
        when(frontier.getAsyncFunctionsThreadPool()).thenReturn(ForkJoinPool.commonPool());
        try (RobotsServiceClient client = new RobotsServiceClient("localhost", server.getPort(), Duration.ofMillis(500))) {
            for (String uri : new String[]{"deny", "stall", "deny", "unavailable"}) {
                var pending = SettableFuture.<Preconditions.PreconditionOutcome>create();
                var callback = new Preconditions.CheckRobotsCallback(new Preconditions.IsAllowedFunc(
                        null, frontier, mock(QueuedUriWrapper.class), mock(StatusWrapper.class), pending));
                var check = client.isAllowed(frontier, QueuedUri.newBuilder().setUri(uri).build(), "crawler",
                        ConfigObject.getDefaultInstance(), ConfigRef.getDefaultInstance());
                Futures.addCallback(check, callback, MoreExecutors.directExecutor());
                if (uri.equals("deny")) {
                    assertThat(pending.get(3, TimeUnit.SECONDS)).isEqualTo(Preconditions.PreconditionOutcome.COMPLETE);
                } else {
                    assertThat(pending.get(3, TimeUnit.SECONDS)).isEqualTo(Preconditions.PreconditionOutcome.FETCH);
                    ExecutionException error = assertThrows(ExecutionException.class, check::get);
                    assertThat(Status.fromThrowable(error.getCause()).getCode()).isEqualTo(
                            uri.equals("stall") ? Status.Code.DEADLINE_EXCEEDED : Status.Code.UNAVAILABLE);
                }
            }
            assertThat(stalled.get()).isTrue();
        } finally {
            server.shutdownNow().awaitTermination(5, TimeUnit.SECONDS);
        }
    }
}
