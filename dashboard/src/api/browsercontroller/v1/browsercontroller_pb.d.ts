import * as jspb from 'google-protobuf'

import * as config_v1_resources_pb from '../../config/v1/resources_pb'; // proto import: "config/v1/resources.proto"
import * as log_v1_resources_pb from '../../log/v1/resources_pb'; // proto import: "log/v1/resources.proto"


export class RegisterNew extends jspb.Message {
  getProxyId(): number;
  setProxyId(value: number): RegisterNew;

  getUri(): string;
  setUri(value: string): RegisterNew;

  getCrawlExecutionId(): string;
  setCrawlExecutionId(value: string): RegisterNew;

  getJobExecutionId(): string;
  setJobExecutionId(value: string): RegisterNew;

  getCollectionRef(): config_v1_resources_pb.ConfigRef | undefined;
  setCollectionRef(value?: config_v1_resources_pb.ConfigRef): RegisterNew;
  hasCollectionRef(): boolean;
  clearCollectionRef(): RegisterNew;

  getMethod(): string;
  setMethod(value: string): RegisterNew;

  getRequestId(): string;
  setRequestId(value: string): RegisterNew;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RegisterNew.AsObject;
  static toObject(includeInstance: boolean, msg: RegisterNew): RegisterNew.AsObject;
  static serializeBinaryToWriter(message: RegisterNew, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RegisterNew;
  static deserializeBinaryFromReader(message: RegisterNew, reader: jspb.BinaryReader): RegisterNew;
}

export namespace RegisterNew {
  export type AsObject = {
    proxyId: number,
    uri: string,
    crawlExecutionId: string,
    jobExecutionId: string,
    collectionRef?: config_v1_resources_pb.ConfigRef.AsObject,
    method: string,
    requestId: string,
  }
}

export class NotifyActivity extends jspb.Message {
  getActivity(): NotifyActivity.Activity;
  setActivity(value: NotifyActivity.Activity): NotifyActivity;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): NotifyActivity.AsObject;
  static toObject(includeInstance: boolean, msg: NotifyActivity): NotifyActivity.AsObject;
  static serializeBinaryToWriter(message: NotifyActivity, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): NotifyActivity;
  static deserializeBinaryFromReader(message: NotifyActivity, reader: jspb.BinaryReader): NotifyActivity;
}

export namespace NotifyActivity {
  export type AsObject = {
    activity: NotifyActivity.Activity,
  }

  export enum Activity { 
    DATA_RECEIVED = 0,
    ALL_DATA_RECEIVED = 1,
  }
}

export class Completed extends jspb.Message {
  getCrawlLog(): log_v1_resources_pb.CrawlLog | undefined;
  setCrawlLog(value?: log_v1_resources_pb.CrawlLog): Completed;
  hasCrawlLog(): boolean;
  clearCrawlLog(): Completed;

  getCached(): boolean;
  setCached(value: boolean): Completed;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Completed.AsObject;
  static toObject(includeInstance: boolean, msg: Completed): Completed.AsObject;
  static serializeBinaryToWriter(message: Completed, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Completed;
  static deserializeBinaryFromReader(message: Completed, reader: jspb.BinaryReader): Completed;
}

export namespace Completed {
  export type AsObject = {
    crawlLog?: log_v1_resources_pb.CrawlLog.AsObject,
    cached: boolean,
  }
}

export class DoRequest extends jspb.Message {
  getNew(): RegisterNew | undefined;
  setNew(value?: RegisterNew): DoRequest;
  hasNew(): boolean;
  clearNew(): DoRequest;

  getNotify(): NotifyActivity | undefined;
  setNotify(value?: NotifyActivity): DoRequest;
  hasNotify(): boolean;
  clearNotify(): DoRequest;

  getCompleted(): Completed | undefined;
  setCompleted(value?: Completed): DoRequest;
  hasCompleted(): boolean;
  clearCompleted(): DoRequest;

  getActionCase(): DoRequest.ActionCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DoRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DoRequest): DoRequest.AsObject;
  static serializeBinaryToWriter(message: DoRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DoRequest;
  static deserializeBinaryFromReader(message: DoRequest, reader: jspb.BinaryReader): DoRequest;
}

export namespace DoRequest {
  export type AsObject = {
    pb_new?: RegisterNew.AsObject,
    notify?: NotifyActivity.AsObject,
    completed?: Completed.AsObject,
  }

  export enum ActionCase { 
    ACTION_NOT_SET = 0,
    NEW = 1,
    NOTIFY = 2,
    COMPLETED = 3,
  }
}

export class NewReply extends jspb.Message {
  getCrawlExecutionId(): string;
  setCrawlExecutionId(value: string): NewReply;

  getJobExecutionId(): string;
  setJobExecutionId(value: string): NewReply;

  getCollectionRef(): config_v1_resources_pb.ConfigRef | undefined;
  setCollectionRef(value?: config_v1_resources_pb.ConfigRef): NewReply;
  hasCollectionRef(): boolean;
  clearCollectionRef(): NewReply;

  getReplacementScript(): config_v1_resources_pb.BrowserScript | undefined;
  setReplacementScript(value?: config_v1_resources_pb.BrowserScript): NewReply;
  hasReplacementScript(): boolean;
  clearReplacementScript(): NewReply;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): NewReply.AsObject;
  static toObject(includeInstance: boolean, msg: NewReply): NewReply.AsObject;
  static serializeBinaryToWriter(message: NewReply, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): NewReply;
  static deserializeBinaryFromReader(message: NewReply, reader: jspb.BinaryReader): NewReply;
}

export namespace NewReply {
  export type AsObject = {
    crawlExecutionId: string,
    jobExecutionId: string,
    collectionRef?: config_v1_resources_pb.ConfigRef.AsObject,
    replacementScript?: config_v1_resources_pb.BrowserScript.AsObject,
  }
}

export class DoReply extends jspb.Message {
  getNew(): NewReply | undefined;
  setNew(value?: NewReply): DoReply;
  hasNew(): boolean;
  clearNew(): DoReply;

  getCancel(): string;
  setCancel(value: string): DoReply;

  getActionCase(): DoReply.ActionCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DoReply.AsObject;
  static toObject(includeInstance: boolean, msg: DoReply): DoReply.AsObject;
  static serializeBinaryToWriter(message: DoReply, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DoReply;
  static deserializeBinaryFromReader(message: DoReply, reader: jspb.BinaryReader): DoReply;
}

export namespace DoReply {
  export type AsObject = {
    pb_new?: NewReply.AsObject,
    cancel: string,
  }

  export enum ActionCase { 
    ACTION_NOT_SET = 0,
    NEW = 1,
    CANCEL = 4,
  }
}

