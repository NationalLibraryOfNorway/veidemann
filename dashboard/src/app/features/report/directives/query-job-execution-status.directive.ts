import {Directive, Host, Inject} from '@angular/core';
import {BASE_LIST} from '../../../shared/directives';
import {BaseList, JobExecutionStatus, ListDataSource} from '../../../shared/models';
import {QueryWithPageLengthDirective} from './query-with-page-length.directive';
import {JobExecutionService, JobExecutionStatusQuery} from '../services';


@Directive({
    selector: '[appQueryJobExecutionStatus]',
    standalone: true
})
export class QueryJobExecutionStatusDirective extends QueryWithPageLengthDirective<JobExecutionStatusQuery, JobExecutionStatus> {

  constructor(protected override service: JobExecutionService,
              @Host() @Inject(BASE_LIST) protected override baseList: BaseList<JobExecutionStatus>,
              protected override dataSource: ListDataSource<JobExecutionStatus>) {
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
