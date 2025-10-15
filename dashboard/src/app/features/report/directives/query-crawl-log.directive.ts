import { Directive, Inject } from '@angular/core';
import { map, switchMap, takeUntil } from 'rxjs/operators';

import { BASE_LIST, QueryDirective } from '../../../shared/directives';
import { BaseList, CrawlLog, ListDataSource } from '../../../shared/models';
import { CrawlLogQuery, CrawlLogService } from '../services';


@Directive({
    selector: '[appQueryCrawlLog]',
    standalone: true
})
export class QueryCrawlLogDirective extends QueryDirective<CrawlLogQuery, CrawlLog> {
  constructor(protected override service: CrawlLogService,
              @Inject(BASE_LIST) protected baseList: BaseList<CrawlLog>) {
    super(service, baseList, new ListDataSource<CrawlLog>());
  }

  override onInit(): void {
    super.onInit();

    // fake counting
    this.query$.pipe(
      switchMap(query => this.dataSource.connect(null).pipe(
        map(dataSource => (query.pageIndex + 1) * query.pageSize + (dataSource.length % query.pageSize === 0 ? 1 : 0)))
      ),
      takeUntil(this.ngUnsubscribe),
    ).subscribe(length => this.baseList.length = length);
  }
}
