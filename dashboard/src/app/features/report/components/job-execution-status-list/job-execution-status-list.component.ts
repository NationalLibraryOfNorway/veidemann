import {ChangeDetectionStrategy, ChangeDetectorRef, Component, forwardRef, Input} from '@angular/core';


import {animate, state, style, transition, trigger} from '@angular/animations';
import {AsyncPipe, DatePipe, NgTemplateOutlet} from '@angular/common';
import {JobExecutionFetchPipe, JobNamePipe} from '../../pipe';
import {MatTableModule} from '@angular/material/table';
import {MatSortModule} from '@angular/material/sort';
import {MatMenuModule} from '@angular/material/menu';
import {MatIcon} from '@angular/material/icon';
import {BASE_LIST} from '../../../../shared/directives';
import {JobExecutionState, JobExecutionStatus, ListDataSource} from '../../../../shared/models';
import {BASE_LIST_IMPORTS, BaseListComponent} from '../../../../shared/components';
import {MatButtonModule} from '@angular/material/button';
import {JobExecutionPreviewComponent} from '../job-execution-preview/job-execution-preview.component';
import {MatPaginatorModule} from '@angular/material/paginator';
import {FlexDirective, LayoutDirective} from '@ngbracket/ngx-layout';

@Component({
  selector: 'app-job-execution-status-list',
  templateUrl: './job-execution-status-list.component.html',
  styleUrls: [
    '../../../../shared/components/base-list/base-list.scss',
    '../../../../shared/components/base-list/base-list-odd-preview.scss',
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    ListDataSource,
    {
      provide: BASE_LIST,
      useExisting: forwardRef(() => JobExecutionStatusListComponent)
    }
  ],
  animations: [
    trigger('detailExpand', [
      state('collapsed', style({height: '0px', minHeight: '0'})),
      state('expanded', style({height: '*'})),
      transition('expanded <=> collapsed', animate('225ms cubic-bezier(0.4, 0.0, 0.2, 1)')),
    ]),
  ],
  imports: [
    AsyncPipe,
    DatePipe,
    FlexDirective,
    JobExecutionFetchPipe,
    JobExecutionPreviewComponent,
    JobNamePipe,
    LayoutDirective,
    MatButtonModule,
    MatIcon,
    MatMenuModule,
    MatPaginatorModule,
    MatSortModule,
    MatTableModule,
    NgTemplateOutlet,
    ...BASE_LIST_IMPORTS,
  ],
  standalone: true
})
export class JobExecutionStatusListComponent extends BaseListComponent<JobExecutionStatus> {
  readonly JobExecutionState = JobExecutionState;

  expandedJobExecutionStatus: JobExecutionStatus | null;

  @Input()
  override multiSelect = false;

  @Input()
  override sortActive = 'startTime';

  override displayedColumns: string[] = ['jobId', 'state', 'desiredState', 'startTime', 'endTime', 'extra', 'action'];

  constructor(protected override cdr: ChangeDetectorRef) {
    super(cdr);
  }
}
