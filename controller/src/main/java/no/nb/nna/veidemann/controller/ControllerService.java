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
package no.nb.nna.veidemann.controller;

import com.google.common.util.concurrent.FutureCallback;
import com.google.protobuf.Empty;
import io.grpc.Metadata;
import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.grpc.stub.StreamObserver;
import no.nb.nna.veidemann.api.config.v1.ConfigObject;
import no.nb.nna.veidemann.api.config.v1.ConfigRef;
import no.nb.nna.veidemann.api.config.v1.Kind;
import no.nb.nna.veidemann.api.config.v1.Role;
import no.nb.nna.veidemann.api.controller.v1.ControllerGrpc;
import no.nb.nna.veidemann.api.controller.v1.CrawlerStatus;
import no.nb.nna.veidemann.api.controller.v1.CrawlerStatus.Builder;
import no.nb.nna.veidemann.api.controller.v1.ExecutionId;
import no.nb.nna.veidemann.api.controller.v1.OpenIdConnectIssuerReply;
import no.nb.nna.veidemann.api.controller.v1.RoleList;
import no.nb.nna.veidemann.api.controller.v1.RunCrawlReply;
import no.nb.nna.veidemann.api.controller.v1.RunCrawlRequest;
import no.nb.nna.veidemann.api.controller.v1.RunStatus;
import no.nb.nna.veidemann.api.frontier.v1.CountResponse;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.CrawlHostGroup;
import no.nb.nna.veidemann.api.frontier.v1.ExecutionIds;
import no.nb.nna.veidemann.api.frontier.v1.JobExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.QueueCountsResponse;
import no.nb.nna.veidemann.commons.auth.AllowedRoles;
import no.nb.nna.veidemann.commons.auth.RolesContextKey;
import no.nb.nna.veidemann.commons.db.ConfigAdapter;
import no.nb.nna.veidemann.commons.db.ExecutionsAdapter;
import no.nb.nna.veidemann.controller.ControllerApiServer.JobExecutionListener;
import no.nb.nna.veidemann.controller.settings.Settings;
import org.checkerframework.checker.nullness.qual.Nullable;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.OffsetDateTime;
import java.util.Collection;
import java.util.List;

import static no.nb.nna.veidemann.controller.JobExecutionUtil.calculateTimeout;
import static no.nb.nna.veidemann.controller.JobExecutionUtil.crawlSeed;

/**
 *
 */
public class ControllerService extends ControllerGrpc.ControllerImplBase {

    private static final Logger LOG = LoggerFactory.getLogger(ControllerService.class);
    static final int MAX_QUEUE_COUNT_IDS = 100;

    private final ConfigAdapter db;
    private final ExecutionsAdapter executionsAdapter;

    private final Settings settings;
    private final List<JobExecutionListener> jobExecutionListeners;

    public ControllerService(Settings settings, ConfigAdapter db, ExecutionsAdapter executionsAdapter,
            List<JobExecutionListener> jobExecutionListeners) {
        this.settings = settings;
        this.jobExecutionListeners = jobExecutionListeners;
        this.db = db;
        this.executionsAdapter = executionsAdapter;
    }

    @Override
    @AllowedRoles({Role.CURATOR, Role.ADMIN, Role.OPERATOR})
    public void runCrawl(RunCrawlRequest request, StreamObserver<RunCrawlReply> responseObserver) {
        try {
            ConfigRef jobRequest = ConfigRef.newBuilder()
                    .setKind(Kind.crawlJob)
                    .setId(request.getJobId())
                    .build();

            ConfigObject job = db.getConfigObject(jobRequest);
            if (job.getCrawlJob().getDisabled()) {
                responseObserver.onError(Status.FAILED_PRECONDITION
                        .withDescription("Crawl job '" + job.getMeta().getName() + "' is disabled")
                        .asRuntimeException());
                return;
            }
            LOG.info("Job '{}' starting", job.getMeta().getName());

            JobExecutionStatus jobExecutionStatus = null;
            boolean addToRunningJob = false;

            jobExecutionStatus = JobExecutionUtil.getRunningJobExecutionStatusForJob(job);
            if (jobExecutionStatus != null) {
                addToRunningJob = true;
                LOG.info("Adding seeds to running job execution '{}'", jobExecutionStatus.getId());
            }

            OffsetDateTime timeout = calculateTimeout(job);

            if (!request.getSeedId().isEmpty()) {
                // Start only the requested seed for the job
                ConfigObject seed = db.getConfigObject(ConfigRef.newBuilder()
                        .setKind(Kind.seed)
                        .setId(request.getSeedId())
                        .build());
                if (seed.getSeed().getDisabled()) {
                    responseObserver.onError(Status.FAILED_PRECONDITION
                            .withDescription("Seed '" + seed.getMeta().getName() + "' is disabled")
                            .asRuntimeException());
                    return;
                }
                if (!JobExecutionUtil.hasFrontierClient(seed)) {
                    responseObserver.onError(Status.UNAVAILABLE
                            .withDescription("No Frontier client is configured for seed '"
                                    + seed.getMeta().getName() + "'")
                            .asRuntimeException());
                    return;
                }
                if (addToRunningJob && JobExecutionUtil.isSeedInJobExecution(seed, jobExecutionStatus)) {
                    sendRunCrawlReply(jobExecutionStatus, responseObserver);
                    return;
                }

                JobExecutionUtil.GetScriptAnnotationsForJob(job);
                boolean createdJobExecution = jobExecutionStatus == null;
                jobExecutionStatus = JobExecutionUtil.createJobExecutionStatusIfNotExist(job, jobExecutionStatus);
                for (JobExecutionListener listener : jobExecutionListeners) {
                    listener.onJobStarting(jobExecutionStatus.getId());
                }
                if (!crawlSeed(null, job, seed, jobExecutionStatus, timeout, false)) {
                    if (createdJobExecution) {
                        JobExecutionUtil.setJobExecutionStateFailedIfEmpty(
                                jobExecutionStatus, 1L, "Frontier rejected the seed submission");
                    }
                    responseObserver.onError(Status.UNAVAILABLE
                            .withDescription("Frontier did not create a crawl execution for seed '"
                                    + seed.getMeta().getName() + "'")
                            .asRuntimeException());
                    return;
                }
                for (JobExecutionListener listener : jobExecutionListeners) {
                    listener.onJobStarted(jobExecutionStatus.getId());
                }
            } else {
                // Start all seeds for the job
                jobExecutionStatus = JobExecutionUtil.submitSeeds(job, jobExecutionStatus, timeout, addToRunningJob, jobExecutionListeners);
                if (jobExecutionStatus == null) {
                    Status status = Status.FAILED_PRECONDITION.withDescription(
                            "No enabled seeds are associated with crawl job '" + job.getMeta().getName() + "'");
                    responseObserver.onError(status.asException());
                    return;
                }
            }

            sendRunCrawlReply(jobExecutionStatus, responseObserver);
        } catch (StatusRuntimeException ex) {
            LOG.warn("Could not start crawl: {}", ex.getStatus());
            responseObserver.onError(ex);
        } catch (Exception ex) {
            LOG.error(ex.getMessage(), ex);
            Status status = Status.UNKNOWN.withDescription(ex.toString());
            responseObserver.onError(status.asException());
        }
    }

    private static void sendRunCrawlReply(
            JobExecutionStatus jobExecutionStatus, StreamObserver<RunCrawlReply> responseObserver) {
        RunCrawlReply reply = RunCrawlReply.newBuilder()
                .setJobExecutionId(jobExecutionStatus.getId())
                .build();
        responseObserver.onNext(reply);
        responseObserver.onCompleted();
    }

    @Override
    @AllowedRoles({Role.OPERATOR, Role.ADMIN, Role.CURATOR})
    public void abortCrawlExecution(ExecutionId request, StreamObserver<CrawlExecutionStatus> responseObserver) {
        try {
            CrawlExecutionStatus status = executionsAdapter.setCrawlExecutionStateAborted(
                    request.getId(), CrawlExecutionStatus.State.ABORTED_MANUAL);

            responseObserver.onNext(status);
            responseObserver.onCompleted();
        } catch (Exception e) {
            LOG.error(e.getMessage(), e);
            Status status = Status.UNKNOWN.withDescription(e.toString());
            responseObserver.onError(status.asException());
        }
    }

    @Override
    @AllowedRoles({Role.OPERATOR, Role.ADMIN, Role.CURATOR})
    public void abortJobExecution(ExecutionId request, StreamObserver<JobExecutionStatus> responseObserver) {
        try {
            JobExecutionStatus status = executionsAdapter.setJobExecutionStateAborted(request.getId());

            responseObserver.onNext(status);
            responseObserver.onCompleted();
        } catch (Exception e) {
            LOG.error(e.getMessage(), e);
            Status status = Status.UNKNOWN.withDescription(e.toString());
            responseObserver.onError(status.asException());
        }
    }

    @Override
    public void getRolesForActiveUser(Empty request, StreamObserver<RoleList> responseObserver) {
        try {
            Collection<Role> roles = RolesContextKey.roles();
            if (roles == null) {
                responseObserver.onNext(RoleList.newBuilder().build());
            } else {
                responseObserver.onNext(RoleList.newBuilder().addAllRole(roles).build());
            }
            responseObserver.onCompleted();
        } catch (Exception e) {
            LOG.error(e.getMessage(), e);
            Status status = Status.UNKNOWN.withDescription(e.toString());
            responseObserver.onError(status.asException());
        }
    }

    @Override
    public void getOpenIdConnectIssuer(Empty request, StreamObserver<OpenIdConnectIssuerReply> responseObserver) {
        try {
            LOG.debug("OpenIdConnectIssuer requested. Returning '{}'", settings.getOpenIdConnectIssuer());
            responseObserver.onNext(OpenIdConnectIssuerReply.newBuilder()
                    .setOpenIdConnectIssuer(settings.getOpenIdConnectIssuer()).build());
            responseObserver.onCompleted();
        } catch (Exception ex) {
            LOG.error(ex.getMessage(), ex);
            Status status = Status.UNKNOWN.withDescription(ex.toString());
            responseObserver.onError(status.asException());
        }
    }

    @Override
    @AllowedRoles({Role.OPERATOR, Role.ADMIN})
    public void pauseCrawler(Empty request, StreamObserver<Empty> responseObserver) {
        try {
            executionsAdapter.setDesiredPausedState(true);
            responseObserver.onNext(Empty.newBuilder().build());
            responseObserver.onCompleted();
        } catch (Exception e) {
            LOG.error(e.getMessage(), e);
            Status status = Status.UNKNOWN.withDescription(e.toString());
            responseObserver.onError(status.asException());
        }
    }

    @Override
    @AllowedRoles({Role.OPERATOR, Role.ADMIN})
    public void unPauseCrawler(Empty request, StreamObserver<Empty> responseObserver) {
        try {
            executionsAdapter.setDesiredPausedState(false);
            responseObserver.onNext(Empty.newBuilder().build());
            responseObserver.onCompleted();
        } catch (Exception e) {
            LOG.error(e.getMessage(), e);
            Status status = Status.UNKNOWN.withDescription(e.toString());
            responseObserver.onError(status.asException());
        }
    }

    @Override
    @AllowedRoles({Role.OPERATOR, Role.ADMIN, Role.CONSULTANT, Role.CURATOR, Role.ANY_USER})
    public void status(Empty request, StreamObserver<CrawlerStatus> responseObserver) {
        try {
            boolean desiredPausedState = executionsAdapter.getDesiredPausedState();
            JobExecutionUtil.queueCountAndBusyChgCount(new FutureCallback<Builder>() {
                @Override
                public void onSuccess(@Nullable Builder result) {
                    boolean isPaused = result.getBusyCrawlHostGroupCount() == 0;

                    RunStatus runStatus = desiredPausedState && isPaused
                            ? RunStatus.PAUSED
                            : desiredPausedState ? RunStatus.PAUSE_REQUESTED : RunStatus.RUNNING;

                    responseObserver.onNext(result
                            .setRunStatus(runStatus)
                            .build());
                    responseObserver.onCompleted();
                }

                @Override
                public void onFailure(Throwable t) {
                    LOG.error(t.getMessage(), t);
                    forwardRpcFailure(t, responseObserver);
                }
            });
        } catch (Exception e) {
            LOG.error(e.getMessage(), e);
            Status status = Status.UNKNOWN.withDescription(e.toString());
            responseObserver.onError(status.asException());
        }
    }

    @Override
    public void queueCountsForCrawlExecutions(
            ExecutionIds request, StreamObserver<QueueCountsResponse> responseObserver) {
        if (!validateExecutionIds(request, responseObserver)) {
            return;
        }
        JobExecutionUtil.queueCountsForCrawlExecutions(request, forwardingCallback(responseObserver));
    }

    @Override
    public void queueCountsForJobExecutions(
            ExecutionIds request, StreamObserver<QueueCountsResponse> responseObserver) {
        if (!validateExecutionIds(request, responseObserver)) {
            return;
        }
        JobExecutionUtil.queueCountsForJobExecutions(request, forwardingCallback(responseObserver));
    }

    @Override
    public void queueCountForCrawlHostGroup(CrawlHostGroup request, StreamObserver<CountResponse> responseObserver) {
        JobExecutionUtil.queueCountForCrawlHostGroup(request, new FutureCallback<CountResponse>() {
            @Override
            public void onSuccess(@Nullable CountResponse result) {
                responseObserver.onNext(result);
                responseObserver.onCompleted();
            }

            @Override
            public void onFailure(Throwable t) {
                LOG.error(t.getMessage(), t);
                forwardRpcFailure(t, responseObserver);
            }
        });
    }

    private static void forwardRpcFailure(Throwable t, StreamObserver<?> responseObserver) {
        Status status = Status.fromThrowable(t);
        if (status.getCode() == Status.Code.UNKNOWN && status.getDescription() == null) {
            status = status.withDescription(t.toString());
        }
        Metadata trailers = Status.trailersFromThrowable(t);
        responseObserver.onError(status.withCause(t).asRuntimeException(trailers));
    }

    private static boolean validateExecutionIds(ExecutionIds request, StreamObserver<?> responseObserver) {
        if (request.getIdCount() > MAX_QUEUE_COUNT_IDS) {
            responseObserver.onError(Status.INVALID_ARGUMENT
                    .withDescription("Execution id list may contain at most " + MAX_QUEUE_COUNT_IDS + " entries")
                    .asRuntimeException());
            return false;
        }
        if (request.getIdList().stream().anyMatch(id -> id.isBlank())) {
            responseObserver.onError(Status.INVALID_ARGUMENT
                    .withDescription("Execution ids must not be blank")
                    .asRuntimeException());
            return false;
        }
        return true;
    }

    private static <T> FutureCallback<T> forwardingCallback(StreamObserver<T> responseObserver) {
        return new FutureCallback<>() {
            @Override
            public void onSuccess(@Nullable T result) {
                responseObserver.onNext(result);
                responseObserver.onCompleted();
            }

            @Override
            public void onFailure(Throwable t) {
                LOG.error(t.getMessage(), t);
                forwardRpcFailure(t, responseObserver);
            }
        };
    }
}
