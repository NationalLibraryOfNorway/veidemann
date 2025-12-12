package no.nb.nna.veidemann.frontier.api;

import io.grpc.*;

import java.util.concurrent.Semaphore;

public class SimpleConcurrencyLimitInterceptor implements ServerInterceptor {

    private final Semaphore permits;

    public SimpleConcurrencyLimitInterceptor(int maxConcurrentRequests) {
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

        ServerCall.Listener<ReqT> delegate = next.startCall(
            new ForwardingServerCall.SimpleForwardingServerCall<ReqT, RespT>(call) {
                @Override
                public void close(Status status, Metadata trailers) {
                    try {
                        super.close(status, trailers);
                    } finally {
                        permits.release();
                    }
                }
            },
            headers);

        return new ForwardingServerCallListener.SimpleForwardingServerCallListener<ReqT>(delegate) {
            @Override
            public void onCancel() {
                try {
                    super.onCancel();
                } finally {
                    permits.release();
                }
            }
        };
    }
}
