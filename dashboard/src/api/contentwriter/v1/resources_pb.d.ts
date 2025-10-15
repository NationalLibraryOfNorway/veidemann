import * as jspb from 'google-protobuf'

import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class CrawledContent extends jspb.Message {
  getDigest(): string;
  setDigest(value: string): CrawledContent;

  getWarcId(): string;
  setWarcId(value: string): CrawledContent;

  getTargetUri(): string;
  setTargetUri(value: string): CrawledContent;

  getDate(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setDate(value?: google_protobuf_timestamp_pb.Timestamp): CrawledContent;
  hasDate(): boolean;
  clearDate(): CrawledContent;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CrawledContent.AsObject;
  static toObject(includeInstance: boolean, msg: CrawledContent): CrawledContent.AsObject;
  static serializeBinaryToWriter(message: CrawledContent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CrawledContent;
  static deserializeBinaryFromReader(message: CrawledContent, reader: jspb.BinaryReader): CrawledContent;
}

export namespace CrawledContent {
  export type AsObject = {
    digest: string,
    warcId: string,
    targetUri: string,
    date?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class StorageRef extends jspb.Message {
  getWarcId(): string;
  setWarcId(value: string): StorageRef;

  getRecordType(): RecordType;
  setRecordType(value: RecordType): StorageRef;

  getStorageRef(): string;
  setStorageRef(value: string): StorageRef;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StorageRef.AsObject;
  static toObject(includeInstance: boolean, msg: StorageRef): StorageRef.AsObject;
  static serializeBinaryToWriter(message: StorageRef, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StorageRef;
  static deserializeBinaryFromReader(message: StorageRef, reader: jspb.BinaryReader): StorageRef;
}

export namespace StorageRef {
  export type AsObject = {
    warcId: string,
    recordType: RecordType,
    storageRef: string,
  }
}

export enum RecordType { 
  WARCINFO = 0,
  RESPONSE = 1,
  RESOURCE = 2,
  REQUEST = 3,
  METADATA = 4,
  REVISIT = 5,
  CONVERSION = 6,
  CONTINUATION = 7,
}
