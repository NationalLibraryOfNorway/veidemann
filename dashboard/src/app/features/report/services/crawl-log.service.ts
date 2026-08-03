import {Injectable} from '@angular/core';
import {LogApiService} from '../../../core';
import {Observable} from 'rxjs';
import {create} from '@bufbuild/protobuf';
import {CrawlLog} from '../../../shared/models';
import {FieldMaskSchema} from '../../../../api/commons/v1/resources_pb';
import {CrawlLogListRequest, CrawlLogListRequestSchema} from '../../../../api/log/v1/log_pb';
import {LoadingService} from '../../../shared/services';
import {Detail, Page, Sort, Watch} from '../../../shared/func';
import {Getter, Searcher} from '../../../shared/directives';


export interface CrawlLogQuery extends Page, Sort, Watch {
  jobExecutionId: string;
  executionId: string;
}


@Injectable({
  providedIn: 'root'
})
export class CrawlLogService extends LoadingService
  implements Getter<CrawlLog>, Searcher<CrawlLogQuery, CrawlLog> {

  constructor(private logApiService: LogApiService) {
    super();
  }

  static getListRequest(query: CrawlLogQuery): CrawlLogListRequest {
    const listRequest = create(CrawlLogListRequestSchema, {
      offset: query.pageIndex * query.pageSize,
      pageSize: query.pageSize
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

    if (query.direction) {
      listRequest.orderByPath = query.active;
      listRequest.orderDescending = query.direction === 'desc';
    }

    return listRequest;
  }


  get(query: Detail): Observable<CrawlLog> {
    const listRequest = create(CrawlLogListRequestSchema, {warcId: [query.id]});
    return this.logApiService.listCrawlLogs(listRequest);
  }

  search(query: CrawlLogQuery): Observable<CrawlLog> {
    return this.load(this.logApiService.listCrawlLogs(CrawlLogService.getListRequest(query)));
  }
}
