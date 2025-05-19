import * as jspb from 'google-protobuf'

import * as commons_v1_resources_pb from '../../commons/v1/resources_pb'; // proto import: "commons/v1/resources.proto"


export class CanonicalizeRequest extends jspb.Message {
  getUri(): string;
  setUri(value: string): CanonicalizeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CanonicalizeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CanonicalizeRequest): CanonicalizeRequest.AsObject;
  static serializeBinaryToWriter(message: CanonicalizeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CanonicalizeRequest;
  static deserializeBinaryFromReader(message: CanonicalizeRequest, reader: jspb.BinaryReader): CanonicalizeRequest;
}

export namespace CanonicalizeRequest {
  export type AsObject = {
    uri: string,
  }
}

export class CanonicalizeResponse extends jspb.Message {
  getUri(): commons_v1_resources_pb.ParsedUri | undefined;
  setUri(value?: commons_v1_resources_pb.ParsedUri): CanonicalizeResponse;
  hasUri(): boolean;
  clearUri(): CanonicalizeResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CanonicalizeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: CanonicalizeResponse): CanonicalizeResponse.AsObject;
  static serializeBinaryToWriter(message: CanonicalizeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CanonicalizeResponse;
  static deserializeBinaryFromReader(message: CanonicalizeResponse, reader: jspb.BinaryReader): CanonicalizeResponse;
}

export namespace CanonicalizeResponse {
  export type AsObject = {
    uri?: commons_v1_resources_pb.ParsedUri.AsObject,
  }
}

