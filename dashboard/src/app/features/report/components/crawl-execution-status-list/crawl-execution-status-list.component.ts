import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {
  CrawlExecutionState,
  crawlExecutionStates,
  CrawlExecutionStatus
} from '../../../../shared/models';
import {JobNamePipe, SeedNamePipe} from '../../pipe';
import {DatePipe} from '@angular/common';
import {REPORT_LIST_IMPORTS, ReportListBaseComponent} from '../report-list/report-list-base';

@Component({
  selector: 'app-crawl-execution-status-list',
  templateUrl: './crawl-execution-status-list.component.html',
  styleUrls: ['../report-list/report-list.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    DatePipe,
    JobNamePipe,
    SeedNamePipe,
    ...REPORT_LIST_IMPORTS,
  ]
})
export class CrawlExecutionStatusListComponent extends ReportListBaseComponent<CrawlExecutionStatus> {
  readonly CrawlExecutionState = CrawlExecutionState;
  readonly crawlExecutionStates = crawlExecutionStates;

  @Input()
  override sortActive = 'startTime';

  @Input() hasOverflowActions: (row: CrawlExecutionStatus) => boolean = () => true;

  override displayedColumns: string[] = ['seedId', 'jobId', 'state', 'desiredState', 'errorCode', 'documentsCrawled', 'startTime', 'endTime', 'action'];

}
