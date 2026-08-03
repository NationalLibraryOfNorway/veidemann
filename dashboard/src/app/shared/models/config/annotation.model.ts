import {create} from '@bufbuild/protobuf';
import {Annotation as AnnotationProto, AnnotationSchema} from '../../../../api/config/v1/resources_pb';

export class Annotation {
  key: string;
  value: string;

  constructor({key = '', value = ''}: Partial<Annotation> = {}) {
    this.key = key;
    this.value = value;
  }

  static fromProto(proto: AnnotationProto) {
    return new Annotation({
      key: proto.key,
      value: proto.value
    });
  }

  static toProto(annotation: Annotation): AnnotationProto {
    return create(AnnotationSchema, {key: annotation.key, value: annotation.value});
  }
}
