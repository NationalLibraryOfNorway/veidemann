import * as jspb from 'google-protobuf'

import * as frontier_v1_resources_pb from '../../frontier/v1/resources_pb'; // proto import: "frontier/v1/resources.proto"
import * as commons_v1_resources_pb from '../../commons/v1/resources_pb'; // proto import: "commons/v1/resources.proto"


export class ScopeCheckRequest extends jspb.Message {
  getQueuedUri(): frontier_v1_resources_pb.QueuedUri | undefined;
  setQueuedUri(value?: frontier_v1_resources_pb.QueuedUri): ScopeCheckRequest;
  hasQueuedUri(): boolean;
  clearQueuedUri(): ScopeCheckRequest;

  getScopeScriptName(): string;
  setScopeScriptName(value: string): ScopeCheckRequest;

  getScopeScript(): string;
  setScopeScript(value: string): ScopeCheckRequest;

  getDebug(): boolean;
  setDebug(value: boolean): ScopeCheckRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ScopeCheckRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ScopeCheckRequest): ScopeCheckRequest.AsObject;
  static serializeBinaryToWriter(message: ScopeCheckRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ScopeCheckRequest;
  static deserializeBinaryFromReader(message: ScopeCheckRequest, reader: jspb.BinaryReader): ScopeCheckRequest;
}

export namespace ScopeCheckRequest {
  export type AsObject = {
    queuedUri?: frontier_v1_resources_pb.QueuedUri.AsObject,
    scopeScriptName: string,
    scopeScript: string,
    debug: boolean,
  }
}

export class ScopeCheckResponse extends jspb.Message {
  getEvaluation(): ScopeCheckResponse.Evaluation;
  setEvaluation(value: ScopeCheckResponse.Evaluation): ScopeCheckResponse;

  getExcludeReason(): number;
  setExcludeReason(value: number): ScopeCheckResponse;

  getIncludeCheckUri(): commons_v1_resources_pb.ParsedUri | undefined;
  setIncludeCheckUri(value?: commons_v1_resources_pb.ParsedUri): ScopeCheckResponse;
  hasIncludeCheckUri(): boolean;
  clearIncludeCheckUri(): ScopeCheckResponse;

  getError(): commons_v1_resources_pb.Error | undefined;
  setError(value?: commons_v1_resources_pb.Error): ScopeCheckResponse;
  hasError(): boolean;
  clearError(): ScopeCheckResponse;

  getConsole(): string;
  setConsole(value: string): ScopeCheckResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ScopeCheckResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ScopeCheckResponse): ScopeCheckResponse.AsObject;
  static serializeBinaryToWriter(message: ScopeCheckResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ScopeCheckResponse;
  static deserializeBinaryFromReader(message: ScopeCheckResponse, reader: jspb.BinaryReader): ScopeCheckResponse;
}

export namespace ScopeCheckResponse {
  export type AsObject = {
    evaluation: ScopeCheckResponse.Evaluation,
    excludeReason: number,
    includeCheckUri?: commons_v1_resources_pb.ParsedUri.AsObject,
    error?: commons_v1_resources_pb.Error.AsObject,
    console: string,
  }

  export enum Evaluation { 
    INCLUDE = 0,
    EXCLUDE = 1,
  }
}

