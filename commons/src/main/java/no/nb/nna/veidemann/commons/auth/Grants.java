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

import no.nb.nna.veidemann.api.config.v1.Kind;
import no.nb.nna.veidemann.api.config.v1.Role;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.lang.reflect.Method;
import java.util.Arrays;
import java.util.Collection;
import java.util.EnumMap;
import java.util.HashMap;
import java.util.Map;
import java.util.Map.Entry;
import java.util.Objects;
import java.util.StringJoiner;

public class Grants {
    private static final Logger LOG = LoggerFactory.getLogger(Grants.class);

    Map<String, Grant> grantsForMethod = new HashMap<>();

    protected void addGrants(String method, Method methodDef) {
        AllowedRoles allowedRolesAnnotation = methodDef.getDeclaredAnnotation(AllowedRoles.class);
        if (allowedRolesAnnotation != null) {
            addGrants(method, allowedRolesAnnotation);
        }

        Authorisations authorisations = methodDef.getDeclaredAnnotation(Authorisations.class);
        if (authorisations != null) {
            for (AllowedRoles allowedRoles : authorisations.value()) {
                addGrants(method, allowedRoles);
            }
        }
    }

    private void addGrants(String method, AllowedRoles allowedRolesAnnotation) {
        for (Kind kind : allowedRolesAnnotation.kind()) {
            if (allowedRolesAnnotation != null && !Arrays.asList(allowedRolesAnnotation.value()).contains(Role.ANY)) {
                grantsForMethod.computeIfAbsent(method, m -> new Grant()).addKind(kind, allowedRolesAnnotation.value());
            }
        }
    }

    public boolean isRequireAuthenticatedUser(String method) {
        return grantsForMethod.get(method) != null;
    }

    public boolean isAllowed(String method, Collection<Role> roles) {
        return isAllowed(method, Kind.undefined, roles);
    }

    public boolean isAllowed(String method, Kind kind, Collection<Role> roles) {
        Grant grant = grantsForMethod.get(method);
        LOG.trace("Checking access for method: {}, kind: {}, roles: {}, using grant: {}", method, kind, roles, grant);
        if (grant == null) {
            return true;
        }
        return grant.isAllowed(kind, roles);
    }

    static class Grant {
        private final EnumMap<Kind, Role[]> rolesForKind = new EnumMap<>(Kind.class);

        public final Grant addKind(Kind kind, Role[] allowedRoles) {
            this.rolesForKind.put(kind, allowedRoles);
            return this;
        }

        boolean isAllowed(Kind kind, Collection<Role> roles) {
            Role[] allowedRoles = rolesForKind.get(kind);
            if (allowedRoles == null && kind != Kind.undefined) {
                allowedRoles = rolesForKind.get(Kind.undefined);
            }
            if (allowedRoles == null) {
                return false;
            }

            for (Role allowedRole : allowedRoles) {
                if (allowedRole == Role.ANY_USER) {
                    return true;
                }
                for (Role role : roles) {
                    if (allowedRole == role) {
                        return true;
                    }
                }
            }
            return false;
        }

        @Override
        public boolean equals(Object o) {
            if (this == o) return true;
            if (o == null || getClass() != o.getClass()) return false;
            Grant grant = (Grant) o;
            if (rolesForKind.size() != grant.rolesForKind.size()) {
                return false;
            }
            return rolesForKind.entrySet().stream()
                    .allMatch(e -> Arrays.equals(e.getValue(), grant.rolesForKind.get(e.getKey())));
        }

        @Override
        public int hashCode() {
            return Objects.hash(rolesForKind);
        }

        @Override
        public String toString() {
            try {
                StringJoiner rfk = new StringJoiner(",", "rolesForKind={", "}");
                for (Entry<Kind, Role[]> e : rolesForKind.entrySet()) {
                    StringJoiner roles = new StringJoiner(", ", e.getKey() + "=[", "]");
                    for (Role r : e.getValue()) {
                        roles.add(r.toString());
                    }
                    rfk.add(roles.toString());
                }
                return Grant.class.getSimpleName() + "[" + rfk.toString() + "]";
            } catch (Throwable e) {
                e.printStackTrace();
                return e.toString();
            }
        }
    }
}
