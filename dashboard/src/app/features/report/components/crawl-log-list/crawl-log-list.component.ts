import {ChangeDetectionStrategy, Component, Input} from '@angular/core';

import {CrawlLog} from '../../../../shared/models';
import {DatePipe} from '@angular/common';
import {REPORT_LIST_IMPORTS, ReportListBaseComponent} from '../report-list/report-list-base';

@Component({
  selector: 'app-crawl-log-list',
  templateUrl: './crawl-log-list.component.html',
  styleUrls: ['./crawl-log-list.component.scss',
    '../report-list/report-list.scss',
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    DatePipe,
    ...REPORT_LIST_IMPORTS,
  ],
  standalone: true
})
export class CrawlLogListComponent extends ReportListBaseComponent<CrawlLog> {

  @Input()
  override sortActive = 'timestamp';

  override displayedColumns: string[] =
    ['requestedUri', 'timestamp', 'statusCode', 'discoveryPath', 'contentType', 'action'];

}
