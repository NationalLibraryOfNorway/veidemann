import {create} from '@bufbuild/protobuf';
import {
  Collection as CollectionProto,
  CollectionSchema,
  Collection_RotationPolicy as RotationPolicyProto,
} from '../../../../api/config/v1/resources_pb';
import {SubCollection} from './subcollection.model';
import {isNumeric} from '../../func';
import type {ConfigObject} from './configobject.model';

export enum RotationPolicy {
  NONE = 0,
  HOURLY = 1,
  DAILY = 2,
  MONTHLY = 3,
  YEARLY = 4,
}

export const rotationPolicies = Object.keys(RotationPolicy).filter(p => !isNumeric(p)).map(key => RotationPolicy[key]);

export class Collection {
  collectionDedupPolicy: RotationPolicy;
  fileRotationPolicy: RotationPolicy;
  compress: boolean;
  fileSize: number;
  subCollectionsList: SubCollection[];

  constructor({
                collectionDedupPolicy = RotationPolicy.NONE,
                fileRotationPolicy = RotationPolicy.NONE,
                compress = false,
                fileSize = 0,
                subCollectionsList = []
              }: Partial<Collection> = {}) {
    this.collectionDedupPolicy = collectionDedupPolicy;
    this.fileRotationPolicy = fileRotationPolicy;
    this.compress = compress;
    this.fileSize = fileSize;
    this.subCollectionsList = subCollectionsList ? subCollectionsList.map(subCollection => new SubCollection(subCollection)) : [];
  }

  static fromProto(proto: CollectionProto): Collection {
    return new Collection({
      collectionDedupPolicy: proto.collectionDedupPolicy as unknown as RotationPolicy,
      fileRotationPolicy: proto.fileRotationPolicy as unknown as RotationPolicy,
      compress: proto.compress,
      fileSize: Number(proto.fileSize),
      subCollectionsList: proto.subCollections.map(SubCollection.fromProto)
    });
  }

  static mergeConfigs(configObjects: ConfigObject[]): Collection {
    const collection = new Collection();
    const compare = configObjects[0].collection;

    collection.collectionDedupPolicy = configObjects.every(
      config => config.collection.collectionDedupPolicy === compare.collectionDedupPolicy
    ) ? compare.collectionDedupPolicy : null;
    collection.fileRotationPolicy = configObjects.every(
      config => config.collection.fileRotationPolicy === compare.fileRotationPolicy
    ) ? compare.fileRotationPolicy : null;
    collection.compress = configObjects.every(
      config => config.collection.compress === compare.compress
    ) ? compare.compress : null;
    collection.fileSize = configObjects.every(
      config => config.collection.fileSize === compare.fileSize
    ) ? compare.fileSize : NaN;
    collection.subCollectionsList = [];

    return collection;
  }

  static toProto(collection: Collection): CollectionProto {
    return create(CollectionSchema, {
      collectionDedupPolicy: collection.collectionDedupPolicy as unknown as RotationPolicyProto,
      fileRotationPolicy: collection.fileRotationPolicy as unknown as RotationPolicyProto,
      compress: collection.compress,
      fileSize: BigInt(collection.fileSize || 0),
      subCollections: collection.subCollectionsList.map(SubCollection.toProto),
    });
  }
}
