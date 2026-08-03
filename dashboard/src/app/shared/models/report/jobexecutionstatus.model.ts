import {create} from '@bufbuild/protobuf';
import {
  JobExecutionStatus as JobExecutionStatusProto,
  JobExecutionStatusSchema,
} from '../../../../api/frontier/v1/resources_pb';
import {fromTimestampProto, isNumeric} from '../../func';
import { ApiError } from '../commons';

export enum JobExecutionState {
  UNDEFINED = 0,
  CREATED = 1,
  RUNNING = 2,
  FINISHED = 3,
  ABORTED_MANUAL = 4,
  FAILED = 5,
  DIED = 6,
}

export const jobExecutionStates: JobExecutionState[] =
  Object.keys(JobExecutionState).filter(p => !isNumeric(p)).map(state => JobExecutionState[state]);

export class JobExecutionStatus {
  static DONE_STATES = [
    JobExecutionState.ABORTED_MANUAL,
    JobExecutionState.FAILED,
    JobExecutionState.FINISHED,
  ];

  id: string;
  jobId: string;
  state: JobExecutionState;
  executionsStateMap: Map<string, number>;
  startTime: string;
  endTime: string;
  documentsCrawled: number;
  bytesCrawled: number;
  urisCrawled: number;
  documentsFailed: number;
  documentsOutOfScope: number;
  documentsRetried: number;
  documentsDenied: number;
  error: ApiError;
  desiredState: JobExecutionState;

  constructor({
                id = '',
                jobId = '',
                state = JobExecutionState.UNDEFINED,
                executionsStateMap = new Map(),
                startTime = '',
                endTime = '',
                documentsCrawled = 0,
                bytesCrawled = 0,
                urisCrawled = 0,
                documentsFailed = 0,
                documentsOutOfScope = 0,
                documentsRetried = 0,
                documentsDenied = 0,
                error = new ApiError(),
                desiredState = JobExecutionState.UNDEFINED
              }: Partial<JobExecutionStatus> = {}) {
    this.id = id;
    this.jobId = jobId;
    this.state = state;
    this.executionsStateMap = executionsStateMap;
    this.startTime = startTime;
    this.endTime = endTime;
    this.documentsCrawled = documentsCrawled;
    this.bytesCrawled = bytesCrawled;
    this.urisCrawled = urisCrawled;
    this.documentsFailed = documentsFailed;
    this.documentsOutOfScope = documentsOutOfScope;
    this.documentsRetried = documentsRetried;
    this.documentsDenied = documentsDenied;
    this.error = error;
    this.desiredState = desiredState;
  }

  static fromProto(proto: JobExecutionStatusProto): JobExecutionStatus {
    return new JobExecutionStatus({
      id: proto.id,
      jobId: proto.jobId,
      state: proto.state as unknown as JobExecutionState,
      executionsStateMap: new Map(Object.entries(proto.executionsState)),
      startTime: fromTimestampProto(proto.startTime),
      endTime: fromTimestampProto(proto.endTime),
      documentsCrawled: Number(proto.documentsCrawled),
      bytesCrawled: Number(proto.bytesCrawled),
      urisCrawled: Number(proto.urisCrawled),
      documentsFailed: Number(proto.documentsFailed),
      documentsOutOfScope: Number(proto.documentsOutOfScope),
      documentsRetried: Number(proto.documentsRetried),
      documentsDenied: Number(proto.documentsDenied),
      error: ApiError.fromProto(proto.error),
      desiredState: proto.desiredState as unknown as JobExecutionState
    });
  }

  static toProto(jobExecutionStatus: JobExecutionStatus): JobExecutionStatusProto {
    return create(JobExecutionStatusSchema, {
      id: jobExecutionStatus.id,
      jobId: jobExecutionStatus.jobId,
      state: jobExecutionStatus.state.valueOf(),
      executionsState: Object.fromEntries(jobExecutionStatus.executionsStateMap),
      documentsCrawled: BigInt(jobExecutionStatus.documentsCrawled || 0),
      bytesCrawled: BigInt(jobExecutionStatus.bytesCrawled || 0),
      urisCrawled: BigInt(jobExecutionStatus.urisCrawled || 0),
      documentsFailed: BigInt(jobExecutionStatus.documentsFailed || 0),
      documentsOutOfScope: BigInt(jobExecutionStatus.documentsOutOfScope || 0),
      documentsRetried: BigInt(jobExecutionStatus.documentsRetried || 0),
      documentsDenied: BigInt(jobExecutionStatus.documentsDenied || 0),
      desiredState: jobExecutionStatus.desiredState.valueOf(),
    });
  }
}
