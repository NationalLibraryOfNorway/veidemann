/*
 * Copyright 2018 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package no.nb.nna.veidemann.controller;

import com.google.protobuf.Empty;
import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.grpc.stub.StreamObserver;
import no.nb.nna.veidemann.api.config.v1.Annotation;
import no.nb.nna.veidemann.api.config.v1.ConfigGrpc;
import no.nb.nna.veidemann.api.config.v1.ConfigObject;
import no.nb.nna.veidemann.api.config.v1.ConfigRef;
import no.nb.nna.veidemann.api.config.v1.DeleteResponse;
import no.nb.nna.veidemann.api.config.v1.GetLabelKeysRequest;
import no.nb.nna.veidemann.api.config.v1.GetScriptAnnotationsRequest;
import no.nb.nna.veidemann.api.config.v1.GetScriptAnnotationsResponse;
import no.nb.nna.veidemann.api.config.v1.Kind;
import no.nb.nna.veidemann.api.config.v1.LabelKeysResponse;
import no.nb.nna.veidemann.api.config.v1.ListCountResponse;
import no.nb.nna.veidemann.api.config.v1.ListRequest;
import no.nb.nna.veidemann.api.config.v1.LogLevels;
import no.nb.nna.veidemann.api.config.v1.Role;
import no.nb.nna.veidemann.api.config.v1.UpdateRequest;
import no.nb.nna.veidemann.api.config.v1.UpdateResponse;
import no.nb.nna.veidemann.api.config.v1.UpdateTaskAccepted;
import no.nb.nna.veidemann.commons.auth.AllowedRoles;
import no.nb.nna.veidemann.commons.auth.Authorisations;
import no.nb.nna.veidemann.commons.auth.EmailContextKey;
import no.nb.nna.veidemann.commons.db.ChangeFeed;
import no.nb.nna.veidemann.commons.db.ConfigAdapter;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Map;
import java.util.UUID;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.RejectedExecutionException;
import java.util.concurrent.atomic.AtomicBoolean;

import static no.nb.nna.veidemann.controller.JobExecutionUtil.handleGet;

public class ConfigService extends ConfigGrpc.ConfigImplBase implements AutoCloseable {

    private static final Logger LOG = LoggerFactory.getLogger(ConfigService.class);

    private final ConfigAdapter db;

    private final ScopeServiceClient scopeServiceClient;

    private final ExecutorService updateExecutor;

    private final AtomicBoolean updateActive = new AtomicBoolean();

    public ConfigService(ConfigAdapter db, ScopeServiceClient scopeServiceClient) {
        this(db, scopeServiceClient, Executors.newSingleThreadExecutor(
                Thread.ofPlatform().name("config-background-update").factory()));
    }

    ConfigService(ConfigAdapter db, ScopeServiceClient scopeServiceClient, ExecutorService updateExecutor) {
        this.db = db;
        this.scopeServiceClient = scopeServiceClient;
        this.updateExecutor = updateExecutor;
    }

    @Override
    @AllowedRoles({Role.READONLY, Role.CURATOR, Role.OPERATOR, Role.ADMIN, Role.CONSULTANT})
    public void getConfigObject(ConfigRef request, StreamObserver<ConfigObject> responseObserver) {
        handleGet(() -> db.getConfigObject(request), responseObserver);
    }

    @Override
    @Authorisations({
            @AllowedRoles(value = {Role.READONLY, Role.CURATOR, Role.OPERATOR, Role.ADMIN, Role.CONSULTANT}),
            @AllowedRoles(value = {Role.ADMIN}, kind = {Kind.roleMapping}),
    })
    public void listConfigObjects(ListRequest request, StreamObserver<ConfigObject> observer) {
        StreamObserver<ConfigObject> responseObserver = new BlockingStreamObserver<>(observer);
        new Thread(() -> {
            try (ChangeFeed<ConfigObject> c = db.listConfigObjects(request);) {
                c.stream().forEach(o -> responseObserver.onNext(o));
                responseObserver.onCompleted();
            } catch (StatusRuntimeException e) {
                LOG.error(e.getMessage(), e);
                responseObserver.onError(e);
            } catch (Exception ex) {
                LOG.error(ex.getMessage(), ex);
                Status status = Status.UNKNOWN.withDescription(ex.toString());
                responseObserver.onError(status.asException());
            }
        }).start();
    }

    @Override
    @Authorisations({
            @AllowedRoles({Role.READONLY, Role.CURATOR, Role.OPERATOR, Role.ADMIN, Role.CONSULTANT}),
            @AllowedRoles(value = {Role.ADMIN}, kind = {Kind.roleMapping}),
    })
    public void countConfigObjects(ListRequest request, StreamObserver<ListCountResponse> responseObserver) {
        handleGet(() -> db.countConfigObjects(request), responseObserver);
    }

    @Override
    @Authorisations({
            @AllowedRoles({Role.CURATOR, Role.OPERATOR, Role.ADMIN}),
            @AllowedRoles(value = {Role.ADMIN}, kind = {Kind.roleMapping}),
            @AllowedRoles(value = {Role.CONSULTANT, Role.CURATOR, Role.OPERATOR, Role.ADMIN},
                    kind = {Kind.seed, Kind.crawlEntity})
    })
    public void saveConfigObject(ConfigObject request, StreamObserver<ConfigObject> responseObserver) {
        try {
            // If kind is seed, canonicalize uri
            if (request.getKind() == Kind.seed) {
                String canonicalizedUri = scopeServiceClient.canonicalize(request.getMeta().getName());
                if (!canonicalizedUri.equals(request.getMeta().getName())) {
                    ConfigObject.Builder b = request.toBuilder();
                    b.getMetaBuilder().setName(canonicalizedUri);
                    request = b.build();
                }
            }

            responseObserver.onNext(db.saveConfigObject(request));
            responseObserver.onCompleted();
        } catch (Exception ex) {
            LOG.error(ex.getMessage(), ex);
            Status status = Status.UNKNOWN.withDescription(ex.toString());
            responseObserver.onError(status.asException());
        }
    }

    @Override
    @Authorisations({
            @AllowedRoles({Role.CURATOR, Role.OPERATOR, Role.ADMIN}),
            @AllowedRoles(value = {Role.ADMIN}, kind = {Kind.roleMapping}),
            @AllowedRoles(value = {Role.ADMIN, Role.CURATOR, Role.OPERATOR, Role.CONSULTANT},
                    kind = {Kind.crawlEntity, Kind.seed})
    })
    public void updateConfigObjects(UpdateRequest request, StreamObserver<UpdateResponse> responseObserver) {
        try {
            responseObserver.onNext(db.updateConfigObjects(request));
            responseObserver.onCompleted();
        } catch (Exception ex) {
            LOG.error(ex.getMessage(), ex);
            Status status = Status.UNKNOWN.withDescription(ex.toString());
            responseObserver.onError(status.asException());
        }
    }

    @Override
    @Authorisations({
            @AllowedRoles({Role.CURATOR, Role.OPERATOR, Role.ADMIN}),
            @AllowedRoles(value = {Role.ADMIN}, kind = {Kind.roleMapping}),
            @AllowedRoles(value = {Role.ADMIN, Role.CURATOR, Role.OPERATOR, Role.CONSULTANT},
                    kind = {Kind.crawlEntity, Kind.seed})
    })
    public void startUpdateConfigObjects(UpdateRequest request,
            StreamObserver<UpdateTaskAccepted> responseObserver) {
        if (!updateActive.compareAndSet(false, true)) {
            responseObserver.onError(Status.RESOURCE_EXHAUSTED
                    .withDescription("A database-wide configuration update is already running")
                    .asException());
            return;
        }

        String taskId = UUID.randomUUID().toString();
        String submittedBy = EmailContextKey.email();
        io.grpc.Context taskContext = io.grpc.Context.current().fork();
        try {
            updateExecutor.execute(taskContext.wrap(() -> runUpdateTask(taskId, submittedBy, request)));
        } catch (RejectedExecutionException e) {
            updateActive.set(false);
            LOG.warn("Configuration update task {} was rejected", taskId, e);
            responseObserver.onError(Status.UNAVAILABLE
                    .withDescription("The background update executor is unavailable")
                    .withCause(e)
                    .asException());
            return;
        }

        LOG.info("Accepted configuration update task {} submittedBy={} kind={}",
                taskId, submittedBy, request.getListRequest().getKind());
        responseObserver.onNext(UpdateTaskAccepted.newBuilder().setTaskId(taskId).build());
        responseObserver.onCompleted();
    }

    private void runUpdateTask(String taskId, String submittedBy, UpdateRequest request) {
        long started = System.nanoTime();
        LOG.info("Starting configuration update task {} submittedBy={} kind={}",
                taskId, submittedBy, request.getListRequest().getKind());
        try {
            UpdateResponse response = db.updateConfigObjects(request);
            LOG.info("Completed configuration update task {} submittedBy={} kind={} updated={} durationMs={}",
                    taskId, submittedBy, request.getListRequest().getKind(), response.getUpdated(),
                    java.util.concurrent.TimeUnit.NANOSECONDS.toMillis(System.nanoTime() - started));
        } catch (Exception e) {
            LOG.error("Failed configuration update task {} submittedBy={} kind={} durationMs={}",
                    taskId, submittedBy, request.getListRequest().getKind(),
                    java.util.concurrent.TimeUnit.NANOSECONDS.toMillis(System.nanoTime() - started), e);
        } finally {
            updateActive.set(false);
        }
    }

    @Override
    public void close() {
        updateExecutor.shutdownNow();
    }

    @Override
    @Authorisations({
            @AllowedRoles({Role.ADMIN}),
            @AllowedRoles(value = {Role.ADMIN}, kind = {Kind.roleMapping})
    })
    public void deleteConfigObject(ConfigObject request, StreamObserver<DeleteResponse> responseObserver) {
        try {
            responseObserver.onNext(db.deleteConfigObject(request));
            responseObserver.onCompleted();
        } catch (Exception ex) {
            LOG.error(ex.getMessage(), ex);
            Status status = Status.UNKNOWN.withDescription(ex.toString());
            responseObserver.onError(status.asException());
        }
    }

    @Override
    @Authorisations({
            @AllowedRoles({Role.READONLY, Role.CURATOR, Role.OPERATOR, Role.ADMIN, Role.CONSULTANT}),
            @AllowedRoles(value = {Role.ADMIN}, kind = {Kind.roleMapping})
    })
    public void getLabelKeys(GetLabelKeysRequest request, StreamObserver<LabelKeysResponse> responseObserver) {
        handleGet(() -> db.getLabelKeys(request), responseObserver);
    }

    /**
     * @param request
     * @param responseObserver
     */
    @Override
    @Authorisations({
            @AllowedRoles({Role.CURATOR, Role.OPERATOR, Role.ADMIN}),
    })
    public void getScriptAnnotations(GetScriptAnnotationsRequest request, StreamObserver<GetScriptAnnotationsResponse> responseObserver) {
        handleGet(() -> {

            ConfigObject jobConfig = db.getConfigObject(request.getJob());
            Map<String, Annotation> annotations = JobExecutionUtil.GetScriptAnnotationsForJob(jobConfig);

            if (request.hasSeed()) {
                ConfigObject seed = db.getConfigObject(request.getSeed());
                annotations = JobExecutionUtil.GetScriptAnnotationOverridesForSeed(seed, jobConfig, annotations);
            }

            GetScriptAnnotationsResponse response = GetScriptAnnotationsResponse.newBuilder()
                    .addAllAnnotation(annotations.values())
                    .build();

            return response;
        }, responseObserver);
    }

    @Override
    @AllowedRoles({Role.OPERATOR, Role.ADMIN})
    public void saveLogConfig(LogLevels request, StreamObserver<LogLevels> responseObserver) {
        try {
            responseObserver.onNext(db.saveLogConfig(request));
            responseObserver.onCompleted();
        } catch (Exception ex) {
            LOG.error(ex.getMessage(), ex);
            Status status = Status.UNKNOWN.withDescription(ex.toString());
            responseObserver.onError(status.asException());
        }
    }

    @Override
    @AllowedRoles({Role.READONLY, Role.OPERATOR, Role.ADMIN})
    public void getLogConfig(Empty request, StreamObserver<LogLevels> responseObserver) {
        try {
            responseObserver.onNext(db.getLogConfig());
            responseObserver.onCompleted();
        } catch (Exception ex) {
            LOG.error(ex.getMessage(), ex);
            Status status = Status.UNKNOWN.withDescription(ex.toString());
            responseObserver.onError(status.asException());
        }
    }
}
