import {ChangeDetectionStrategy, Component, Input} from '@angular/core';


import {AsyncPipe, DatePipe, NgTemplateOutlet} from '@angular/common';
import {JobNamePipe} from '../../pipe';
import {MatTableModule} from '@angular/material/table';
import {MatSortModule} from '@angular/material/sort';
import {MatMenuModule} from '@angular/material/menu';
import {MatIcon} from '@angular/material/icon';
import {JobExecutionState, JobExecutionStatus} from '../../../../shared/models';
import {BASE_LIST_IMPORTS, BaseListComponent} from '../../../../shared/components';
import {MatButtonModule} from '@angular/material/button';
import {FlexDirective, LayoutDirective} from '@ngbracket/ngx-layout';

@Component({
  selector: 'app-job-execution-status-list',
  templateUrl: './job-execution-status-list.component.html',
  styleUrls: [
    '../../../../shared/components/base-list/base-list.scss',
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    AsyncPipe,
    DatePipe,
    FlexDirective,
    JobNamePipe,
    LayoutDirective,
    MatButtonModule,
    MatIcon,
    MatMenuModule,
    MatSortModule,
    MatTableModule,
    NgTemplateOutlet,
    ...BASE_LIST_IMPORTS,
  ],
  standalone: true
})
export class JobExecutionStatusListComponent extends BaseListComponent<JobExecutionStatus> {
  readonly JobExecutionState = JobExecutionState;

  @Input()
  override multiSelect = false;

  @Input()
  override sortActive = 'startTime';

  override displayedColumns: string[] = ['jobId', 'state', 'desiredState', 'startTime', 'endTime', 'extra', 'action'];

}
