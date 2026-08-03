import {create} from '@bufbuild/protobuf';
import {ConfigRef as ConfigRefProto, ConfigRefSchema} from '../../../../api/config/v1/resources_pb';
import {Kind} from './kind.model';

export class ConfigRef {
  kind: Kind;
  id: string;

  constructor({
                kind = Kind.UNDEFINED,
                id = ''
              }: Partial<ConfigRef> = {}) {
    this.id = id;
    this.kind = kind;
  }

  static fromProto(proto: ConfigRefProto): ConfigRef {
    return new ConfigRef({
      id: proto.id,
      kind: proto.kind.valueOf()
    });
  }

  static toProto(configRef: ConfigRef): ConfigRefProto {
    if (!configRef) {
      return undefined;
    }
    return create(ConfigRefSchema, {id: configRef.id, kind: configRef.kind.valueOf()});
  }
}
