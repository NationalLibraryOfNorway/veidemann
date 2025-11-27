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

import no.nb.nna.veidemann.api.config.v1.ConfigObject;
import no.nb.nna.veidemann.api.config.v1.Kind;
import no.nb.nna.veidemann.api.config.v1.ListRequest;
import no.nb.nna.veidemann.api.config.v1.Role;
import no.nb.nna.veidemann.api.config.v1.RoleMapping;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.commons.db.DbService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Collection;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.locks.Lock;
import java.util.concurrent.locks.ReentrantLock;

public class UserRoleMapper {
    private static final Logger LOG = LoggerFactory.getLogger(UserRoleMapper.class);

    private static final ScheduledExecutorService updaterService = Executors.newSingleThreadScheduledExecutor();

    Map<String, Set<Role>> rolesByEmail = new HashMap<>();
    Map<String, Set<Role>> rolesByGroup = new HashMap<>();
    Map<String, Set<Role>> rolesByApiKey = new HashMap<>();

    Lock roleUpdateLock = new ReentrantLock();

    public UserRoleMapper() {
        updaterService.scheduleAtFixedRate(() -> {
            updateRoleMappings();
        }, 0, 60, TimeUnit.SECONDS);
    }

    public Collection<Role> getRolesForUser(String email, Collection<String> groups, Collection<Role> roles) {
        roleUpdateLock.lock();
        try {
            if (email != null && rolesByEmail.containsKey(email.toLowerCase())) {
                email = email.toLowerCase();
                roles.addAll(rolesByEmail.get(email));
            }
            if (groups != null) {
                for (String group : groups) {
                    if (rolesByGroup.containsKey(group)) {
                        roles.addAll(rolesByGroup.get(group));
                    }
                }
            }
            LOG.debug("Get roles for user. Email: {}, Groups: {}, reolved to roles: {}", email, groups, roles);
            return roles;
        } finally {
            roleUpdateLock.unlock();
        }
    }

    public Collection<Role> getRolesForApiKey(String apiKey, Collection<Role> roles) {
        roleUpdateLock.lock();
        try {
            if (apiKey != null && rolesByApiKey.containsKey(apiKey)) {
                roles.addAll(rolesByApiKey.get(apiKey));
                roles.add(Role.ANY_USER);
            }
            LOG.debug("Get roles for apiKey: {}, resolved to roles: {}", apiKey, roles);
            return roles;
        } finally {
            roleUpdateLock.unlock();
        }
    }

    private void updateRoleMappings() {
        LOG.trace("Update role mappings");
        Map<String, Set<Role>> rolesByEmailTmp = new HashMap<>();
        Map<String, Set<Role>> rolesByGroupTmp = new HashMap<>();
        Map<String, Set<Role>> rolesByApiKeyTmp = new HashMap<>();

        try {
            DbService.getInstance().getConfigAdapter()
                    .listConfigObjects(ListRequest.newBuilder().setKind(Kind.roleMapping).build())
                    .stream().forEach(rm -> addRoleMapping(rm, rolesByEmailTmp, rolesByGroupTmp, rolesByApiKeyTmp));
        } catch (DbException e) {
            LOG.warn("Could not get role mappings from DB", e);
        }

        roleUpdateLock.lock();
        try {
            rolesByEmail = rolesByEmailTmp;
            rolesByGroup = rolesByGroupTmp;
            rolesByApiKey = rolesByApiKeyTmp;
        } finally {
            roleUpdateLock.unlock();
        }
    }

    private void addRoleMapping(ConfigObject rmConfig, Map<String, Set<Role>> emailRoles, Map<String, Set<Role>> groupRoles, Map<String, Set<Role>> apiKeyRoles) {
        RoleMapping rm = rmConfig.getRoleMapping();
        switch (rmConfig.getRoleMapping().getEmailOrGroupCase()) {
            case EMAIL:
                LOG.trace("Adding role for email: {}, roles: {}", rm.getEmail().toLowerCase(), rm.getRoleList());
                rm.getRoleList().forEach(role -> addRoleToList(emailRoles, rm.getEmail().toLowerCase(), role));
                break;
            case GROUP:
                LOG.trace("Adding role for group: {}, roles: {}", rm.getGroup(), rm.getRoleList());
                rm.getRoleList().forEach(role -> addRoleToList(groupRoles, rm.getGroup(), role));
                break;
            case API_KEY:
                long validUntil = com.google.protobuf.util.Timestamps.toMillis(rm.getApiKey().getValidUntil());
                if (validUntil <= 0 || validUntil > System.currentTimeMillis()) {
                    LOG.trace("Adding role for apiKey: {}, roles: {}", rm.getApiKey().getToken(), rm.getRoleList());
                    rm.getRoleList().forEach(role -> addRoleToList(apiKeyRoles, rm.getApiKey().getToken(), role));
                } else {
                    LOG.debug("ApiKey {} with roles: {} is expired", rm.getApiKey().getToken(), rm.getRoleList());
                }
                break;
        }
    }

    private void addRoleToList(Map<String, Set<Role>> roles, String key, Role role) {
        Set<Role> roleList = roles.get(key);
        if (roleList == null) {
            roleList = new HashSet<>();
            roles.put(key, roleList);
        }
        roleList.add(role);
    }
}
