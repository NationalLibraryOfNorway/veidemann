import * as jspb from 'google-protobuf'

import * as config_v1_resources_pb from '../../config/v1/resources_pb'; // proto import: "config/v1/resources.proto"
import * as contentwriter_v1_resources_pb from '../../contentwriter/v1/resources_pb'; // proto import: "contentwriter/v1/resources.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class Data extends jspb.Message {
  getRecordNum(): number;
  setRecordNum(value: number): Data;

  getData(): Uint8Array | string;
  getData_asU8(): Uint8Array;
  getData_asB64(): string;
  setData(value: Uint8Array | string): Data;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Data.AsObject;
  static toObject(includeInstance: boolean, msg: Data): Data.AsObject;
  static serializeBinaryToWriter(message: Data, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Data;
  static deserializeBinaryFromReader(message: Data, reader: jspb.BinaryReader): Data;
}

export namespace Data {
  export type AsObject = {
    recordNum: number,
    data: Uint8Array | string,
  }
}

export class WriteRequestMeta extends jspb.Message {
  getExecutionId(): string;
  setExecutionId(value: string): WriteRequestMeta;

  getTargetUri(): string;
  setTargetUri(value: string): WriteRequestMeta;

  getRecordMetaMap(): jspb.Map<number, WriteRequestMeta.RecordMeta>;
  clearRecordMetaMap(): WriteRequestMeta;

  getFetchTimeStamp(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setFetchTimeStamp(value?: google_protobuf_timestamp_pb.Timestamp): WriteRequestMeta;
  hasFetchTimeStamp(): boolean;
  clearFetchTimeStamp(): WriteRequestMeta;

  getIpAddress(): string;
  setIpAddress(value: string): WriteRequestMeta;

  getCollectionRef(): config_v1_resources_pb.ConfigRef | undefined;
  setCollectionRef(value?: config_v1_resources_pb.ConfigRef): WriteRequestMeta;
  hasCollectionRef(): boolean;
  clearCollectionRef(): WriteRequestMeta;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): WriteRequestMeta.AsObject;
  static toObject(includeInstance: boolean, msg: WriteRequestMeta): WriteRequestMeta.AsObject;
  static serializeBinaryToWriter(message: WriteRequestMeta, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): WriteRequestMeta;
  static deserializeBinaryFromReader(message: WriteRequestMeta, reader: jspb.BinaryReader): WriteRequestMeta;
}

export namespace WriteRequestMeta {
  export type AsObject = {
    executionId: string,
    targetUri: string,
    recordMetaMap: Array<[number, WriteRequestMeta.RecordMeta.AsObject]>,
    fetchTimeStamp?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    ipAddress: string,
    collectionRef?: config_v1_resources_pb.ConfigRef.AsObject,
  }

  export class RecordMeta extends jspb.Message {
    getRecordNum(): number;
    setRecordNum(value: number): RecordMeta;

    getType(): contentwriter_v1_resources_pb.RecordType;
    setType(value: contentwriter_v1_resources_pb.RecordType): RecordMeta;

    getRecordContentType(): string;
    setRecordContentType(value: string): RecordMeta;

    getBlockDigest(): string;
    setBlockDigest(value: string): RecordMeta;

    getPayloadDigest(): string;
    setPayloadDigest(value: string): RecordMeta;

    getSize(): number;
    setSize(value: number): RecordMeta;

    getSubCollection(): config_v1_resources_pb.Collection.SubCollectionType;
    setSubCollection(value: config_v1_resources_pb.Collection.SubCollectionType): RecordMeta;

    getWarcConcurrentToList(): Array<string>;
    setWarcConcurrentToList(value: Array<string>): RecordMeta;
    clearWarcConcurrentToList(): RecordMeta;
    addWarcConcurrentTo(value: string, index?: number): RecordMeta;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): RecordMeta.AsObject;
    static toObject(includeInstance: boolean, msg: RecordMeta): RecordMeta.AsObject;
    static serializeBinaryToWriter(message: RecordMeta, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): RecordMeta;
    static deserializeBinaryFromReader(message: RecordMeta, reader: jspb.BinaryReader): RecordMeta;
  }

  export namespace RecordMeta {
    export type AsObject = {
      recordNum: number,
      type: contentwriter_v1_resources_pb.RecordType,
      recordContentType: string,
      blockDigest: string,
      payloadDigest: string,
      size: number,
      subCollection: config_v1_resources_pb.Collection.SubCollectionType,
      warcConcurrentToList: Array<string>,
    }
  }

}

export class WriteRequest extends jspb.Message {
  getMeta(): WriteRequestMeta | undefined;
  setMeta(value?: WriteRequestMeta): WriteRequest;
  hasMeta(): boolean;
  clearMeta(): WriteRequest;

  getProtocolHeader(): Data | undefined;
  setProtocolHeader(value?: Data): WriteRequest;
  hasProtocolHeader(): boolean;
  clearProtocolHeader(): WriteRequest;

  getPayload(): Data | undefined;
  setPayload(value?: Data): WriteRequest;
  hasPayload(): boolean;
  clearPayload(): WriteRequest;

  getCancel(): string;
  setCancel(value: string): WriteRequest;

  getValueCase(): WriteRequest.ValueCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): WriteRequest.AsObject;
  static toObject(includeInstance: boolean, msg: WriteRequest): WriteRequest.AsObject;
  static serializeBinaryToWriter(message: WriteRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): WriteRequest;
  static deserializeBinaryFromReader(message: WriteRequest, reader: jspb.BinaryReader): WriteRequest;
}

export namespace WriteRequest {
  export type AsObject = {
    meta?: WriteRequestMeta.AsObject,
    protocolHeader?: Data.AsObject,
    payload?: Data.AsObject,
    cancel: string,
  }

  export enum ValueCase { 
    VALUE_NOT_SET = 0,
    META = 1,
    PROTOCOL_HEADER = 2,
    PAYLOAD = 3,
    CANCEL = 4,
  }
}

export class WriteResponseMeta extends jspb.Message {
  getRecordMetaMap(): jspb.Map<number, WriteResponseMeta.RecordMeta>;
  clearRecordMetaMap(): WriteResponseMeta;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): WriteResponseMeta.AsObject;
  static toObject(includeInstance: boolean, msg: WriteResponseMeta): WriteResponseMeta.AsObject;
  static serializeBinaryToWriter(message: WriteResponseMeta, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): WriteResponseMeta;
  static deserializeBinaryFromReader(message: WriteResponseMeta, reader: jspb.BinaryReader): WriteResponseMeta;
}

export namespace WriteResponseMeta {
  export type AsObject = {
    recordMetaMap: Array<[number, WriteResponseMeta.RecordMeta.AsObject]>,
  }

  export class RecordMeta extends jspb.Message {
    getRecordNum(): number;
    setRecordNum(value: number): RecordMeta;

    getType(): contentwriter_v1_resources_pb.RecordType;
    setType(value: contentwriter_v1_resources_pb.RecordType): RecordMeta;

    getWarcId(): string;
    setWarcId(value: string): RecordMeta;

    getStorageRef(): string;
    setStorageRef(value: string): RecordMeta;

    getBlockDigest(): string;
    setBlockDigest(value: string): RecordMeta;

    getPayloadDigest(): string;
    setPayloadDigest(value: string): RecordMeta;

    getRevisitReferenceId(): string;
    setRevisitReferenceId(value: string): RecordMeta;

    getCollectionFinalName(): string;
    setCollectionFinalName(value: string): RecordMeta;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): RecordMeta.AsObject;
    static toObject(includeInstance: boolean, msg: RecordMeta): RecordMeta.AsObject;
    static serializeBinaryToWriter(message: RecordMeta, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): RecordMeta;
    static deserializeBinaryFromReader(message: RecordMeta, reader: jspb.BinaryReader): RecordMeta;
  }

  export namespace RecordMeta {
    export type AsObject = {
      recordNum: number,
      type: contentwriter_v1_resources_pb.RecordType,
      warcId: string,
      storageRef: string,
      blockDigest: string,
      payloadDigest: string,
      revisitReferenceId: string,
      collectionFinalName: string,
    }
  }

}

export class WriteReply extends jspb.Message {
  getMeta(): WriteResponseMeta | undefined;
  setMeta(value?: WriteResponseMeta): WriteReply;
  hasMeta(): boolean;
  clearMeta(): WriteReply;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): WriteReply.AsObject;
  static toObject(includeInstance: boolean, msg: WriteReply): WriteReply.AsObject;
  static serializeBinaryToWriter(message: WriteReply, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): WriteReply;
  static deserializeBinaryFromReader(message: WriteReply, reader: jspb.BinaryReader): WriteReply;
}

export namespace WriteReply {
  export type AsObject = {
    meta?: WriteResponseMeta.AsObject,
  }
}

