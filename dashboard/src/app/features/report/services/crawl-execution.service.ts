import { Injectable, inject } from '@angular/core';
import { EMPTY, Observable } from 'rxjs';
import { catchError, shareReplay } from 'rxjs/operators';
import { create } from '@bufbuild/protobuf';

import { FieldMaskSchema } from '../../../../api/commons/v1/resources_pb';
import { CrawlExecutionsListRequest, CrawlExecutionsListRequestSchema } from '../../../../api/report/v1/report_pb';
import { ReportApiService } from '../../../core';
import { Detail, Sort, toTimestampProto, Watch } from '../../../shared/func';
import { ConfigObject, ConfigRef, CrawlExecutionState, CrawlExecutionStatus, Kind, ListRange } from '../../../shared/models';
import { ConfigService, LoadingService } from '../../../shared/services';

export interface CrawlExecutionStatusQuery extends Sort, Watch {
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
export class CrawlExecutionService extends LoadingService {
  private reportApiService = inject(ReportApiService);
  private configService = inject(ConfigService);

  private readonly cache: Map<string, Observable<ConfigObject>>;

  constructor() {
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
    const seed$ = this.configService.get(configRef, {suppressNotFound: true}).pipe(
      shareReplay(1),
      catchError(() => {
        this.cache.delete(id);
        return EMPTY;
      })
    );
    this.cache.set(id, seed$);

    return seed$;
  }

  search(query: CrawlExecutionStatusQuery, range: ListRange): Observable<CrawlExecutionStatus> {
    return this.reportApiService.listCrawlExecutions(this.getListRequest(query, range));
  }

  private getListRequest(query: CrawlExecutionStatusQuery, range: ListRange): CrawlExecutionsListRequest {
    const listRequest = create(CrawlExecutionsListRequestSchema, {
      offset: range.offset,
      pageSize: range.pageSize
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
