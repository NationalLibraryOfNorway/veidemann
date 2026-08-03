import { Injectable, inject } from '@angular/core';
import {LogApiService} from '../../../core';
import {Observable} from 'rxjs';
import {create} from '@bufbuild/protobuf';
import {CrawlLog} from '../../../shared/models';
import {FieldMaskSchema} from '../../../../api/commons/v1/resources_pb';
import {CrawlLogListRequest, CrawlLogListRequestSchema} from '../../../../api/log/v1/log_pb';
import {LoadingService} from '../../../shared/services';
import {Detail, Sort, Watch} from '../../../shared/func';
import {ListRange} from '../../../shared/models';


export interface CrawlLogQuery extends Sort, Watch {
  jobExecutionId: string;
  executionId: string;
}


@Injectable({
  providedIn: 'root'
})
export class CrawlLogService extends LoadingService {
  private logApiService = inject(LogApiService);


  static getListRequest(query: CrawlLogQuery, range: ListRange): CrawlLogListRequest {
    const listRequest = create(CrawlLogListRequestSchema, {
      offset: range.offset,
      pageSize: range.pageSize
    });
    const queryTemplate = new CrawlLog();
    const fieldMask = create(FieldMaskSchema);

    if (query.jobExecutionId) {
      queryTemplate.jobExecutionId = query.jobExecutionId;
      fieldMask.paths.push('jobExecutionId');
    }

    if (query.executionId) {
      queryTemplate.executionId = query.executionId;
      fieldMask.paths.push('executionId');
    }

    if (fieldMask.paths.length > 0) {
      listRequest.queryTemplate = CrawlLog.toProto(queryTemplate);
      listRequest.queryMask = fieldMask;
    }

    if (query.watch) {
      listRequest.watch = query.watch;
    }

    return listRequest;
  }


  get(query: Detail): Observable<CrawlLog> {
    const listRequest = create(CrawlLogListRequestSchema, {warcId: [query.id]});
    return this.logApiService.listCrawlLogs(listRequest);
  }

  search(query: CrawlLogQuery, range: ListRange): Observable<CrawlLog> {
    return this.logApiService.listCrawlLogs(CrawlLogService.getListRequest(query, range));
  }
}
