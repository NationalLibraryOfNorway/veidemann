import { Injectable } from '@angular/core';
import { EMPTY, Observable } from 'rxjs';
import { catchError, shareReplay } from 'rxjs/operators';
import { create } from '@bufbuild/protobuf';

import { FieldMaskSchema } from '../../../../api/commons/v1/resources_pb';
import { CrawlExecutionsListRequest, CrawlExecutionsListRequestSchema } from '../../../../api/report/v1/report_pb';
import { ReportApiService } from '../../../core';
import { Getter, Searcher } from '../../../shared/directives';
import { Detail, Page, Sort, toTimestampProto, Watch } from '../../../shared/func';
import { ConfigObject, ConfigRef, CrawlExecutionState, CrawlExecutionStatus, Kind } from '../../../shared/models';
import { ConfigService, LoadingService } from '../../../shared/services';

export interface CrawlExecutionStatusQuery extends Page, Sort, Watch {
  jobId: string;
  jobExecutionId: string;
  seedId: string;
  stateList: CrawlExecutionState[];
  hasError: boolean;
  startTimeTo: string;
  startTimeFrom: string;
}

@Injectable({
  providedIn: 'root'
})
export class CrawlExecutionService extends LoadingService
  implements Searcher<CrawlExecutionStatusQuery, CrawlExecutionStatus>, Getter<CrawlExecutionStatus> {
  private readonly cache: Map<string, Observable<ConfigObject>>;

  constructor(private reportApiService: ReportApiService,
              private configService: ConfigService) {
    super();
    this.cache = new Map();
  }

  get(query: Detail & Watch): Observable<CrawlExecutionStatus> {
    const listRequest = create(CrawlExecutionsListRequestSchema, {id: [query.id], watch: query.watch});
    return this.reportApiService.listCrawlExecutions(listRequest);
  }

  getSeed(id: string): Observable<ConfigObject> {
    const configRef = new ConfigRef({id, kind: Kind.SEED});
    if (this.cache.has(id)) {
      return this.cache.get(id);
    }
    const seed$ = this.configService.get(configRef).pipe(
      shareReplay(1),
      catchError(() => {
        this.cache.delete(id);
        return EMPTY;
      })
    );
    this.cache.set(id, seed$);

    return seed$;
  }

  search(query: CrawlExecutionStatusQuery): Observable<CrawlExecutionStatus> {
    return this.load(this.reportApiService.listCrawlExecutions(this.getListRequest(query)));
  }

  private getListRequest(query: CrawlExecutionStatusQuery): CrawlExecutionsListRequest {
    const listRequest = create(CrawlExecutionsListRequestSchema, {
      offset: query.pageIndex * query.pageSize,
      pageSize: query.pageSize
    });
    const queryTemplate = new CrawlExecutionStatus();
    const fieldMask = create(FieldMaskSchema);

    if (query.jobId) {
      queryTemplate.jobId = query.jobId;
      fieldMask.paths.push('jobId');
    }

    if (query.jobExecutionId) {
      queryTemplate.jobExecutionId = query.jobExecutionId;
      fieldMask.paths.push('jobExecutionId');
    }

    if (query.seedId) {
      queryTemplate.seedId = query.seedId;
      fieldMask.paths.push('seedId');
    }

    if (fieldMask.paths.length > 0) {
      listRequest.queryTemplate = CrawlExecutionStatus.toProto(queryTemplate);
      listRequest.queryMask = fieldMask;
    }

    if (query.hasError) {
      listRequest.hasError = query.hasError;
    }

    if (query.startTimeTo) {
      listRequest.startTimeTo = toTimestampProto(query.startTimeTo);
    }

    if (query.startTimeFrom) {
      listRequest.startTimeFrom = toTimestampProto(query.startTimeFrom);
    }

    if (query.stateList.length) {
      listRequest.state = query.stateList.map(state => state.valueOf());
    }

    if (query.watch) {
      listRequest.watch = query.watch;
    }

    if (query.direction && query.active) {
      listRequest.orderByPath = query.active;
      listRequest.orderDescending = query.direction === 'desc';
    }

    return listRequest;
  }
}
