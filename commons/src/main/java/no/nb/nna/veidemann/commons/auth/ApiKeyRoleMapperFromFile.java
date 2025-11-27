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

import no.nb.nna.veidemann.api.config.v1.Role;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.BufferedReader;
import java.io.FileReader;
import java.io.IOException;
import java.util.Arrays;
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

public class ApiKeyRoleMapperFromFile implements ApiKeyRoleMapper {
    private static final Logger LOG = LoggerFactory.getLogger(ApiKeyRoleMapperFromFile.class);

    private static final ScheduledExecutorService updaterService = Executors.newSingleThreadScheduledExecutor();

    Map<String, Set<Role>> rolesByApiKey = new HashMap<>();

    Lock roleUpdateLock = new ReentrantLock();

    private final String fileName;

    public ApiKeyRoleMapperFromFile(String fileName) {
        this.fileName = fileName;

        updaterService.scheduleAtFixedRate(() -> {
            updateRoleMappings();
        }, 0, 60, TimeUnit.SECONDS);
    }

    @Override
    public Collection<Role> getRolesForAPiKey(String apiKey, Collection<Role> roles) {
        roleUpdateLock.lock();
        try {
            if (apiKey != null && rolesByApiKey.containsKey(apiKey)) {
                roles.addAll(rolesByApiKey.get(apiKey));
            }
            LOG.debug("Get roles for client. ApiKey: {}, resolved to roles: {}", apiKey, roles);
            return roles;
        } finally {
            roleUpdateLock.unlock();
        }
    }

    private void updateRoleMappings() {
        LOG.trace("Update role mappings");
        Map<String, Set<Role>> rolesByApiKeyTmp = new HashMap<>();

        try (FileReader r = new FileReader(fileName)) {
            BufferedReader b = new BufferedReader(r);

            String line = b.readLine();
            while (line != null) {
                line = line.trim();
                if (!line.isEmpty() && !line.startsWith("#")) {
                    addRoleMapping(line, rolesByApiKeyTmp);
                }
                line = b.readLine();
            }
        } catch (IOException e) {
            LOG.warn("Could not get role mappings from file", e);
        }

        roleUpdateLock.lock();
        try {
            rolesByApiKey = rolesByApiKeyTmp;
        } finally {
            roleUpdateLock.unlock();
        }
    }

    private void addRoleMapping(String line, Map<String, Set<Role>> apiKeyRoles) {
        String[] tokens = line.split("\\s+");
        if (tokens.length > 1) {
            String apiKey = tokens[0];
            addRoleToList(apiKeyRoles, apiKey, Role.ANY_USER);
            for (int i = 1; i < tokens.length; i++) {
                try {
                    Role r = Role.valueOf(tokens[i]);
                    LOG.trace("Adding role for api key: {}, role: {}", apiKey, r);
                    addRoleToList(apiKeyRoles, apiKey, r);
                } catch (Exception e) {
                    LOG.warn("Could not parse role from '{}'", tokens[i], e);
                }
            }
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
