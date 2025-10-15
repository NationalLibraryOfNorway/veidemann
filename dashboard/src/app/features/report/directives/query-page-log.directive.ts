import { Directive, Inject } from '@angular/core';
import { map, switchMap, takeUntil } from 'rxjs/operators';
import { BASE_LIST, QueryDirective } from '../../../shared/directives';
import { BaseList, ListDataSource } from '../../../shared/models';
import { PageLog } from '../../../shared/models/log';
import { PageLogQuery, PageLogService } from '../services';


@Directive({
    selector: '[appQueryPageLog]',
    standalone: true
})
export class QueryPageLogDirective extends QueryDirective<PageLogQuery, PageLog> {
  constructor(protected override service: PageLogService,
              @Inject(BASE_LIST) protected baseList: BaseList<PageLog>) {
    super(service, baseList, new ListDataSource<PageLog>());
  }

  override onInit(): void {
    super.onInit();

    // fake counting
    this.query$.pipe(
      switchMap(query => this.dataSource.connect(null).pipe(
        map(v => (query.pageIndex + 1) * query.pageSize + (v.length % query.pageSize === 0 ? 1 : 0)))
      ),
      takeUntil(this.ngUnsubscribe),
    ).subscribe(length => this.baseList.length = length);
  }
}
