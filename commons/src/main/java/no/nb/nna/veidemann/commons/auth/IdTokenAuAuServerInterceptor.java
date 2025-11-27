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
package no.nb.nna.veidemann.commons.auth;

import com.nimbusds.jwt.JWTClaimsSet;
import io.grpc.Context;
import io.grpc.Contexts;
import io.grpc.Metadata;
import io.grpc.ServerCall;
import io.grpc.ServerCallHandler;
import no.nb.nna.veidemann.api.config.v1.Role;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Collection;
import java.util.List;

/**
 * Authenticate a user using OpenId connect and then map the user identity to a set of roles defined in Veidemann config.
 * <p>
 * This interceptor is must be followed by {@link AuthorisationAuAuServerInterceptor}.
 */
public class IdTokenAuAuServerInterceptor extends AuAuServerInterceptor {
    private static final Logger LOG = LoggerFactory.getLogger(IdTokenAuAuServerInterceptor.class);

    private final UserRoleMapper userRoleMapper;

    private final IdTokenValidator idTokenValidator;

    public IdTokenAuAuServerInterceptor(UserRoleMapper userRoleMapper, IdTokenValidator idTokenValidator) {
        this.userRoleMapper = userRoleMapper;
        this.idTokenValidator = idTokenValidator;
    }

    @Override
    public <ReqT, RespT> ServerCall.Listener<ReqT> interceptCall(
            ServerCall<ReqT, RespT> call,
            Metadata requestHeaders,
            ServerCallHandler<ReqT, RespT> next) {

        String method = call.getMethodDescriptor().getFullMethodName();
        LOG.debug("Method: {}", method);

        Collection<Role> roles = getRoleList();

        // Check if user is logged in
        JWTClaimsSet claims = validateBearerToken(requestHeaders);
        if (claims == null) {
            Context contextWithEmailAndRoles = Context.current()
                    .withValue(RolesContextKey.getKey(), roles);
            return Contexts.interceptCall(contextWithEmailAndRoles, call, requestHeaders, next);
        }

        // User is logged in. Add ANY_USER independently of specific role assignments
        roles.add(Role.ANY_USER);

        String email = (String) claims.getClaim("email");
        List<String> groups = (List<String>) claims.getClaim("groups");

        LOG.debug("E-mail: {}", email);
        LOG.debug("Groups: {}", groups);

        roles = userRoleMapper.getRolesForUser(email, groups, roles);
        LOG.debug("Roles: {}", roles);

        Context contextWithEmailAndRoles = Context.current()
                .withValues(EmailContextKey.getKey(), email, RolesContextKey.getKey(), roles);

        return Contexts.interceptCall(contextWithEmailAndRoles, call, requestHeaders, next);
    }

    private JWTClaimsSet validateBearerToken(Metadata requestHeaders) {
        String bearerToken = getAuthorizationToken(requestHeaders, "bearer");
        LOG.trace("Bearer token: {}", bearerToken);

        if (!bearerToken.isEmpty()) {
            JWTClaimsSet claims = idTokenValidator.verifyIdToken(bearerToken);
            LOG.trace("Claims: {}", claims);
            return claims;
        }
        return null;
    }
}
