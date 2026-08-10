package no.nb.nna.veidemann.frontier.api;

import io.grpc.*;

import java.util.concurrent.Semaphore;
import java.util.concurrent.atomic.AtomicBoolean;

public class SimpleConcurrencyLimitInterceptor implements ServerInterceptor {

    private final Semaphore permits;

    public SimpleConcurrencyLimitInterceptor(int maxConcurrentRequests) {
        if (maxConcurrentRequests <= 0) {
            throw new IllegalArgumentException("maxConcurrentRequests must be greater than zero");
        }
        this.permits = new Semaphore(maxConcurrentRequests);
    }

    @Override
    public <ReqT, RespT> ServerCall.Listener<ReqT> interceptCall(
            ServerCall<ReqT, RespT> call,
            Metadata headers,
            ServerCallHandler<ReqT, RespT> next) {

        if (!permits.tryAcquire()) {
            // Too many concurrent calls – fail fast
            call.close(
                Status.RESOURCE_EXHAUSTED.withDescription("concurrency limit reached"),
                new Metadata()
            );
            return new ServerCall.Listener<ReqT>() { }; // no-op listener
        }

        AtomicBoolean released = new AtomicBoolean();
        Runnable releasePermit = () -> {
            if (released.compareAndSet(false, true)) {
                permits.release();
            }
        };

        ServerCall.Listener<ReqT> delegate;
        try {
            delegate = next.startCall(
                new ForwardingServerCall.SimpleForwardingServerCall<ReqT, RespT>(call) {
                    @Override
                    public void close(Status status, Metadata trailers) {
                        try {
                            super.close(status, trailers);
                        } finally {
                            releasePermit.run();
                        }
                    }
                },
                headers);
        } catch (RuntimeException | Error e) {
            releasePermit.run();
            throw e;
        }

        return new ForwardingServerCallListener.SimpleForwardingServerCallListener<ReqT>(delegate) {
            @Override
            public void onCancel() {
                try {
                    super.onCancel();
                } finally {
                    releasePermit.run();
                }
            }
        };
    }
}
