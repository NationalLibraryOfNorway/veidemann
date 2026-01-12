import {ChangeDetectionStrategy, ChangeDetectorRef, Component, forwardRef, Input} from '@angular/core';
import {BASE_LIST_IMPORTS, BaseListComponent} from '../../../../shared/components';
import {
  CrawlExecutionState,
  crawlExecutionStates,
  CrawlExecutionStatus,
  ListDataSource
} from '../../../../shared/models';
import {BASE_LIST} from '../../../../shared/directives';
import {animate, state, style, transition, trigger} from '@angular/animations';
import {MatPaginatorModule} from '@angular/material/paginator';
import {MatTableModule} from '@angular/material/table';
import {CrawlExecutionFetchPipe, ExecutionQueueCountPipe, JobNamePipe, SeedNamePipe} from '../../pipe';
import {AsyncPipe, DatePipe, NgTemplateOutlet} from '@angular/common';
import {UrlFormatPipe} from '../../../../shared/pipes/url-format.pipe';
import {MatSortModule} from '@angular/material/sort';
import {FlexLayoutModule} from '@ngbracket/ngx-layout';
import {MatIcon} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {MatMenuModule} from '@angular/material/menu';
import {CrawlExecutionPreviewComponent} from '../crawl-execution-preview/crawl-execution-preview.component';

@Component({
  selector: 'app-crawl-execution-status-list',
  templateUrl: './crawl-execution-status-list.component.html',
  styleUrls: [
    '../../../../shared/components/base-list/base-list.scss',
    '../../../../shared/components/base-list/base-list-odd-preview.scss',
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    ListDataSource,
    {
      provide: BASE_LIST,
      useExisting: forwardRef(() => CrawlExecutionStatusListComponent)
    }
  ],
  animations: [
    trigger('detailExpand', [
      state('collapsed', style({height: '0px', minHeight: '0'})),
      state('expanded', style({height: '*'})),
      transition('expanded <=> collapsed', animate('225ms cubic-bezier(0.4, 0.0, 0.2, 1)')),
    ]),
  ],
  standalone: true,
  imports: [
    AsyncPipe,
    CrawlExecutionFetchPipe,
    CrawlExecutionPreviewComponent,
    DatePipe,
    ExecutionQueueCountPipe,
    FlexLayoutModule,
    JobNamePipe,
    MatButtonModule,
    MatIcon,
    MatMenuModule,
    MatPaginatorModule,
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

  constructor(protected override cdr: ChangeDetectorRef) {
    super(cdr);
  }
}
