import { Injectable, inject } from '@angular/core';
import {Observable} from 'rxjs';
import {create} from '@bufbuild/protobuf';
import { Detail, Sort } from '../../../shared/func';
import { LoadingService } from '../../../shared/services';
import { ConfigObject, ListRange, PageLog } from '../../../shared/models';
import { LogApiService } from '../../../core';
import { FieldMaskSchema } from '../../../../api/commons/v1/resources_pb';
import { PageLogListRequest, PageLogListRequestSchema } from '../../../../api/log/v1/log_pb';


export interface PageLogQuery extends Sort {
  executionId: string;
}

@Injectable({
  providedIn: 'root'
})
export class PageLogService extends LoadingService {
  private logApiService = inject(LogApiService);

  private readonly cache: Map<string, ConfigObject>;

  constructor() {
    super();
    this.cache = new Map();
  }

  get(query: Detail): Observable<PageLog> {
    const listRequest = create(PageLogListRequestSchema, {warcId: [query.id]});
    return this.logApiService.listPageLogs(listRequest);
  }

  search(query: PageLogQuery, range: ListRange): Observable<PageLog> {
    return this.logApiService.listPageLogs(this.getListRequest(query, range));
  }

  private getListRequest(query: PageLogQuery, range: ListRange): PageLogListRequest {
    const listRequest = create(PageLogListRequestSchema, {
      offset: range.offset,
      pageSize: range.pageSize
    });
    const queryTemplate = new PageLog();
    const fieldMask = create(FieldMaskSchema);

    if (query.executionId) {
      queryTemplate.executionId = query.executionId;
      fieldMask.paths.push('executionId');
    }

    if (fieldMask.paths.length > 0) {
      listRequest.queryTemplate = PageLog.toProto(queryTemplate);
      listRequest.queryMask = fieldMask;
    }

    return listRequest;
  }
}
