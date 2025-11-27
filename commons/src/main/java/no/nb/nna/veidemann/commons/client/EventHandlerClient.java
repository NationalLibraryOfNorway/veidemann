/*
 * Copyright 2019 National Library of Norway.
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

package no.nb.nna.veidemann.commons.client;

import io.grpc.CallCredentials;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.Metadata;
import io.grpc.Status;
import io.grpc.Status.Code;
import io.grpc.StatusRuntimeException;
import io.opentracing.contrib.grpc.TracingClientInterceptor;
import io.opentracing.util.GlobalTracer;
import no.nb.nna.veidemann.api.eventhandler.v1.Data;
import no.nb.nna.veidemann.api.eventhandler.v1.EventHandlerGrpc;
import no.nb.nna.veidemann.api.eventhandler.v1.EventObject;
import no.nb.nna.veidemann.api.eventhandler.v1.EventObject.Severity;
import no.nb.nna.veidemann.api.eventhandler.v1.SaveRequest;
import no.nb.nna.veidemann.commons.auth.AuAuServerInterceptor;
import no.nb.nna.veidemann.commons.settings.CommonSettings;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Arrays;
import java.util.concurrent.Executor;
import java.util.concurrent.TimeUnit;

/**
 *
 */
public class EventHandlerClient implements AutoCloseable {

    private static final Logger LOG = LoggerFactory.getLogger(EventHandlerClient.class);

    private final ManagedChannel channel;

    private final EventHandlerGrpc.EventHandlerBlockingStub blockingStub;

    public EventHandlerClient(final String host, final int port, final String apiKey) {
        this(ManagedChannelBuilder.forAddress(host, port).usePlaintext(), apiKey);
        LOG.info("Event handler client pointing to " + host + ":" + port);
    }

    public EventHandlerClient(final CommonSettings settings) {
        this(settings.getEventHandlerHost(), settings.getEventHandlerPort(), settings.getEventApiKey());
    }

    public EventHandlerClient(final ManagedChannel channel, final String apiKey) {
        this.channel = channel;
        blockingStub = EventHandlerGrpc.newBlockingStub(channel).withCallCredentials(createCredentials(apiKey));
    }

    public EventHandlerClient(final ManagedChannelBuilder<?> channelBuilder, final String apiKey) {
        LOG.info("Setting up Event handler client");
        TracingClientInterceptor tracingInterceptor = TracingClientInterceptor.newBuilder().withTracer(GlobalTracer.get()).build();
        channel = channelBuilder.intercept(tracingInterceptor).build();
        blockingStub = EventHandlerGrpc.newBlockingStub(channel).withCallCredentials(createCredentials(apiKey));
    }

    public EventObject createEvent(String type, String source, Severity severity, String comment, Data... data) {
        EventObject eventObject = EventObject.newBuilder()
                .setType(type)
                .setSource(source)
                .setSeverity(severity)
                .addAllData(Arrays.asList(data))
                .build();
        return saveEventObject(eventObject, comment);
    }

    public EventObject saveEventObject(EventObject eventObject, String comment) {
        try {
            SaveRequest request = SaveRequest.newBuilder()
                    .setObject(eventObject)
                    .setComment(comment)
                    .build();

            return GrpcUtil.forkedCall(() -> blockingStub.saveEventObject(request));
        } catch (StatusRuntimeException ex) {
            Code code = ex.getStatus().getCode();
            if (code.equals(Status.CANCELLED.getCode())
                    || code.equals(Status.DEADLINE_EXCEEDED.getCode())
                    || code.equals(Status.ABORTED.getCode())) {
                LOG.warn("Request was aborted", ex);
            } else {
                LOG.error("RPC failed: " + ex.getStatus(), ex);
            }
            throw ex;
        }
    }

    @Override
    public void close() {
        try {
            channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
        } catch (InterruptedException ex) {
            throw new RuntimeException(ex);
        }
    }

    private CallCredentials createCredentials(String apiKey) {
        return new CallCredentials() {
            @Override
            public void applyRequestMetadata(RequestInfo requestInfo, Executor appExecutor, MetadataApplier applier) {
                Metadata headers = new Metadata();
                headers.put(AuAuServerInterceptor.AUTHORIZATION_KEY, "ApiKey " + apiKey);
                applier.apply(headers);
            }

            @Override
            public void thisUsesUnstableApi() {

            }
        };
    }

}
