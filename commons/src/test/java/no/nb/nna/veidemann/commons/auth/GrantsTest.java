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

import com.google.common.collect.ImmutableList;
import io.grpc.stub.StreamObserver;
import no.nb.nna.veidemann.api.config.v1.Kind;
import no.nb.nna.veidemann.api.config.v1.Role;
import no.nb.nna.veidemann.api.eventhandler.v1.EventHandlerGrpc;
import no.nb.nna.veidemann.api.eventhandler.v1.EventObject;
import no.nb.nna.veidemann.api.eventhandler.v1.EventRef;
import no.nb.nna.veidemann.api.eventhandler.v1.ListCountResponse;
import no.nb.nna.veidemann.api.eventhandler.v1.ListRequest;
import no.nb.nna.veidemann.api.eventhandler.v1.SaveRequest;
import no.nb.nna.veidemann.commons.auth.Grants.Grant;

import java.lang.reflect.Method;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.entry;

class GrantsTest {

    @org.junit.jupiter.api.Test
    void addGrants() {
        Grants grants = new Grants();
        String methodName = "countEventObjects";
        grants.addGrants(methodName, getMethodDef(methodName));
        assertThat(grants.grantsForMethod).hasSize(0);

        grants = new Grants();
        methodName = "saveEventObject";
        grants.addGrants(methodName, getMethodDef(methodName));
        assertThat(grants.grantsForMethod).hasSize(1)
                .containsOnly(
                        entry(methodName, new Grant()
                                .addKind(Kind.roleMapping, new Role[]{Role.ADMIN, Role.CURATOR, Role.SYSTEM})
                                .addKind(Kind.browserConfig, new Role[]{Role.ADMIN, Role.CURATOR, Role.SYSTEM}))
                );

        grants = new Grants();
        methodName = "getEventObject";
        grants.addGrants(methodName, getMethodDef(methodName));
        assertThat(grants.grantsForMethod).hasSize(1)
                .containsOnly(
                        entry(methodName, new Grant()
                                .addKind(Kind.undefined, new Role[]{Role.ADMIN})
                                .addKind(Kind.browserScript, new Role[]{Role.READONLY, Role.CURATOR, Role.OPERATOR, Role.ADMIN})
                                .addKind(Kind.browserConfig, new Role[]{Role.READONLY, Role.CURATOR, Role.OPERATOR, Role.ADMIN}))
                );
    }

    @org.junit.jupiter.api.Test
    void isRequireAuthenticatedUser() {
        Grants grants = new Grants();
        String methodName = "countEventObjects";
        grants.addGrants(methodName, getMethodDef(methodName));
        assertThat(grants.isRequireAuthenticatedUser(methodName)).isFalse();

        grants = new Grants();
        methodName = "saveEventObject";
        grants.addGrants(methodName, getMethodDef(methodName));
        assertThat(grants.isRequireAuthenticatedUser(methodName)).isTrue();

        grants = new Grants();
        methodName = "getEventObject";
        grants.addGrants(methodName, getMethodDef(methodName));
        assertThat(grants.isRequireAuthenticatedUser(methodName)).isTrue();
    }

    @org.junit.jupiter.api.Test
    void isAllowed_method() {
        Grants grants = new Grants();
        String methodName = "countEventObjects";
        grants.addGrants(methodName, getMethodDef(methodName));
        assertThat(grants.isAllowed(methodName, ImmutableList.of(Role.ANY))).isTrue();
        assertThat(grants.isAllowed(methodName, ImmutableList.of(Role.CURATOR))).isTrue();

        grants = new Grants();
        methodName = "saveEventObject";
        grants.addGrants(methodName, getMethodDef(methodName));
        assertThat(grants.isAllowed(methodName, ImmutableList.of(Role.ANY))).isFalse();
        assertThat(grants.isAllowed(methodName, ImmutableList.of(Role.CURATOR))).isFalse();

        grants = new Grants();
        methodName = "getEventObject";
        grants.addGrants(methodName, getMethodDef(methodName));
        assertThat(grants.isAllowed(methodName, ImmutableList.of(Role.ANY))).isFalse();
        assertThat(grants.isAllowed(methodName, ImmutableList.of(Role.CURATOR))).isFalse();
        assertThat(grants.isAllowed(methodName, ImmutableList.of(Role.ADMIN))).isTrue();
    }

    @org.junit.jupiter.api.Test
    void isAllowed_method_kind() {
        Grants grants = new Grants();
        String methodName = "countEventObjects";
        grants.addGrants(methodName, getMethodDef(methodName));
        assertThat(grants.isAllowed(methodName, Kind.undefined, ImmutableList.of(Role.ANY))).isTrue();
        assertThat(grants.isAllowed(methodName, Kind.browserConfig, ImmutableList.of(Role.ANY))).isTrue();
        assertThat(grants.isAllowed(methodName, Kind.undefined, ImmutableList.of(Role.CURATOR))).isTrue();
        assertThat(grants.isAllowed(methodName, Kind.browserConfig, ImmutableList.of(Role.CURATOR))).isTrue();

        grants = new Grants();
        methodName = "saveEventObject";
        grants.addGrants(methodName, getMethodDef(methodName));
        assertThat(grants.isAllowed(methodName, Kind.undefined, ImmutableList.of(Role.ANY))).isFalse();
        assertThat(grants.isAllowed(methodName, Kind.browserConfig, ImmutableList.of(Role.ANY))).isFalse();
        assertThat(grants.isAllowed(methodName, Kind.undefined, ImmutableList.of(Role.CURATOR))).isFalse();
        assertThat(grants.isAllowed(methodName, Kind.browserConfig, ImmutableList.of(Role.CURATOR))).isTrue();

        grants = new Grants();
        methodName = "getEventObject";
        grants.addGrants(methodName, getMethodDef(methodName));
        assertThat(grants.isAllowed(methodName, Kind.undefined, ImmutableList.of(Role.ANY))).isFalse();
        assertThat(grants.isAllowed(methodName, Kind.browserConfig, ImmutableList.of(Role.ANY))).isFalse();
        assertThat(grants.isAllowed(methodName, Kind.undefined, ImmutableList.of(Role.CURATOR))).isFalse();
        assertThat(grants.isAllowed(methodName, Kind.browserConfig, ImmutableList.of(Role.CURATOR))).isTrue();
        assertThat(grants.isAllowed(methodName, Kind.crawlEntity, ImmutableList.of(Role.CURATOR))).isFalse();
        assertThat(grants.isAllowed(methodName, Kind.undefined, ImmutableList.of(Role.ADMIN))).isTrue();
        assertThat(grants.isAllowed(methodName, Kind.crawlEntity, ImmutableList.of(Role.ADMIN))).isTrue();

        grants = new Grants();
        methodName = "listEventObjects";
        grants.addGrants(methodName, getMethodDef(methodName));
        assertThat(grants.isAllowed(methodName, Kind.undefined, ImmutableList.of(Role.CURATOR))).isTrue();
        assertThat(grants.isAllowed(methodName, Kind.browserConfig, ImmutableList.of(Role.CURATOR))).isTrue();
        assertThat(grants.isAllowed(methodName, Kind.undefined, ImmutableList.of(Role.CURATOR))).isTrue();
        assertThat(grants.isAllowed(methodName, Kind.roleMapping, ImmutableList.of(Role.CURATOR))).isFalse();
        assertThat(grants.isAllowed(methodName, Kind.roleMapping, ImmutableList.of(Role.ADMIN))).isTrue();
    }

    private Method getMethodDef(String methodName) {
        for (Method m : TestService.class.getDeclaredMethods()) {
            if (m.getName().equals(methodName)) {
                return m;
            }
        }
        throw new RuntimeException("Method " + methodName + " not found");
    }

    public class TestService extends EventHandlerGrpc.EventHandlerImplBase {
        @Override
        @AllowedRoles({Role.ADMIN, Role.CURATOR, Role.READONLY, Role.ANY})
        public void countEventObjects(ListRequest request, StreamObserver<ListCountResponse> responseObserver) {
        }

        @Override
        @AllowedRoles(value = {Role.ADMIN, Role.CURATOR, Role.SYSTEM}, kind = {Kind.roleMapping, Kind.browserConfig})
        public void saveEventObject(SaveRequest request, StreamObserver<EventObject> responseObserver) {
        }

        @Override
        @Authorisations({
                @AllowedRoles(value = {Role.READONLY, Role.CURATOR, Role.OPERATOR, Role.ADMIN}, kind = {Kind.browserConfig, Kind.browserScript}),
                @AllowedRoles(value = {Role.ANY}, kind = {Kind.crawlConfig}),
                @AllowedRoles(value = {Role.ADMIN})
        })
        public void getEventObject(EventRef request, StreamObserver<EventObject> responseObserver) {
        }

        @Override
        @Authorisations({
                @AllowedRoles(value = {Role.READONLY, Role.CURATOR, Role.OPERATOR, Role.ADMIN}),
                @AllowedRoles(value = {Role.ADMIN}, kind = {Kind.roleMapping})
        })
        public void listEventObjects(ListRequest request, StreamObserver<EventObject> responseObserver) {
        }
    }
}