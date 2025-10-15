import {Directive, Inject} from '@angular/core';
import {BASE_LIST} from '../../../shared/directives';
import {CrawlExecutionService, CrawlExecutionStatusQuery} from '../services';
import {CrawlExecutionStatus} from '../../../shared/models/report';
import {BaseList, ListDataSource} from '../../../shared/models';
import {QueryWithPageLengthDirective} from './query-with-page-length.directive';


@Directive({
    selector: '[appQueryCrawlExecutionStatus]',
    standalone: true
})
export class QueryCrawlExecutionStatusDirective extends QueryWithPageLengthDirective<CrawlExecutionStatusQuery, CrawlExecutionStatus> {

  constructor(protected override service: CrawlExecutionService,
              @Inject(BASE_LIST) protected override baseList: BaseList<CrawlExecutionStatus>,
              protected override dataSource: ListDataSource<CrawlExecutionStatus>) {
    super(service, baseList, dataSource);
  }

  protected override onQuery() {
    if (this.query.watch) {
      this.subject.next(this.query);
    } else {
      super.onQuery();
    }
  }
}
