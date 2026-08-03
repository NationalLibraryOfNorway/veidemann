import {create} from '@bufbuild/protobuf';
import {
  Collection_SubCollection as SubCollectionProto,
  Collection_SubCollectionSchema,
  Collection_SubCollectionType as SubCollectionTypeProto,
} from '../../../../api/config/v1/resources_pb';
import {isNumeric} from '../../func';


export enum SubCollectionType {
  UNDEFINED = 0,
  SCREENSHOT = 1,
  DNS = 2,
}

export const subCollectionTypes = Object.keys(SubCollectionType).filter(p => !isNumeric(p)).map(key => SubCollectionType[key]);

export class SubCollection {
  type: SubCollectionType;
  name: string;

  constructor({
                type = SubCollectionType.UNDEFINED,
                name = ''
              }: Partial<SubCollection> = {}) {
    this.type = type;
    this.name = name;
  }

  static fromProto(proto: SubCollectionProto): SubCollection {
    return new SubCollection({
      type: proto.type as unknown as SubCollectionType,
      name: proto.name
    });
  }

  static toProto(subCollection: SubCollection): SubCollectionProto {
    return create(Collection_SubCollectionSchema, {
      type: subCollection.type as unknown as SubCollectionTypeProto,
      name: subCollection.name
    });
  }
}
