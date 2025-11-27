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

import io.grpc.BindableService;
import io.grpc.Context;
import io.grpc.Contexts;
import io.grpc.ForwardingServerCallListener.SimpleForwardingServerCallListener;
import io.grpc.Metadata;
import io.grpc.ServerCall;
import io.grpc.ServerCall.Listener;
import io.grpc.ServerCallHandler;
import io.grpc.Status;
import no.nb.nna.veidemann.api.config.v1.ConfigObject;
import no.nb.nna.veidemann.api.config.v1.ConfigRef;
import no.nb.nna.veidemann.api.config.v1.GetLabelKeysRequest;
import no.nb.nna.veidemann.api.config.v1.Kind;
import no.nb.nna.veidemann.api.config.v1.ListRequest;
import no.nb.nna.veidemann.api.config.v1.Role;
import no.nb.nna.veidemann.api.config.v1.UpdateRequest;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Arrays;
import java.util.Collection;

/**
 * Add authorization to all requests made to this service.
 * <p>
 * This interceptor is meant to be inserted after one or more {@link AuAuServerInterceptor}s
 * which map credentials into a list of roles.
 */
public class AuthorisationAuAuServerInterceptor extends AuAuServerInterceptor {
    private static final Logger LOG = LoggerFactory.getLogger(AuthorisationAuAuServerInterceptor.class);

    public static final Listener NOOP_LISTENER = new Listener() {
    };

    private Grants grants = new Grants();

    public AuthorisationAuAuServerInterceptor(BindableService service) {
        Class serviceClass = service.getClass();

        String serviceName = service.bindService().getServiceDescriptor().getName();
        Arrays.stream(serviceClass.getDeclaredMethods())
                .forEach(m -> {
                    grants.addGrants(serviceName + "/" + m.getName().substring(0, 1).toUpperCase() + m.getName().substring(1), m);
                });
    }

    @Override
    public <ReqT, RespT> Listener<ReqT> interceptCall(
            ServerCall<ReqT, RespT> call,
            Metadata requestHeaders,
            ServerCallHandler<ReqT, RespT> next) {

        String method = call.getMethodDescriptor().getFullMethodName();
        LOG.debug("Method: {}", method);

        Collection<Role> roles = getRoleList();

        // All users should have the ANY role. Even if their not logged in.
        roles.add(Role.ANY);

        // Check if user is logged in
        if (!roles.contains(Role.ANY_USER)) {
            if (grants.isRequireAuthenticatedUser(method)) {
                call.close(Status.UNAUTHENTICATED, new Metadata());
                return NOOP_LISTENER;
            } else {
                Context contextWithEmailAndRoles = Context.current()
                        .withValue(RolesContextKey.getKey(), roles);
                return Contexts.interceptCall(contextWithEmailAndRoles, call, requestHeaders, next);
            }
        }

        // If the call is for the config service, then we also need to take the Kind from the request into account
        if ("veidemann.api.config.v1.Config".equals(call.getMethodDescriptor().getServiceName())) {
            ServerCall.Listener<ReqT> nextListener = next.startCall(call, requestHeaders);

            nextListener = new SimpleForwardingServerCallListener<ReqT>(nextListener) {
                @Override
                public void onMessage(ReqT message) {
                    Kind kind;
                    if (message instanceof ConfigRef) {
                        kind = ((ConfigRef) message).getKind();
                    } else if (message instanceof ConfigObject) {
                        kind = ((ConfigObject) message).getKind();
                    } else if (message instanceof ListRequest) {
                        kind = ((ListRequest) message).getKind();
                    } else if (message instanceof UpdateRequest) {
                        kind = ((UpdateRequest) message).getListRequest().getKind();
                    } else if (message instanceof GetLabelKeysRequest) {
                        kind = ((GetLabelKeysRequest) message).getKind();
                    } else {
                        kind = Kind.undefined;
                    }

                    LOG.debug("Kind: {}", kind);

                    // Check if user has required role
                    if (!grants.isAllowed(method, kind, roles)) {
                        call.close(Status.PERMISSION_DENIED, new Metadata());
                    } else {
                        super.onMessage(message);
                    }
                }
            };

            return nextListener;
        }

        // Check if user has required role
        if (!grants.isAllowed(method, roles)) {
            call.close(Status.PERMISSION_DENIED, new Metadata());
            return NOOP_LISTENER;
        }

        return next.startCall(call, requestHeaders);
    }
}
