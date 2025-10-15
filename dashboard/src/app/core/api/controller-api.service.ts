import {Injectable} from '@angular/core';

import {EMPTY, from, Observable} from 'rxjs';
import {catchError, map} from 'rxjs/operators';
import {Empty} from 'google-protobuf/google/protobuf/empty_pb';

import {ControllerPromiseClient} from '../../../api';
import {AuthService} from '../auth';
import {Role} from '../../shared/models/config';
import {
  CrawlerStatus,
  ExecutionId,
  RunCrawlReply,
  RunCrawlRequest
} from '../../shared/models/controller/controller.model';
import {ApplicationErrorHandler} from '../error.handler';
import {CrawlExecutionStatus, JobExecutionStatus} from '../../shared/models/report';
import {CountResponse} from '../../shared/models';
import {AppConfig} from '../../app.config';


@Injectable({
  providedIn: 'root'
})
export class ControllerApiService {

  private controllerPromiseClient: ControllerPromiseClient;

  constructor(private authService: AuthService,
              private appConfig: AppConfig,
              private errorHandler: ApplicationErrorHandler) {}

  private getClient(): ControllerPromiseClient {
    if (!this.controllerPromiseClient) {
      if (!this.appConfig.grpcWebUrl) {
        throw new Error('grpcWebUrl is not configured yet');
      }
      this.controllerPromiseClient = new ControllerPromiseClient(this.appConfig.grpcWebUrl, null, null);
    }
    return this.controllerPromiseClient;
  }

  async getOpenIdConnectIssuer(): Promise<string> {
    const response = await this.getClient()
      .getOpenIdConnectIssuer(new Empty());
    return response.getOpenIdConnectIssuer();
  }

  async getRolesForActiveUser(): Promise<Role[]> {
    const roleList = await this.getClient()
      .getRolesForActiveUser(new Empty(), this.authService.metadata);
    return roleList.getRoleList();
  }

  getCrawlerStatus(): Observable<CrawlerStatus> {
    return from(this.controllerPromiseClient.status(new Empty(), this.authService.metadata))
      .pipe(map(CrawlerStatus.fromProto));
  }

  pauseCrawler(): void {
    this.controllerPromiseClient.pauseCrawler(new Empty(), this.authService.metadata);
  }

  unpauseCrawler(): void {
    this.controllerPromiseClient.unPauseCrawler(new Empty(), this.authService.metadata);
  }

  runCrawl(request: RunCrawlRequest): Observable<RunCrawlReply> {
    return from(this.controllerPromiseClient.runCrawl(RunCrawlRequest.toProto(request), this.authService.metadata))
      .pipe(
        map(RunCrawlReply.fromProto),
        catchError(error => {
          this.errorHandler.handleError(error);
          return EMPTY;
        })
      );
  }

  abortJobExecution(request: ExecutionId): Observable<JobExecutionStatus> {
    return from(this.controllerPromiseClient.abortJobExecution(ExecutionId.toProto(request), this.authService.metadata))
      .pipe(
        map(jobExecutionStaus => JobExecutionStatus.fromProto(jobExecutionStaus)),
        catchError(error => {
          this.errorHandler.handleError(error);
          return EMPTY;
        })
      );
  }

  abortCrawlExecution(request: ExecutionId): Observable<CrawlExecutionStatus> {
    return from(this.controllerPromiseClient.abortCrawlExecution(ExecutionId.toProto(request), this.authService.metadata))
      .pipe(
        map(crawlExecutionStatus => CrawlExecutionStatus.fromProto(crawlExecutionStatus)),
        catchError(error => {
          this.errorHandler.handleError(error);
          return EMPTY;
        })
      );
  }

  queueCountForCrawlExecution(request: ExecutionId): Observable<CountResponse> {
    return from(this.controllerPromiseClient.queueCountForCrawlExecution(ExecutionId.toProto(request), this.authService.metadata))
      .pipe(
        map(countResponse => CountResponse.fromProto(countResponse)),
        catchError(error => {
          this.errorHandler.handleError(error);
          return EMPTY;
        })
      );
  }
}
