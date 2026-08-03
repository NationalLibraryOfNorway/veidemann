import {create} from '@bufbuild/protobuf';
import {Label as LabelProto, LabelSchema} from '../../../../api/config/v1/resources_pb';


export class Label {
  key: string;
  value: string;

  constructor({key = '', value = ''}: Partial<Label> = {}) {
    this.key = key;
    this.value = value;
  }

  static fromProto(proto: LabelProto): Label {
    return new Label({
      key: proto.key,
      value: proto.value
    });
  }

  static toProto(label: Label): LabelProto {
    return create(LabelSchema, {key: label.key, value: label.value});
  }
}
