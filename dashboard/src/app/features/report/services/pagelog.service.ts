import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {create} from '@bufbuild/protobuf';
import { Detail, Page, Sort, Watch } from '../../../shared/func';
import { LoadingService } from '../../../shared/services';
import { ConfigObject, PageLog } from '../../../shared/models';
import { LogApiService } from '../../../core';
import { FieldMaskSchema } from '../../../../api/commons/v1/resources_pb';
import { PageLogListRequest, PageLogListRequestSchema } from '../../../../api/log/v1/log_pb';


export interface PageLogQuery extends Page, Sort, Watch {
  uri: string;
  executionId: string;
  jobExecutionId: string;
  // offset: number;
  // orderByPath: string;
  // orderDescending: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class PageLogService extends LoadingService {
  private readonly cache: Map<string, ConfigObject>;

  constructor(private logApiService: LogApiService) {
    super();
    this.cache = new Map();
  }

  get(query: Detail): Observable<PageLog> {
    const listRequest = create(PageLogListRequestSchema, {warcId: [query.id]});
    return this.logApiService.listPageLogs(listRequest);
  }

  search(query: PageLogQuery): Observable<PageLog> {
    return this.logApiService.listPageLogs(this.getListRequest(query));
  }

  private getListRequest(query: PageLogQuery): PageLogListRequest {
    const listRequest = create(PageLogListRequestSchema, {
      offset: query.pageIndex * query.pageSize,
      pageSize: query.pageSize
    });
    const queryTemplate = new PageLog();
    const fieldMask = create(FieldMaskSchema);

    if (query.executionId) {
      queryTemplate.executionId = query.executionId;
      fieldMask.paths.push('executionId');
    }

    if (query.jobExecutionId) {
      queryTemplate.jobExecutionId = query.jobExecutionId;
      fieldMask.paths.push('jobExecutionId');
    }

    if (query.uri) {
      queryTemplate.uri = query.uri;
      fieldMask.paths.push('uri');
    }

    if (fieldMask.paths.length > 0) {
      listRequest.queryTemplate = PageLog.toProto(queryTemplate);
      listRequest.queryMask = fieldMask;
    }

    if (query.watch) {
      listRequest.watch = query.watch;
    }

    if (query.active && query.direction) {
      listRequest.orderByPath = query.active;
      listRequest.orderDescending = query.direction === 'desc';
    }

    return listRequest;
  }
}
