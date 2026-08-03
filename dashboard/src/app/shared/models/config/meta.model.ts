import {create} from '@bufbuild/protobuf';
import {Meta as MetaProto, MetaSchema} from '../../../../api/config/v1/resources_pb';
import {fromTimestampProto} from '../../func';
import {Label} from './label.model';
import {Annotation} from './annotation.model';

export class Meta {
  name: string;
  description: string;
  created: string;
  createdBy: string;
  lastModified: string;
  lastModifiedBy: string;
  labelList: Label[];
  annotationList: Annotation[];

  constructor({
                labelList = [],
                annotationList = [],
                description = '',
                name = '',
                created = '',
                createdBy = '',
                lastModified = '',
                lastModifiedBy = '',
              }: Partial<Meta> = {}) {
    this.name = name;
    this.description = description;
    this.created = created;
    this.createdBy = createdBy;
    this.lastModified = lastModified;
    this.lastModifiedBy = lastModifiedBy;
    this.labelList = labelList ? labelList.map(label => new Label(label)) : [];
    this.annotationList = annotationList ? annotationList.map(annotation => new Annotation(annotation)) : [];
  }

  static fromProto(proto: MetaProto): Meta {
    return new Meta({
      name: proto.name,
      description: proto.description,
      created: fromTimestampProto(proto.created),
      createdBy: proto.createdBy,
      lastModified: fromTimestampProto(proto.lastModified),
      lastModifiedBy: proto.lastModifiedBy,
      labelList: proto.label.map(Label.fromProto),
      annotationList: proto.annotation.map(Annotation.fromProto)
    });
  }

  static toProto(meta: Meta): MetaProto {
    return create(MetaSchema, {
      name: meta.name,
      description: meta.description,
      label: meta.labelList.map(Label.toProto),
      annotation: meta.annotationList.map(Annotation.toProto),
    });
  }
}
