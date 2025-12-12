package no.nb.nna.veidemann.frontier.api;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.google.protobuf.Empty;

import io.grpc.Status;
import io.grpc.Status.Code;
import io.grpc.StatusRuntimeException;
import io.grpc.stub.StreamObserver;
import io.opentracing.Scope;
import io.opentracing.Span;
import io.opentracing.noop.NoopSpan;
import io.opentracing.tag.Tags;
import no.nb.nna.veidemann.api.commons.v1.Error;
import no.nb.nna.veidemann.api.frontier.v1.PageHarvest;
import no.nb.nna.veidemann.commons.ExtraStatusCodes;
import no.nb.nna.veidemann.frontier.worker.IllegalSessionException;
import no.nb.nna.veidemann.frontier.worker.PostFetchHandler;

public class PageCompletedHandler implements StreamObserver<PageHarvest> {
    private static final Logger LOG = LoggerFactory.getLogger(PageCompletedHandler.class);

    final Context ctx;
    final StreamObserver<Empty> responseObserver;
    private PostFetchHandler postFetchHandler;
    private final Span span;

    public PageCompletedHandler(Context ctx, StreamObserver<Empty> responseObserver) {
        this.responseObserver = responseObserver;
        this.ctx = ctx;

        Span active = ctx.getFrontier().getTracer().scopeManager().activeSpan();
        this.span = active != null ? active : NoopSpan.INSTANCE;

        ctx.startPageComplete();
    }

    private void sendError() {
        try {
            responseObserver.onError(Status.ABORTED.asException());
        } catch (Exception e) {
            // OK if this fails; we're already in an error path.
            LOG.debug("Failed sending error to client", e);
        } finally {
            ctx.setObserverCompleted();
            postFetchHandler = null;
        }
    }

    @Override
    public void onNext(PageHarvest value) {
        if (postFetchHandler == null) {
            try {
                postFetchHandler = new PostFetchHandler(
                        value.getSessionToken(),
                        ctx.getFrontier());
            } catch (IllegalSessionException e) {
                LOG.warn("Illegal session: {}", e.toString());
                sendError();
                return;
            } catch (Exception e) {
                LOG.warn("Failed to load PostFetchHandler: {}", e.toString(), e);
                sendError();
                return;
            }
        }

        switch (value.getMsgCase()) {
            case METRICS:
                try {
                    postFetchHandler.postFetchSuccess(value.getMetrics());
                } catch (Exception e) {
                    LOG.warn("Failed to execute postFetchSuccess: {}", e.toString(), e);
                    sendError();
                }
                break;

            case OUTLINK:
                try {
                    postFetchHandler.queueOutlink(value.getOutlink());
                } catch (Exception e) {
                    LOG.warn("Could not queue outlink '{}'", value.getOutlink().getUri(), e);
                    sendError();
                }
                break;

            case ERROR:
                try {
                    postFetchHandler.postFetchFailure(value.getError());
                } catch (Exception e) {
                    LOG.warn("Failed to execute postFetchFailure: {}", e.toString(), e);
                    sendError();
                }
                break;

            case MSG_NOT_SET:
            default:
                LOG.warn("Received PageHarvest with no message set");
                // You could treat this as a protocol error and sendError() if you want.
                break;
        }
    }

    @Override
    public void onError(Throwable t) {
        if (postFetchHandler == null
                || (t instanceof StatusRuntimeException
                        && ((StatusRuntimeException) t).getStatus().getCode() == Code.CANCELLED)) {
            ctx.setObserverCompleted();
            postFetchHandler = null;
            return;
        }

        LOG.warn("gRPC Error from harvester", t);
        try (Scope scope = ctx.getFrontier().getTracer().scopeManager().activate(span)) {
            try {
                Error error = ExtraStatusCodes.RUNTIME_EXCEPTION
                        .toFetchError("Browser controller failed: " + t.toString());
                postFetchHandler.postFetchFailure(error);
                postFetchHandler.postFetchFinally(true);
            } catch (Exception e) {
                LOG.error("Failed to execute postFetchFinally after error: {}", e.toString(), e);
            }
        } finally {
            ctx.setObserverCompleted();
            postFetchHandler = null;
        }
    }

    @Override
    public void onCompleted() {
        if (postFetchHandler == null) {
            ctx.setObserverCompleted();
            return;
        }

        Span completeSpan = ctx.getFrontier().getTracer().buildSpan("completeFetch")
                .withTag(Tags.COMPONENT, "Frontier")
                .withTag(Tags.SPAN_KIND, Tags.SPAN_KIND_SERVER)
                .start();

        try (Scope scope = ctx.getFrontier().getTracer().scopeManager().activate(completeSpan)) {
            try {
                postFetchHandler.postFetchFinally(false);
                LOG.trace("Done with uri {}", postFetchHandler.getUri().getUri());
                postFetchHandler = null;
            } catch (Exception e) {
                LOG.error("Failed to execute postFetchFinally: {}", e.toString(), e);
                sendError();
                return;
            }

            try {
                responseObserver.onNext(Empty.getDefaultInstance());
                responseObserver.onCompleted();
            } catch (Exception e) {
                LOG.error("Failed to execute onCompleted: {}", e.toString(), e);
            }
        } finally {
            ctx.setObserverCompleted();
            completeSpan.finish();
        }
    }
}
