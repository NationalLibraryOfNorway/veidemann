import {ApiError} from '../commons/api-error.model';
import {create} from '@bufbuild/protobuf';
import {CrawlLog as CrawlLogProto, CrawlLogSchema} from '../../../../api/log/v1/resources_pb';
import {fromTimestampProto, toTimestampProto} from '../../func';
import {fromRethinkTimeStamp} from '../../func/rethinkdb';
import type {Timestamp} from '../../func/rethinkdb';

export class CrawlLog {
  id: string;
  warcId: string;
  timeStamp: string;
  statusCode: number;
  size: number;
  requestedUri: string;
  responseUri: string;
  discoveryPath: string;
  referrer: string;
  contentType: string;
  fetchTimeStamp: string;
  fetchTimeMs: number;
  blockDigest: string;
  payloadDigest: string;
  storageRef: string;
  recordType: string;
  warcRefersTo: string;
  ipAddress: string;
  executionId: string;
  retries: number;
  error: ApiError;
  jobExecutionId: string;
  collectionFinalName: string;
  method: string;

  constructor({
                id = '',
                warcId = '',
                timeStamp = '',
                statusCode = 0,
                size = 0,
                requestedUri = '',
                responseUri = '',
                discoveryPath = '',
                referrer = '',
                contentType = '',
                fetchTimeStamp = '',
                fetchTimeMs = 0,
                blockDigest = '',
                payloadDigest = '',
                storageRef = '',
                recordType = '',
                warcRefersTo = '',
                ipAddress = '',
                executionId = '',
                retries = 0,
                error = new ApiError(),
                jobExecutionId = '',
                collectionFinalName = '',
                method = ''
              }: Partial<CrawlLog> = {}) {
    this.id = id || warcId;
    this.warcId = warcId;
    this.timeStamp = timeStamp;
    this.statusCode = statusCode;
    this.size = size;
    this.requestedUri = requestedUri;
    this.responseUri = responseUri;
    this.discoveryPath = discoveryPath;
    this.referrer = referrer;
    this.contentType = contentType;
    this.fetchTimeStamp = fetchTimeStamp;
    this.fetchTimeMs = fetchTimeMs;
    this.blockDigest = blockDigest;
    this.payloadDigest = payloadDigest;
    this.storageRef = storageRef;
    this.recordType = recordType;
    this.warcRefersTo = warcRefersTo;
    this.ipAddress = ipAddress;
    this.executionId = executionId;
    this.retries = retries;
    this.error = error;
    this.jobExecutionId = jobExecutionId;
    this.collectionFinalName = collectionFinalName;
    this.method = method;
  }

  /**
   * A function that transforms the results. This function is called for each member of the object.
   * If a member contains nested objects, the nested objects are transformed before the parent object is.
   * @see JSON.parse
   */
  static reviver(key: string, value: unknown): unknown {
    switch (key) {
      case 'timeStamp':
      case 'fetchTimeStamp':
        return fromRethinkTimeStamp(value as Timestamp);
      default:
        return value;
    }
  }

  static fromProto(proto: CrawlLogProto): CrawlLog {
    return new CrawlLog({
      warcId: proto.warcId,
      timeStamp: fromTimestampProto(proto.timeStamp),
      statusCode: proto.statusCode,
      size: Number(proto.size),
      requestedUri: proto.requestedUri,
      responseUri: proto.responseUri,
      discoveryPath: proto.discoveryPath,
      referrer: proto.referrer,
      contentType: proto.contentType,
      fetchTimeStamp: fromTimestampProto(proto.fetchTimeStamp),
      fetchTimeMs: Number(proto.fetchTimeMs),
      blockDigest: proto.blockDigest,
      payloadDigest: proto.payloadDigest,
      storageRef: proto.storageRef,
      recordType: proto.recordType,
      warcRefersTo: proto.warcRefersTo,
      ipAddress: proto.ipAddress,
      executionId: proto.executionId,
      retries: proto.retries,
      error: ApiError.fromProto(proto.error),
      jobExecutionId: proto.jobExecutionId,
      collectionFinalName: proto.collectionFinalName,
      method: proto.method
    });
  }

  static toProto(crawlLog: CrawlLog): CrawlLogProto {
    return create(CrawlLogSchema, {
      warcId: crawlLog.warcId,
      timeStamp: toTimestampProto(crawlLog.timeStamp),
      statusCode: crawlLog.statusCode,
      size: BigInt(crawlLog.size || 0),
      requestedUri: crawlLog.requestedUri,
      responseUri: crawlLog.responseUri,
      discoveryPath: crawlLog.discoveryPath,
      referrer: crawlLog.referrer,
      contentType: crawlLog.contentType,
      fetchTimeStamp: toTimestampProto(crawlLog.fetchTimeStamp),
      fetchTimeMs: BigInt(crawlLog.fetchTimeMs || 0),
      blockDigest: crawlLog.blockDigest,
      payloadDigest: crawlLog.payloadDigest,
      storageRef: crawlLog.storageRef,
      recordType: crawlLog.recordType,
      warcRefersTo: crawlLog.warcRefersTo,
      ipAddress: crawlLog.ipAddress,
      executionId: crawlLog.executionId,
      retries: crawlLog.retries,
      jobExecutionId: crawlLog.jobExecutionId,
      collectionFinalName: crawlLog.collectionFinalName,
      method: crawlLog.method,
    });
  }
}
