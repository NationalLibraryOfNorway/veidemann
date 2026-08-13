import { Injectable, inject } from '@angular/core';
import {EMPTY, Observable} from 'rxjs';
import {create} from '@bufbuild/protobuf';

import {FieldMaskSchema} from '../../../../api/commons/v1/resources_pb';
import {JobExecutionsListRequest, JobExecutionsListRequestSchema} from '../../../../api/report/v1/report_pb';
import {ConfigObject, ConfigRef, JobExecutionState, JobExecutionStatus, Kind} from '../../../shared/models';
import {ReportApiService} from '../../../core';
import {catchError, shareReplay} from 'rxjs/operators';
import {Detail, Sort, toTimestampProto} from '../../../shared/func';
import {ConfigService, LoadingService} from '../../../shared/services';
import {ListRange} from '../../../shared/models';

export interface JobExecutionStatusQuery extends Sort {
  jobId: string;
  stateList: JobExecutionState[];
  startTimeTo: string;
  startTimeFrom: string;
}

@Injectable({
  providedIn: 'root'
})
export class JobExecutionService extends LoadingService {
  private reportApiService = inject(ReportApiService);
  private configService = inject(ConfigService);


  private readonly cache: Map<string, Observable<ConfigObject>>;

  constructor() {
    super();
    this.cache = new Map();
  }

  private static getListRequest(query: JobExecutionStatusQuery, range: ListRange): JobExecutionsListRequest {
    const listRequest = create(JobExecutionsListRequestSchema, {
      offset: range.offset,
      pageSize: range.pageSize
    });
    const queryTemplate = new JobExecutionStatus();
    const fieldMask = create(FieldMaskSchema);

    if (query.jobId) {
      queryTemplate.jobId = query.jobId;
      fieldMask.paths.push('jobId');
    }

    if (fieldMask.paths.length > 0) {
      listRequest.queryTemplate = JobExecutionStatus.toProto(queryTemplate);
      listRequest.queryMask = fieldMask;
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

    if (query.direction && query.active) {
      listRequest.orderByPath = query.active;
      listRequest.orderDescending = query.direction === 'desc';
    }

    return listRequest;
  }

  get(query: Detail): Observable<JobExecutionStatus> {
    const listRequest = create(JobExecutionsListRequestSchema, {id: [query.id], watch: query.watch});
    return this.reportApiService.listJobExecutions(listRequest);
  }

  getJob(id: string): Observable<ConfigObject> {
    const configRef = new ConfigRef({id, kind: Kind.CRAWLJOB});
    if (this.cache.has(id)) {
      return this.cache.get(id);
    }
    const job$: Observable<ConfigObject> = this.configService.get(configRef).pipe(
      shareReplay(1),
      catchError(() => {
        this.cache.delete(id);
        return EMPTY;
      })
    );
    this.cache.set(id, job$);

    return job$;
  }

  search(query: JobExecutionStatusQuery, range: ListRange): Observable<JobExecutionStatus> {
    return this.reportApiService.listJobExecutions(JobExecutionService.getListRequest(query, range));
  }
}
