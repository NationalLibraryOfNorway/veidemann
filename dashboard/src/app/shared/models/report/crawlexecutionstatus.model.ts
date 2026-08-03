import {create} from '@bufbuild/protobuf';
import {
  CrawlExecutionStatus as CrawlExecutionStatusProto,
  CrawlExecutionStatusSchema,
} from '../../../../api/frontier/v1/resources_pb';
import {fromTimestampProto, isNumeric, toTimestampProto} from '../../func';
import {ApiError} from '../commons/api-error.model';
import {fromRethinkTimeStamp} from '../../func/rethinkdb';
import type {Timestamp} from '../../func/rethinkdb';

export enum CrawlExecutionState {
  UNDEFINED = 0,
  CREATED = 1,
  FETCHING = 2,
  SLEEPING = 3,
  FINISHED = 4,
  ABORTED_TIMEOUT = 5,
  ABORTED_SIZE = 6,
  ABORTED_MANUAL = 7,
  FAILED = 8,
  DIED = 9,
}

export const crawlExecutionStates: CrawlExecutionState[] =
  Object.keys(CrawlExecutionState).filter(p => !isNumeric(p)).map(state => CrawlExecutionState[state]);

export class CrawlExecutionStatus {
  static DONE_STATES = [
    CrawlExecutionState.ABORTED_MANUAL,
    CrawlExecutionState.ABORTED_SIZE,
    CrawlExecutionState.ABORTED_TIMEOUT,
    CrawlExecutionState.FAILED,
    CrawlExecutionState.FINISHED,
  ];

  id: string;
  state: CrawlExecutionState;
  jobId: string;
  seedId: string;
  startTime: string;
  endTime: string;
  documentsCrawled: number;
  bytesCrawled: number;
  urisCrawled: number;
  documentsFailed: number;
  documentsOutOfScope: number;
  documentsRetried: number;
  documentsDenied: number;
  lastChangeTime: string;
  createdTime: string;
  currentUriIdList: string[];
  jobExecutionId: string;
  error: ApiError;
  desiredState: CrawlExecutionState;

  constructor({
                id = '',
                state = CrawlExecutionState.UNDEFINED,
                jobId = '',
                seedId = '',
                startTime = '',
                endTime = '',
                documentsCrawled = 0,
                bytesCrawled = 0,
                urisCrawled = 0,
                documentsFailed = 0,
                documentsOutOfScope = 0,
                documentsRetried = 0,
                documentsDenied = 0,
                lastChangeTime = '',
                createdTime = '',
                currentUriIdList = [],
                jobExecutionId = '',
                error = new ApiError(),
                desiredState = CrawlExecutionState.UNDEFINED,
              }: Partial<CrawlExecutionStatus> = {}) {
    this.id = id;
    this.jobId = jobId;
    this.seedId = seedId;
    this.state = state;
    this.startTime = startTime;
    this.endTime = endTime;
    this.documentsCrawled = documentsCrawled;
    this.bytesCrawled = bytesCrawled;
    this.urisCrawled = urisCrawled;
    this.documentsFailed = documentsFailed;
    this.documentsOutOfScope = documentsOutOfScope;
    this.documentsRetried = documentsRetried;
    this.documentsDenied = documentsDenied;
    this.lastChangeTime = lastChangeTime;
    this.createdTime = createdTime;
    this.currentUriIdList = currentUriIdList;
    this.jobExecutionId = jobExecutionId;
    this.error = error;
    this.desiredState = desiredState;
  }

  /**
   * A function that transforms the results. This function is called for each member of the object.
   * If a member contains nested objects, the nested objects are transformed before the parent object is.
   * @see JSON.parse
   */
  static reviver(key: string, value: unknown): unknown {
    switch (key) {
      case 'state':
        return CrawlExecutionState[value as keyof typeof CrawlExecutionState];
      case 'startTime':
      case 'endTime':
      case 'lastChangeTime':
      case 'createdTime':
        return fromRethinkTimeStamp(value as Timestamp);
      default:
        return value;
    }
  }

  static fromProto(proto: CrawlExecutionStatusProto): CrawlExecutionStatus {
    const crawlExecutionStatus = new CrawlExecutionStatus({
      id: proto.id,
      jobId: proto.jobId,
      seedId: proto.seedId,
      state: proto.state as unknown as CrawlExecutionState,
      startTime: fromTimestampProto(proto.startTime),
      endTime: fromTimestampProto(proto.endTime),
      documentsCrawled: Number(proto.documentsCrawled),
      bytesCrawled: Number(proto.bytesCrawled),
      urisCrawled: Number(proto.urisCrawled),
      documentsFailed: Number(proto.documentsFailed),
      documentsOutOfScope: Number(proto.documentsOutOfScope),
      documentsRetried: Number(proto.documentsRetried),
      documentsDenied: Number(proto.documentsDenied),
      lastChangeTime: fromTimestampProto(proto.lastChangeTime),
      createdTime: fromTimestampProto(proto.createdTime),
      currentUriIdList: proto.currentUriId,
      jobExecutionId: proto.jobExecutionId,
      error: ApiError.fromProto(proto.error),
      desiredState: proto.desiredState as unknown as CrawlExecutionState
    });
    return crawlExecutionStatus;
  }

  static toProto(crawlExecutionStatus: CrawlExecutionStatus): CrawlExecutionStatusProto {
    return create(CrawlExecutionStatusSchema, {
      id: crawlExecutionStatus.id,
      jobId: crawlExecutionStatus.jobId,
      seedId: crawlExecutionStatus.seedId,
      state: crawlExecutionStatus.state.valueOf(),
      startTime: toTimestampProto(crawlExecutionStatus.startTime),
      endTime: toTimestampProto(crawlExecutionStatus.endTime),
      documentsCrawled: BigInt(crawlExecutionStatus.documentsCrawled || 0),
      bytesCrawled: BigInt(crawlExecutionStatus.bytesCrawled || 0),
      urisCrawled: BigInt(crawlExecutionStatus.urisCrawled || 0),
      documentsFailed: BigInt(crawlExecutionStatus.documentsFailed || 0),
      documentsOutOfScope: BigInt(crawlExecutionStatus.documentsOutOfScope || 0),
      documentsRetried: BigInt(crawlExecutionStatus.documentsRetried || 0),
      documentsDenied: BigInt(crawlExecutionStatus.documentsDenied || 0),
      lastChangeTime: toTimestampProto(crawlExecutionStatus.lastChangeTime),
      createdTime: toTimestampProto(crawlExecutionStatus.createdTime),
      currentUriId: crawlExecutionStatus.currentUriIdList,
      jobExecutionId: crawlExecutionStatus.jobExecutionId,
      desiredState: crawlExecutionStatus.desiredState.valueOf(),
    });
  }
}
