import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {BASE_LIST_IMPORTS, BaseListComponent} from '../../../../shared/components';
import {
  CrawlExecutionState,
  crawlExecutionStates,
  CrawlExecutionStatus
} from '../../../../shared/models';
import {MatTableModule} from '@angular/material/table';
import {ExecutionQueueCountPipe, JobNamePipe, SeedNamePipe} from '../../pipe';
import {AsyncPipe, DatePipe, NgTemplateOutlet} from '@angular/common';
import {UrlFormatPipe} from '../../../../shared/pipes/url-format.pipe';
import {MatSortModule} from '@angular/material/sort';
import {FlexLayoutModule} from '@ngbracket/ngx-layout';
import {MatIcon} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {MatMenuModule} from '@angular/material/menu';

@Component({
  selector: 'app-crawl-execution-status-list',
  templateUrl: './crawl-execution-status-list.component.html',
  styleUrls: [
    '../../../../shared/components/base-list/base-list.scss',
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    AsyncPipe,
    DatePipe,
    ExecutionQueueCountPipe,
    FlexLayoutModule,
    JobNamePipe,
    MatButtonModule,
    MatIcon,
    MatMenuModule,
    MatSortModule,
    MatTableModule,
    NgTemplateOutlet,
    SeedNamePipe,
    UrlFormatPipe,
    ...BASE_LIST_IMPORTS
  ]
})
export class CrawlExecutionStatusListComponent extends BaseListComponent<CrawlExecutionStatus> {
  readonly CrawlExecutionState = CrawlExecutionState;
  readonly crawlExecutionStates = crawlExecutionStates;

  override multiSelect = false;

  @Input()
  override sortActive = 'startTime';

  override displayedColumns: string[] = ['seedId', 'jobId', 'state', 'desiredState', 'errorCode', 'documentsCrawled', 'queueCount', 'startTime', 'endTime', 'extra', 'action'];

}
