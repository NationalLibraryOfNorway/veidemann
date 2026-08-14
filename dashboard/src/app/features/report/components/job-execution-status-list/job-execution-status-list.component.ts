import {ChangeDetectionStrategy, Component, Input} from '@angular/core';


import {DatePipe, DecimalPipe} from '@angular/common';
import {JobNamePipe} from '../../pipe';
import {JobExecutionState, JobExecutionStatus} from '../../../../shared/models';
import {durationBetweenDates, isValidDate} from '../../../../shared/func';
import {FileSizePipe} from '../../../../shared/pipes/filesize.pipe';
import {jobExecutionStatePresentation} from '../../func';
import {REPORT_LIST_IMPORTS, ReportListBaseComponent} from '../report-list/report-list-base';

@Component({
  selector: 'app-job-execution-status-list',
  templateUrl: './job-execution-status-list.component.html',
  styleUrls: ['../report-list/report-list.scss'],
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

  @Input()
  queueCounts: ReadonlyMap<string, number> = new Map();

  override displayedColumns: string[] = [
    'jobId',
    'state',
    'queueSize',
    'documentsCrawled',
    'bytesCrawled',
    'startTime',
    'endTime',
    'duration',
    'action',
  ];

  duration(row: JobExecutionStatus): string {
    if (!row.startTime || !isValidDate(new Date(row.startTime)) ||
      (row.endTime && !isValidDate(new Date(row.endTime)))) {
      return '';
    }
    return durationBetweenDates(row.startTime, row.endTime);
  }

  endTimeFallback(row: JobExecutionStatus): string {
    const state = jobExecutionStatePresentation(row.state);
    const desiredState = jobExecutionStatePresentation(row.desiredState);
    if (state.lifecycle === 'active') {
      return desiredState.lifecycle !== 'undefined' && desiredState.label !== state.label
        ? desiredState.label
        : '';
    }
    return $localize`:@@commonNotAvailable:Not available`;
  }

  queueCount(row: JobExecutionStatus): number | null {
    if (JobExecutionStatus.DONE_STATES.includes(row.state)) {
      return 0;
    }
    return this.queueCounts.get(row.id) ?? null;
  }

}
