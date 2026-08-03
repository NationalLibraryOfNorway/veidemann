import {Injectable} from '@angular/core';
import {create} from '@bufbuild/protobuf';
import {CallOptions, Client, createClient} from '@connectrpc/connect';
import {createGrpcWebTransport} from '@connectrpc/connect-web';
import {EMPTY, Observable} from 'rxjs';
import {catchError, defaultIfEmpty, map} from 'rxjs/operators';

import {FieldMaskSchema} from '../../../api/commons/v1/resources_pb';
import {
  CrawlExecutionsListRequest,
  CrawlExecutionsListRequestSchema,
  JobExecutionsListRequest,
  JobExecutionsListRequestSchema,
  Report,
} from '../../../api/report/v1/report_pb';
import {AuthService} from '../auth';
import {ErrorService} from '../error.service';
import {CrawlExecutionStatus, JobExecutionStatus} from '../../shared/models';
import {AppConfig} from '../../app.config';
import {fromServerStream} from './connect-observable';

@Injectable({providedIn: 'root'})
export class ReportApiService {
  private client?: Client<typeof Report>;

  constructor(
    private authService: AuthService,
    private appConfig: AppConfig,
    private errorService: ErrorService,
  ) {}

  private getClient(): Client<typeof Report> {
    if (!this.client) {
      if (!this.appConfig.grpcWebUrl) {
        throw new Error('grpcWebUrl is not configured yet');
      }
      this.client = createClient(Report, createGrpcWebTransport({
        baseUrl: this.appConfig.grpcWebUrl,
      }));
    }
    return this.client;
  }

  private get callOptions(): CallOptions {
    return {headers: this.authService.metadata};
  }

  listJobExecutions(listRequest: JobExecutionsListRequest): Observable<JobExecutionStatus> {
    return fromServerStream(signal => this.getClient().listJobExecutions(listRequest, {
      ...this.callOptions,
      signal,
    })).pipe(
      map(JobExecutionStatus.fromProto),
      catchError(error => {
        this.errorService.dispatch(error);
        return EMPTY;
      }),
    );
  }

  listCrawlExecutions(listRequest: CrawlExecutionsListRequest): Observable<CrawlExecutionStatus> {
    return fromServerStream(signal => this.getClient().listExecutions(listRequest, {
      ...this.callOptions,
      signal,
    })).pipe(
      map(CrawlExecutionStatus.fromProto),
      catchError(error => {
        this.errorService.dispatch(error);
        return EMPTY;
      }),
    );
  }

  getLastJobStatus(jobId: string): Observable<JobExecutionStatus> {
    return this.listJobExecutions(this.jobRequest(jobId, 1, true)).pipe(defaultIfEmpty(null));
  }

  getLastSeedStatus(seedId: string, pageSize = 1): Observable<CrawlExecutionStatus> {
    return this.listCrawlExecutions(this.crawlRequest(seedId, pageSize, true)).pipe(defaultIfEmpty(null));
  }

  getJobStatus(jobId: string): Observable<JobExecutionStatus> {
    return this.listJobExecutions(this.jobRequest(jobId));
  }

  getSeedStatus(seedId: string): Observable<CrawlExecutionStatus> {
    return this.listCrawlExecutions(this.crawlRequest(seedId));
  }

  private jobRequest(jobId: string, pageSize = 0, newestFirst = false): JobExecutionsListRequest {
    return create(JobExecutionsListRequestSchema, {
      queryMask: create(FieldMaskSchema, {paths: ['jobId']}),
      queryTemplate: JobExecutionStatus.toProto(new JobExecutionStatus({jobId})),
      orderByPath: newestFirst ? 'startTime' : '',
      orderDescending: newestFirst,
      pageSize,
    });
  }

  private crawlRequest(seedId: string, pageSize = 0, newestFirst = false): CrawlExecutionsListRequest {
    return create(CrawlExecutionsListRequestSchema, {
      queryMask: create(FieldMaskSchema, {paths: ['seedId']}),
      queryTemplate: CrawlExecutionStatus.toProto(new CrawlExecutionStatus({seedId})),
      orderByPath: newestFirst ? 'startTime' : '',
      orderDescending: newestFirst,
      pageSize,
    });
  }
}
