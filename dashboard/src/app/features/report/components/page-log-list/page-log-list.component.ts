import {ChangeDetectionStrategy, Component} from '@angular/core';
import {PageLog} from '../../../../shared/models';
import {REPORT_LIST_IMPORTS, ReportListBaseComponent} from '../report-list/report-list-base';

@Component({
  selector: 'app-pagelog-list',
  templateUrl: './page-log-list.component.html',
  styleUrls: ['../report-list/report-list.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    ...REPORT_LIST_IMPORTS,
  ]
})
export class PageLogListComponent extends ReportListBaseComponent<PageLog> {

  override displayedColumns: string[] = ['uri', 'nrOfResources', 'nrOfOutlinks'];

}
