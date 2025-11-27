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

import java.util.Collection;

public class ApiKeyRoleMapperFromConfig implements ApiKeyRoleMapper {
    private static final Logger LOG = LoggerFactory.getLogger(ApiKeyRoleMapperFromConfig.class);
    private final UserRoleMapper userRoleMapper;

    public ApiKeyRoleMapperFromConfig(UserRoleMapper userRoleMapper) {
        this.userRoleMapper = userRoleMapper;
    }

    @Override
    public Collection<Role> getRolesForAPiKey(String apiKey, Collection<Role> roles) {
        return userRoleMapper.getRolesForApiKey(apiKey, roles);
    }
}
