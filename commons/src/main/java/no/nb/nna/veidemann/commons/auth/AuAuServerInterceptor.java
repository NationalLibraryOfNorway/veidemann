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

import io.grpc.BindableService;
import io.grpc.Metadata;
import io.grpc.ServerInterceptor;
import io.grpc.ServerInterceptors;
import io.grpc.ServerServiceDefinition;
import no.nb.nna.veidemann.api.config.v1.Role;

import java.util.Collection;
import java.util.HashSet;

public abstract class AuAuServerInterceptor implements ServerInterceptor, AutoCloseable {
    public static final Metadata.Key<String> AUTHORIZATION_KEY =
            Metadata.Key.of("authorization", Metadata.ASCII_STRING_MARSHALLER);

    public ServerServiceDefinition intercept(BindableService bindableService) {
        return ServerInterceptors.intercept(bindableService, this);
    }

    public ServerServiceDefinition intercept(ServerServiceDefinition serviceDef) {
        return ServerInterceptors.intercept(serviceDef, this);
    }

    public Collection<Role> getRoleList() {
        Collection<Role> roles = RolesContextKey.roles();
        if (roles == null) {
            roles = new HashSet<>();
        }
        return roles;
    }

    @Override
    public void close() {
    }

    public String getAuthorizationToken(Metadata requestHeaders, String type) {
        type = type.toLowerCase();
        Iterable<String> authHeaders = requestHeaders.getAll(AUTHORIZATION_KEY);
        if (authHeaders != null) {
            for (String h : requestHeaders.getAll(AUTHORIZATION_KEY)) {
                if (h.toLowerCase().startsWith(type)) {
                    String[] parts = h.split("\\s+", 2);
                    if (parts.length == 2) {
                        return parts[1];
                    }
                    return h;
                }
            }
        }
        return "";
    }
}