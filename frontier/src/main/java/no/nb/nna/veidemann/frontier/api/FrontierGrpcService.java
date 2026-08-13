/*
 * Copyright 2017 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package no.nb.nna.veidemann.frontier.api;

import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.concurrent.TimeUnit;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.slf4j.MDC;

import com.google.protobuf.Empty;

import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.grpc.stub.ServerCallStreamObserver;
import io.grpc.stub.StreamObserver;
import no.nb.nna.veidemann.api.frontier.v1.CountResponse;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionId;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.CrawlHostGroup;
import no.nb.nna.veidemann.api.frontier.v1.CrawlSeedRequest;
import no.nb.nna.veidemann.api.frontier.v1.ExecutionIds;
import no.nb.nna.veidemann.api.frontier.v1.FrontierGrpc;
import no.nb.nna.veidemann.api.frontier.v1.PageHarvest;
import no.nb.nna.veidemann.api.frontier.v1.PageHarvestSpec;
import no.nb.nna.veidemann.api.frontier.v1.QueueCountsResponse;
import no.nb.nna.veidemann.frontier.worker.Frontier;

/**
 * gRPC implementation of the Frontier API.
 */
public class FrontierGrpcService extends FrontierGrpc.FrontierImplBase implements AutoCloseable {

    private static final Logger LOG = LoggerFactory.getLogger(FrontierGrpcService.class);
    static final int MAX_QUEUE_COUNT_IDS = 100;

    private final Frontier frontier;
    private final Context ctx;

    public FrontierGrpcService(Frontier frontier) {
        this.frontier = frontier;
        this.ctx = new Context(frontier);
    }

    public Frontier getFrontier() {
        return frontier;
    }

    /** Stop internal executors / context, but do NOT close the Frontier itself. */
    public void shutdown() {
        ctx.shutdown();
    }

    public boolean awaitTermination(long timeout, TimeUnit unit) throws InterruptedException {
        return ctx.awaitTermination(timeout, unit);
    }

    public void awaitTermination() throws InterruptedException {
        ctx.awaitTermination();
    }

    @Override
    public void close() {
        shutdown();
    }

    @Override
    public void crawlSeed(CrawlSeedRequest request, StreamObserver<CrawlExecutionId> responseObserver) {
        MDC.clear();
        MDC.put("uri", request.getSeed().getMeta().getName());
        try {
            CrawlExecutionStatus reply = frontier.scheduleSeed(request);
            responseObserver.onNext(CrawlExecutionId.newBuilder().setId(reply.getId()).build());
            responseObserver.onCompleted();
        } catch (StatusRuntimeException e) {
            // gRPC error from downstream; keep status, log with some detail
            LOG.error("Crawl seed error (gRPC status: {}): {}", e.getStatus(), e.getMessage());
            responseObserver.onError(e);
        } catch (Exception e) {
            LOG.error("Crawl seed error: {}", e.getMessage(), e);
            Status status = Status.UNKNOWN.withDescription(e.toString()).withCause(e);
            responseObserver.onError(status.asException());
        } finally {
            MDC.clear();
        }
    }

    @Override
    public void getNextPage(Empty request, StreamObserver<PageHarvestSpec> responseObserver) {
        GetNextPageHandler.onNext(ctx, responseObserver);
    }

    @Override
    public StreamObserver<PageHarvest> pageCompleted(StreamObserver<Empty> responseObserver) {
        return new PageCompletedHandler(ctx, (ServerCallStreamObserver<Empty>) responseObserver);
    }

    @Override
    public void busyCrawlHostGroupCount(Empty request, StreamObserver<CountResponse> responseObserver) {
        CountResponse response = CountResponse.newBuilder()
                .setCount(ctx.getCrawlQueueManager().busyCrawlHostGroupCount())
                .build();
        responseObserver.onNext(response);
        responseObserver.onCompleted();
    }

    @Override
    public void queueCountTotal(Empty request, StreamObserver<CountResponse> responseObserver) {
        CountResponse response = CountResponse.newBuilder()
                .setCount(ctx.getCrawlQueueManager().queueCountTotal())
                .build();
        responseObserver.onNext(response);
        responseObserver.onCompleted();
    }

    @Override
    public void queueCountsForCrawlExecutions(
            ExecutionIds request, StreamObserver<QueueCountsResponse> responseObserver) {
        List<String> ids = validateExecutionIds(request, responseObserver);
        if (ids == null) {
            return;
        }
        responseObserver.onNext(QueueCountsResponse.newBuilder()
                .putAllCounts(ctx.getCrawlQueueManager().countByCrawlExecutions(ids))
                .build());
        responseObserver.onCompleted();
    }

    @Override
    public void queueCountsForJobExecutions(
            ExecutionIds request, StreamObserver<QueueCountsResponse> responseObserver) {
        List<String> ids = validateExecutionIds(request, responseObserver);
        if (ids == null) {
            return;
        }
        responseObserver.onNext(QueueCountsResponse.newBuilder()
                .putAllCounts(ctx.getCrawlQueueManager().countByJobExecutions(ids))
                .build());
        responseObserver.onCompleted();
    }

    @Override
    public void queueCountForCrawlHostGroup(CrawlHostGroup request, StreamObserver<CountResponse> responseObserver) {
        CountResponse response = CountResponse.newBuilder()
                .setCount(ctx.getCrawlQueueManager().countByCrawlHostGroup(request))
                .build();
        responseObserver.onNext(response);
        responseObserver.onCompleted();
    }

    private static List<String> validateExecutionIds(ExecutionIds request, StreamObserver<?> responseObserver) {
        if (request.getIdCount() > MAX_QUEUE_COUNT_IDS) {
            responseObserver.onError(Status.INVALID_ARGUMENT
                    .withDescription("Execution id list may contain at most " + MAX_QUEUE_COUNT_IDS + " entries")
                    .asRuntimeException());
            return null;
        }
        if (request.getIdList().stream().anyMatch(String::isBlank)) {
            responseObserver.onError(Status.INVALID_ARGUMENT
                    .withDescription("Execution ids must not be blank")
                    .asRuntimeException());
            return null;
        }
        return new ArrayList<>(new LinkedHashSet<>(request.getIdList()));
    }
}
