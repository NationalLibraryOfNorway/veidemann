import * as jspb from 'google-protobuf'

import * as config_v1_resources_pb from '../../config/v1/resources_pb'; // proto import: "config/v1/resources.proto"
import * as commons_v1_resources_pb from '../../commons/v1/resources_pb'; // proto import: "commons/v1/resources.proto"


export class ResolveRequest extends jspb.Message {
  getHost(): string;
  setHost(value: string): ResolveRequest;

  getPort(): number;
  setPort(value: number): ResolveRequest;

  getExecutionId(): string;
  setExecutionId(value: string): ResolveRequest;

  getCollectionRef(): config_v1_resources_pb.ConfigRef | undefined;
  setCollectionRef(value?: config_v1_resources_pb.ConfigRef): ResolveRequest;
  hasCollectionRef(): boolean;
  clearCollectionRef(): ResolveRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ResolveRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ResolveRequest): ResolveRequest.AsObject;
  static serializeBinaryToWriter(message: ResolveRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ResolveRequest;
  static deserializeBinaryFromReader(message: ResolveRequest, reader: jspb.BinaryReader): ResolveRequest;
}

export namespace ResolveRequest {
  export type AsObject = {
    host: string,
    port: number,
    executionId: string,
    collectionRef?: config_v1_resources_pb.ConfigRef.AsObject,
  }
}

export class ResolveReply extends jspb.Message {
  getHost(): string;
  setHost(value: string): ResolveReply;

  getPort(): number;
  setPort(value: number): ResolveReply;

  getTextualIp(): string;
  setTextualIp(value: string): ResolveReply;

  getRawIp(): Uint8Array | string;
  getRawIp_asU8(): Uint8Array;
  getRawIp_asB64(): string;
  setRawIp(value: Uint8Array | string): ResolveReply;

  getError(): commons_v1_resources_pb.Error | undefined;
  setError(value?: commons_v1_resources_pb.Error): ResolveReply;
  hasError(): boolean;
  clearError(): ResolveReply;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ResolveReply.AsObject;
  static toObject(includeInstance: boolean, msg: ResolveReply): ResolveReply.AsObject;
  static serializeBinaryToWriter(message: ResolveReply, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ResolveReply;
  static deserializeBinaryFromReader(message: ResolveReply, reader: jspb.BinaryReader): ResolveReply;
}

export namespace ResolveReply {
  export type AsObject = {
    host: string,
    port: number,
    textualIp: string,
    rawIp: Uint8Array | string,
    error?: commons_v1_resources_pb.Error.AsObject,
  }
}

