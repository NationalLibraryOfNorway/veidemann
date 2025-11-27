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

package no.nb.nna.veidemann.commons.auth;

import com.google.protobuf.Empty;
import io.grpc.Attributes;
import io.grpc.CallCredentials;
import io.grpc.ManagedChannel;
import io.grpc.Metadata;
import io.grpc.MethodDescriptor;
import io.grpc.Server;
import io.grpc.inprocess.InProcessChannelBuilder;
import io.grpc.inprocess.InProcessServerBuilder;
import io.grpc.stub.StreamObserver;
import no.nb.nna.veidemann.api.config.v1.Role;
import no.nb.nna.veidemann.api.ooshandler.v1.OosHandlerGrpc;
import no.nb.nna.veidemann.api.ooshandler.v1.SubmitUriRequest;
import org.assertj.core.api.Assertions;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.util.Collection;
import java.util.concurrent.Executor;

public class ApiKeyAuAuServerInterceptorTest {

    private final String uniqueServerName = "in-process server for " + getClass();

    private InProcessServerBuilder inProcessServerBuilder;

    private Server inProcessServer;

    private ManagedChannel inProcessChannel;

    private OosHandlerGrpc.OosHandlerBlockingStub blockingStub;

    private TestService service;

    @BeforeEach
    public void beforeEachTest() throws IOException {
        inProcessServerBuilder = InProcessServerBuilder.forName(uniqueServerName).directExecutor();
        inProcessChannel = InProcessChannelBuilder.forName(uniqueServerName).directExecutor().build();
        blockingStub = OosHandlerGrpc.newBlockingStub(inProcessChannel);

        service = new TestService();
        ApiKeyRoleMapper apiKeyRoleMapper = new ApiKeyRoleMapperFromFile("src/test/resources/apikey_rolemapping");
        ApiKeyAuAuServerInterceptor apiKeyAuAuServerInterceptor = new ApiKeyAuAuServerInterceptor(apiKeyRoleMapper);
        ApiKeyRoleMapper apiKeyRoleMapper2 = new ApiKeyRoleMapperFromFile("src/test/resources/apikey_rolemapping2");
        ApiKeyAuAuServerInterceptor apiKeyAuAuServerInterceptor2 = new ApiKeyAuAuServerInterceptor(apiKeyRoleMapper2);
        inProcessServer = inProcessServerBuilder
                .addService(apiKeyAuAuServerInterceptor.intercept(apiKeyAuAuServerInterceptor2.intercept(service)))
                .build().start();
    }

    @AfterEach
    public void afterEachTest() {
        inProcessServer.shutdownNow();
        inProcessChannel.shutdownNow();
    }

    @Test
    public void interceptCall() {
        blockingStub.withCallCredentials(createCredentials("myApiKey")).submitUri(SubmitUriRequest.getDefaultInstance());
        Assertions.assertThat(service.lastRoles).containsExactlyInAnyOrder(Role.ANY_USER, Role.SYSTEM, Role.OPERATOR);
    }

    public static class TestService extends OosHandlerGrpc.OosHandlerImplBase {
        Collection<Role> lastRoles;

        @Override
        @AllowedRoles({Role.ADMIN, Role.CURATOR, Role.READONLY})
        public void submitUri(SubmitUriRequest request, StreamObserver<Empty> responseObserver) {
            lastRoles = RolesContextKey.roles();
            responseObserver.onNext(Empty.getDefaultInstance());
            responseObserver.onCompleted();
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