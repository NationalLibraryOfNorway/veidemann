import {create} from '@bufbuild/protobuf';
import {
  CrawlerStatus as CrawlerStatusProto,
  RunCrawlReply as RunCrawlReplyProto,
  RunCrawlReplySchema,
  RunCrawlRequest as RunCrawlRequestProto,
  RunCrawlRequestSchema,
} from '../../../../api/controller/v1/controller_pb';
import {ExecutionId as ExecutionIdProto, ExecutionIdSchema} from '../../../../api/controller/v1/resources_pb';

export class CrawlerStatus {
  runStatus: RunStatus;
  busyCrawlHostGroupCount: number;
  queueSize: number;

  constructor({
                runStatus = 0,
                busyCrawlHostGroupCount = 0,
                queueSize = 0,
              }: Partial<CrawlerStatus> = {}) {
    this.runStatus = runStatus;
    this.busyCrawlHostGroupCount = busyCrawlHostGroupCount;
    this.queueSize = queueSize;
  }

  static fromProto(proto: CrawlerStatusProto): CrawlerStatus {
    return new CrawlerStatus({
      queueSize: Number(proto.queueSize),
      busyCrawlHostGroupCount: Number(proto.busyCrawlHostGroupCount),
      runStatus: proto.runStatus.valueOf() as RunStatus
    });
  }
}


export enum RunStatus {
  RUNNING = 0,
  PAUSED = 1,
  PAUSE_REQUESTED = 2,
}

export class RunCrawlRequest {
  jobId: string;
  seedId: string;

  constructor({
                jobId = '',
                seedId = ''
              }: Partial<RunCrawlRequest> = {}) {
    this.jobId = jobId;
    this.seedId = seedId;
  }

  static fromProto(proto: RunCrawlRequestProto): RunCrawlRequest {
    return new RunCrawlRequest({
      jobId: proto.jobId,
      seedId: proto.seedId
    });
  }

  static toProto(runCrawlRequest: RunCrawlRequest): RunCrawlRequestProto {
    return create(RunCrawlRequestSchema, {jobId: runCrawlRequest.jobId, seedId: runCrawlRequest.seedId});
  }
}

export class RunCrawlReply {
  jobExecutionId: string;

  constructor({
                jobExecutionId = ''
              }: Partial<RunCrawlReply> = {}) {
    this.jobExecutionId = jobExecutionId;
  }

  static fromProto(proto: RunCrawlReplyProto): RunCrawlReply {
    return new RunCrawlReply({
      jobExecutionId: proto.jobExecutionId
    });
  }

  static toProto(runCrawlReply: RunCrawlReply): RunCrawlReplyProto {
    return create(RunCrawlReplySchema, {jobExecutionId: runCrawlReply.jobExecutionId});
  }
}

export class ExecutionId {
  id: string;

  constructor({
                id = ''
              }: Partial<ExecutionId> = {}) {
    this.id = id;
  }

  static fromProto(proto: ExecutionIdProto): ExecutionId {
    return new ExecutionId({
      id: proto.id
    });
  }

  static toProto(executionId: ExecutionId): ExecutionIdProto {
    return create(ExecutionIdSchema, {id: executionId.id});
  }
}
