import {ConfigObject} from './configobject.model';
import {create} from '@bufbuild/protobuf';
import {
  ApiKey as ApiKeyProto,
  ApiKeySchema,
  RoleMapping as RoleMappingProto,
  RoleMappingSchema,
} from '../../../../api/config/v1/resources_pb';
import {Role} from './role.model';
import {fromTimestampProto, toTimestampProto} from '../../func';

export class RoleMapping {
  apiKey?: ApiKey;
  email?: string;
  group?: string;
  roleList: Role[];

  constructor({
                apiKey = new ApiKey(),
                email = '',
                group = '',
                roleList = []
              }: Partial<RoleMapping> = {}) {
    this.apiKey = apiKey;
    this.email = email;
    this.group = group;
    this.roleList = roleList ? [...roleList] : [];
  }

  static fromProto(proto: RoleMappingProto): RoleMapping {
    return new RoleMapping({
      apiKey: proto.emailOrGroup.case === 'apiKey' ? ApiKey.fromProto(proto.emailOrGroup.value) : undefined,
      email: proto.emailOrGroup.case === 'email' ? proto.emailOrGroup.value : '',
      group: proto.emailOrGroup.case === 'group' ? proto.emailOrGroup.value : '',
      roleList: proto.role
    });
  }

  static toProto(roleMapping: RoleMapping): RoleMappingProto {
    const emailOrGroup = roleMapping.group
      ? {case: 'group' as const, value: roleMapping.group}
      : roleMapping.email
        ? {case: 'email' as const, value: roleMapping.email}
        : roleMapping.apiKey
          ? {case: 'apiKey' as const, value: ApiKey.toProto(roleMapping.apiKey)}
          : {case: undefined, value: undefined};
    return create(RoleMappingSchema, {emailOrGroup, role: roleMapping.roleList});
  }

  static mergeConfigs(configObjects: ConfigObject[]): RoleMapping {
    const roleMapping = new RoleMapping();

    const commonRoles = getCommonRoles(configObjects);

    for (const role of commonRoles) {
      const gotRole = configObjects.every((cfg) => cfg.roleMapping.roleList.indexOf(role) !== -1);
      if (gotRole) {
        roleMapping.roleList.push(role);
      }
    }
    return roleMapping;
  }
}

function getCommonRoles(configObjects: ConfigObject[]): Role[] {
  return Array.from(new Set(
    configObjects
      .map(configObject => configObject.roleMapping.roleList)
      .reduce((acc, curr) => acc.concat(curr), [])
  ));
}

export class ApiKey {
  token: string;
  validUntil?: string;

  constructor({
                token = '',
                validUntil = '',
              }: Partial<ApiKey> = {}) {
    this.token = token;
    this.validUntil = validUntil;
  }

  static fromProto(proto: ApiKeyProto): ApiKey {
    return new ApiKey({
      token: proto.token,
      validUntil: fromTimestampProto(proto.validUntil),
    });
  }

  static toProto(apiKey: ApiKey): ApiKeyProto {
    return create(ApiKeySchema, {token: apiKey.token, validUntil: toTimestampProto(apiKey.validUntil)});
  }
}
