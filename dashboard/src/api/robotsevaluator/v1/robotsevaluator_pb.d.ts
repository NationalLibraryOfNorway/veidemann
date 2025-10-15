import * as jspb from 'google-protobuf'

import * as config_v1_resources_pb from '../../config/v1/resources_pb'; // proto import: "config/v1/resources.proto"


export class IsAllowedRequest extends jspb.Message {
  getJobExecutionId(): string;
  setJobExecutionId(value: string): IsAllowedRequest;

  getExecutionId(): string;
  setExecutionId(value: string): IsAllowedRequest;

  getUri(): string;
  setUri(value: string): IsAllowedRequest;

  getUserAgent(): string;
  setUserAgent(value: string): IsAllowedRequest;

  getPoliteness(): config_v1_resources_pb.ConfigObject | undefined;
  setPoliteness(value?: config_v1_resources_pb.ConfigObject): IsAllowedRequest;
  hasPoliteness(): boolean;
  clearPoliteness(): IsAllowedRequest;

  getCollectionRef(): config_v1_resources_pb.ConfigRef | undefined;
  setCollectionRef(value?: config_v1_resources_pb.ConfigRef): IsAllowedRequest;
  hasCollectionRef(): boolean;
  clearCollectionRef(): IsAllowedRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): IsAllowedRequest.AsObject;
  static toObject(includeInstance: boolean, msg: IsAllowedRequest): IsAllowedRequest.AsObject;
  static serializeBinaryToWriter(message: IsAllowedRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): IsAllowedRequest;
  static deserializeBinaryFromReader(message: IsAllowedRequest, reader: jspb.BinaryReader): IsAllowedRequest;
}

export namespace IsAllowedRequest {
  export type AsObject = {
    jobExecutionId: string,
    executionId: string,
    uri: string,
    userAgent: string,
    politeness?: config_v1_resources_pb.ConfigObject.AsObject,
    collectionRef?: config_v1_resources_pb.ConfigRef.AsObject,
  }
}

export class IsAllowedReply extends jspb.Message {
  getIsAllowed(): boolean;
  setIsAllowed(value: boolean): IsAllowedReply;

  getCrawlDelay(): number;
  setCrawlDelay(value: number): IsAllowedReply;

  getCacheDelay(): number;
  setCacheDelay(value: number): IsAllowedReply;

  getSitemapList(): Array<string>;
  setSitemapList(value: Array<string>): IsAllowedReply;
  clearSitemapList(): IsAllowedReply;
  addSitemap(value: string, index?: number): IsAllowedReply;

  getOtherFieldsList(): Array<IsAllowedReply.OtherField>;
  setOtherFieldsList(value: Array<IsAllowedReply.OtherField>): IsAllowedReply;
  clearOtherFieldsList(): IsAllowedReply;
  addOtherFields(value?: IsAllowedReply.OtherField, index?: number): IsAllowedReply.OtherField;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): IsAllowedReply.AsObject;
  static toObject(includeInstance: boolean, msg: IsAllowedReply): IsAllowedReply.AsObject;
  static serializeBinaryToWriter(message: IsAllowedReply, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): IsAllowedReply;
  static deserializeBinaryFromReader(message: IsAllowedReply, reader: jspb.BinaryReader): IsAllowedReply;
}

export namespace IsAllowedReply {
  export type AsObject = {
    isAllowed: boolean,
    crawlDelay: number,
    cacheDelay: number,
    sitemapList: Array<string>,
    otherFieldsList: Array<IsAllowedReply.OtherField.AsObject>,
  }

  export class OtherField extends jspb.Message {
    getName(): string;
    setName(value: string): OtherField;

    getValue(): string;
    setValue(value: string): OtherField;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): OtherField.AsObject;
    static toObject(includeInstance: boolean, msg: OtherField): OtherField.AsObject;
    static serializeBinaryToWriter(message: OtherField, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): OtherField;
    static deserializeBinaryFromReader(message: OtherField, reader: jspb.BinaryReader): OtherField;
  }

  export namespace OtherField {
    export type AsObject = {
      name: string,
      value: string,
    }
  }

}

