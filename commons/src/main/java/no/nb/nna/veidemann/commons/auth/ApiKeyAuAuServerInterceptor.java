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

import io.grpc.Context;
import io.grpc.Contexts;
import io.grpc.Metadata;
import io.grpc.ServerCall;
import io.grpc.ServerCall.Listener;
import io.grpc.ServerCallHandler;
import no.nb.nna.veidemann.api.config.v1.Role;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Collection;

/**
 * Authenticate a client by mapping an api key to a set of roles defined in an {@link ApiKeyRoleMapper}.
 * <p>
 * This interceptor must be followed by {@link AuthorisationAuAuServerInterceptor}.
 */
public class ApiKeyAuAuServerInterceptor extends AuAuServerInterceptor {
    private static final Logger LOG = LoggerFactory.getLogger(IdTokenAuAuServerInterceptor.class);

    private final ApiKeyRoleMapper apiKeyRoleMapper;

    public ApiKeyAuAuServerInterceptor(ApiKeyRoleMapper apiKeyRoleMapper) {
        this.apiKeyRoleMapper = apiKeyRoleMapper;
    }

    @Override
    public <ReqT, RespT> Listener<ReqT> interceptCall(ServerCall<ReqT, RespT> call, Metadata requestHeaders, ServerCallHandler<ReqT, RespT> next) {
        String apiKey = getAuthorizationToken(requestHeaders, "apikey");
        if (!apiKey.isEmpty()) {
            Collection<Role> roles = getRoleList();
            roles.addAll(apiKeyRoleMapper.getRolesForAPiKey(apiKey, roles));

            Context contextWithAllRoles = Context.current()
                    .withValue(RolesContextKey.getKey(), roles);
            return Contexts.interceptCall(contextWithAllRoles, call, requestHeaders, next);
        }

        return next.startCall(call, requestHeaders);
    }
}
