import { Injectable, inject } from '@angular/core';
import {create} from '@bufbuild/protobuf';
import {CallOptions, Client, createClient} from '@connectrpc/connect';
import {createGrpcWebTransport} from '@connectrpc/connect-web';
import {EMPTY, from, Observable} from 'rxjs';
import {catchError, map} from 'rxjs/operators';

import {Controller} from '../../../api/controller/v1/controller_pb';
import {ExecutionIds, ExecutionIdsSchema, QueueCountsResponse} from '../../../api/frontier/v1/frontier_pb';
import {AuthService} from '../auth';
import {Role} from '../../shared/models/config';
import {
  CrawlerStatus,
  ExecutionId,
  RunCrawlReply,
  RunCrawlRequest,
} from '../../shared/models/controller/controller.model';
import {ApplicationErrorHandler} from '../error.handler';
import {CrawlExecutionStatus, JobExecutionStatus} from '../../shared/models/report';
import {AppConfig} from '../../app.config';

@Injectable({providedIn: 'root'})
export class ControllerApiService {
  private authService = inject(AuthService);
  private appConfig = inject(AppConfig);
  private errorHandler = inject(ApplicationErrorHandler);

  private client?: Client<typeof Controller>;

  private getClient(): Client<typeof Controller> {
    if (!this.client) {
      if (!this.appConfig.grpcWebUrl) {
        throw new Error('grpcWebUrl is not configured yet');
      }
      this.client = createClient(Controller, createGrpcWebTransport({
        baseUrl: this.appConfig.grpcWebUrl,
      }));
    }
    return this.client;
  }

  private get callOptions(): CallOptions {
    return {headers: this.authService.metadata};
  }

  async getOpenIdConnectIssuer(): Promise<string> {
    const response = await this.getClient().getOpenIdConnectIssuer({}, this.callOptions);
    return response.openIdConnectIssuer;
  }

  async getRolesForActiveUser(): Promise<Role[]> {
    const response = await this.getClient().getRolesForActiveUser({}, this.callOptions);
    return response.role as Role[];
  }

  getCrawlerStatus(): Observable<CrawlerStatus> {
    return from(this.getClient().status({}, this.callOptions)).pipe(map(CrawlerStatus.fromProto));
  }

  pauseCrawler(): Observable<void> {
    return from(this.getClient().pauseCrawler({}, this.callOptions)).pipe(
      map(() => undefined),
      catchError(error => {
        this.errorHandler.handleError(error);
        return EMPTY;
      }),
    );
  }

  unpauseCrawler(): Observable<void> {
    return from(this.getClient().unPauseCrawler({}, this.callOptions)).pipe(
      map(() => undefined),
      catchError(error => {
        this.errorHandler.handleError(error);
        return EMPTY;
      }),
    );
  }

  runCrawl(request: RunCrawlRequest): Observable<RunCrawlReply> {
    return from(this.getClient().runCrawl(RunCrawlRequest.toProto(request), this.callOptions)).pipe(
      map(RunCrawlReply.fromProto),
      catchError(error => {
        this.errorHandler.handleError(error);
        return EMPTY;
      }),
    );
  }

  abortJobExecution(request: ExecutionId): Observable<JobExecutionStatus> {
    return from(this.getClient().abortJobExecution(ExecutionId.toProto(request), this.callOptions)).pipe(
      map(JobExecutionStatus.fromProto),
      catchError(error => {
        this.errorHandler.handleError(error);
        return EMPTY;
      }),
    );
  }

  abortCrawlExecution(request: ExecutionId): Observable<CrawlExecutionStatus> {
    return from(this.getClient().abortCrawlExecution(ExecutionId.toProto(request), this.callOptions)).pipe(
      map(CrawlExecutionStatus.fromProto),
      catchError(error => {
        this.errorHandler.handleError(error);
        return EMPTY;
      }),
    );
  }

  queueCountsForCrawlExecutions(ids: readonly string[]): Observable<ReadonlyMap<string, number>> {
    return this.queueCounts(ids, request => this.getClient().queueCountsForCrawlExecutions(request, this.callOptions));
  }

  queueCountsForJobExecutions(ids: readonly string[]): Observable<ReadonlyMap<string, number>> {
    return this.queueCounts(ids, request => this.getClient().queueCountsForJobExecutions(request, this.callOptions));
  }

  private queueCounts(
    ids: readonly string[],
    requestCounts: (request: ExecutionIds) => PromiseLike<QueueCountsResponse>,
  ): Observable<ReadonlyMap<string, number>> {
    const executionIds = create(ExecutionIdsSchema, {id: [...ids]});
    return from(requestCounts(executionIds)).pipe(
      map(response => new Map(
        Object.entries(response.counts).map(([id, count]) => [id, Number(count)]),
      )),
      catchError(error => {
        this.errorHandler.handleError(error);
        return EMPTY;
      }),
    );
  }
}
