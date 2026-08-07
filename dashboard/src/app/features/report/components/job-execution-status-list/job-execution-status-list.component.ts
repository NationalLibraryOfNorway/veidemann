import {ChangeDetectionStrategy, Component, Input} from '@angular/core';


import {DatePipe, DecimalPipe} from '@angular/common';
import {JobNamePipe} from '../../pipe';
import {JobExecutionState, JobExecutionStatus} from '../../../../shared/models';
import {durationBetweenDates, isValidDate} from '../../../../shared/func';
import {REPORT_LIST_IMPORTS, ReportListBaseComponent} from '../report-list/report-list-base';

@Component({
  selector: 'app-job-execution-status-list',
  templateUrl: './job-execution-status-list.component.html',
  styleUrls: ['../report-list/report-list.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    DatePipe,
    DecimalPipe,
    JobNamePipe,
    ...REPORT_LIST_IMPORTS,
  ],
  standalone: true
})
export class JobExecutionStatusListComponent extends ReportListBaseComponent<JobExecutionStatus> {
  readonly JobExecutionState = JobExecutionState;

  @Input()
  override sortActive = 'startTime';

  @Input()
  embedded = false;

  override displayedColumns: string[] = ['jobId', 'state', 'desiredState', 'startTime', 'endTime', 'action'];

  duration(row: JobExecutionStatus): string {
    if (!row.startTime || !isValidDate(new Date(row.startTime)) ||
      (row.endTime && !isValidDate(new Date(row.endTime)))) {
      return '';
    }
    return durationBetweenDates(row.startTime, row.endTime);
  }

}
