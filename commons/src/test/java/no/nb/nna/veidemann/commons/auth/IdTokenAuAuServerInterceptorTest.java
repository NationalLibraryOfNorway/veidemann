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
import com.nimbusds.jwt.JWTClaimsSet;
import io.grpc.Attributes;
import io.grpc.CallCredentials;
import io.grpc.ManagedChannel;
import io.grpc.Metadata;
import io.grpc.MethodDescriptor;
import io.grpc.Server;
import io.grpc.inprocess.InProcessChannelBuilder;
import io.grpc.inprocess.InProcessServerBuilder;
import io.grpc.stub.StreamObserver;
import net.minidev.json.JSONArray;
import no.nb.nna.veidemann.api.config.v1.Role;
import no.nb.nna.veidemann.api.ooshandler.v1.OosHandlerGrpc;
import no.nb.nna.veidemann.api.ooshandler.v1.SubmitUriRequest;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.stubbing.Answer;

import java.io.IOException;
import java.util.Collection;
import java.util.concurrent.Executor;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.anyCollection;
import static org.mockito.Mockito.anyList;
import static org.mockito.Mockito.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

public class IdTokenAuAuServerInterceptorTest {

    private final String uniqueServerName = "in-process server for " + getClass();

    private InProcessServerBuilder inProcessServerBuilder;

    private Server inProcessServer;

    private ManagedChannel inProcessChannel;

    private OosHandlerGrpc.OosHandlerBlockingStub blockingStub;

    private OosHandlerGrpc.OosHandlerStub asyncStub;

    private TestService service;

    @BeforeEach
    public void beforeEachTest() throws IOException {
        inProcessServerBuilder = InProcessServerBuilder.forName(uniqueServerName).directExecutor();
        inProcessChannel = InProcessChannelBuilder.forName(uniqueServerName).directExecutor().build();
        blockingStub = OosHandlerGrpc.newBlockingStub(inProcessChannel);
        asyncStub = OosHandlerGrpc.newStub(inProcessChannel);

        IdTokenValidator idValidatorMock = mock(IdTokenValidator.class);
        UserRoleMapper roleMapperMock = mock(UserRoleMapper.class);
        when(idValidatorMock.verifyIdToken("token1"))
                .thenReturn(new JWTClaimsSet.Builder()
                        .claim("email", "user@example.com")
                        .claim("groups", new JSONArray())
                        .build());
        when(roleMapperMock.getRolesForUser(eq("user@example.com"), anyList(), anyCollection()))
                .thenAnswer((Answer<Collection<Role>>) invocation -> {
                    Collection<Role> roles = invocation.getArgument(2);
                    roles.add(Role.READONLY);
                    return roles;
                });

        service = new TestService();
        IdTokenAuAuServerInterceptor idTokenAuAuServerInterceptor = new IdTokenAuAuServerInterceptor(roleMapperMock, idValidatorMock);
        inProcessServer = inProcessServerBuilder
                .addService(idTokenAuAuServerInterceptor.intercept(service))
                .build();
        inProcessServer.start();
    }

    @AfterEach
    public void afterEachTest() {
        inProcessServer.shutdownNow();
        inProcessChannel.shutdownNow();
    }

    @Test
    public void interceptCall() {
        CallCredentials cred = createCredentials("token1");

        blockingStub.withCallCredentials(cred).submitUri(SubmitUriRequest.getDefaultInstance());
        assertThat(service.lastRoles).containsExactlyInAnyOrder(Role.READONLY, Role.ANY_USER);

        blockingStub.submitUri(SubmitUriRequest.getDefaultInstance());
        assertThat(service.lastRoles).isEmpty();
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

    private CallCredentials createCredentials(String bearerToken) {
        return new CallCredentials() {
            @Override
            public void applyRequestMetadata(RequestInfo requestInfo, Executor appExecutor, MetadataApplier applier) {
                Metadata headers = new Metadata();
                headers.put(AuAuServerInterceptor.AUTHORIZATION_KEY, "Bearer " + bearerToken);
                applier.apply(headers);
            }

            @Override
            public void thisUsesUnstableApi() {

            }
        };
    }
}