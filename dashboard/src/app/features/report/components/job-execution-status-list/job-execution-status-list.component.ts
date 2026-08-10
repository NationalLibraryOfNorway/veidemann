import {ChangeDetectionStrategy, Component, Input} from '@angular/core';


import {DatePipe, DecimalPipe} from '@angular/common';
import {JobNamePipe} from '../../pipe';
import {JobExecutionState, JobExecutionStatus} from '../../../../shared/models';
import {durationBetweenDates, isValidDate} from '../../../../shared/func';
import {FileSizePipe} from '../../../../shared/pipes/filesize.pipe';
import {REPORT_LIST_IMPORTS, ReportListBaseComponent} from '../report-list/report-list-base';

@Component({
  selector: 'app-job-execution-status-list',
  templateUrl: './job-execution-status-list.component.html',
  styleUrls: ['./job-execution-status-list.component.scss', '../report-list/report-list.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    DatePipe,
    DecimalPipe,
    FileSizePipe,
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

  showDesiredStateBadge(row: JobExecutionStatus): boolean {
    return !this.displayedColumns.includes('desiredState')
      && row.desiredState !== JobExecutionState.UNDEFINED;
  }

  desiredStateAriaLabel(state: JobExecutionState): string {
    return $localize`:@@crawlJobExecutionReportListDesiredStateBadgeAriaLabel:Desired state: ${JobExecutionState[state]}`;
  }

}
