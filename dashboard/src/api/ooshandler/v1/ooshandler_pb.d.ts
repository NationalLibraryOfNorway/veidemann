import * as jspb from 'google-protobuf'

import * as frontier_v1_resources_pb from '../../frontier/v1/resources_pb'; // proto import: "frontier/v1/resources.proto"
import * as google_protobuf_empty_pb from 'google-protobuf/google/protobuf/empty_pb'; // proto import: "google/protobuf/empty.proto"


export class SubmitUriRequest extends jspb.Message {
  getUri(): frontier_v1_resources_pb.QueuedUri | undefined;
  setUri(value?: frontier_v1_resources_pb.QueuedUri): SubmitUriRequest;
  hasUri(): boolean;
  clearUri(): SubmitUriRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SubmitUriRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SubmitUriRequest): SubmitUriRequest.AsObject;
  static serializeBinaryToWriter(message: SubmitUriRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SubmitUriRequest;
  static deserializeBinaryFromReader(message: SubmitUriRequest, reader: jspb.BinaryReader): SubmitUriRequest;
}

export namespace SubmitUriRequest {
  export type AsObject = {
    uri?: frontier_v1_resources_pb.QueuedUri.AsObject,
  }
}

