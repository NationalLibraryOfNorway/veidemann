import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { catchError, defaultIfEmpty, map } from 'rxjs/operators';

import { AuthService } from '../auth';
import { ErrorService } from '../error.service';
import { AppConfig } from '../../app.config';

import { createClient } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';

// Generated proto types
import {
  Report,
  JobExecutionsListRequest,
  CrawlExecutionsListRequest,
} from '../../../api/gen/report/v1/report_pb';

// Your domain models
import { JobExecutionStatus, CrawlExecutionStatus } from '../../shared/models';
import {create} from '@bufbuild/protobuf';

@Injectable({
  providedIn: 'root'
})
export class ReportApiService {
  private reportClientConnect: ReturnType<typeof createClient>;

  constructor(
    private authService: AuthService,
    private appConfig: AppConfig,
    private errorService: ErrorService
  ) {
    const transport = createConnectTransport({
      baseUrl: appConfig.grpcWebUrl,
    });

    // Use the generated GenService
    this.reportClientConnect = createClient(Report, transport);
  }

  /**
   * Stream JobExecutions and map to domain model
   */
  listJobExecutions(listRequest: JobExecutionsListRequest): Observable<JobExecutionStatus> {
    return new Observable<JobExecutionStatus>(observer => {
      const asyncIterable = this.reportClientConnect['listJobExecutions'](listRequest);

      (async () => {
        try {
          for await (const proto of asyncIterable) {
            observer.next(JobExecutionStatus.fromProto(proto));
          }
          observer.complete();
        } catch (err) {
          this.errorService.dispatch(err);
          observer.error(err);
        }
      })();

      // Optional cleanup/cancellation if using AbortController
      return () => {};
    });
  }

  /**
   * Stream CrawlExecutions and map to domain model
   */
  listCrawlExecutions(listRequest: CrawlExecutionsListRequest): Observable<CrawlExecutionStatus> {
    return new Observable<CrawlExecutionStatus>(observer => {
      const asyncIterable = this.reportClientConnect['listExecutions'](listRequest);

      (async () => {
        try {
          for await (const proto of asyncIterable) {
            observer.next(CrawlExecutionStatus.fromProto(proto));
          }
          observer.complete();
        } catch (err) {
          this.errorService.dispatch(err);
          observer.error(err);
        }
      })();

      return () => {};
    });
  }

  /**
   * Convenience: get last job execution
   */
  getLastJobStatus(jobId: string): Observable<JobExecutionStatus | null> {
    const request = create(JobExecutionsListRequest, {
      id: [],
      state: [],
      watch: false,
      orderByPath: 'startTime',
      orderDescending: true,
      pageSize: 1,
    });

    const template = new JobExecutionStatus();
    const mask = request.queryMask ?? {};
    template.jobId = jobId;

    request.queryTemplate = JobExecutionStatus.toProto(template);
    request.orderByPath = 'startTime';
    request.orderDescending = true;
    request.pageSize = 1;

    return this.listJobExecutions(request).pipe(defaultIfEmpty(null));
  }

  /**
   * Convenience: get last crawl for a seed
   */
  getLastSeedStatus(seedId: string, pageSize = 1): Observable<CrawlExecutionStatus | null> {
    const request = create(CrawlExecutionsListRequest, {
      id: [],
      state: [],
      watch: false,
      orderByPath: 'startTime',
      orderDescending: true,
      pageSize: 1,
    });
    const template = new CrawlExecutionStatus();
    template.seedId = seedId;

    request.queryTemplate = CrawlExecutionStatus.toProto(template);
    request.orderByPath = 'startTime';
    request.orderDescending = true;
    request.pageSize = pageSize;

    return this.listCrawlExecutions(request).pipe(defaultIfEmpty(null));
  }
}
