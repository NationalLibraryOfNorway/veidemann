import {AsyncPipe, DatePipe, DecimalPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {MatCardModule} from '@angular/material/card';
import {MatIcon} from '@angular/material/icon';
import {RouterLink} from '@angular/router';

import {
  CrawlExecutionState,
  ExtraStatusCodes,
  JobExecutionStatus
} from '../../../../shared/models/report';
import {FileSizePipe} from '../../../../shared/pipes/filesize.pipe';
import {crawlExecutionStatePresentation, jobExecutionStatePresentation} from '../../func';
import {JobexecutionTotalQueuePipe, JobNamePipe} from '../../pipe';

interface CrawlExecutionStateCount {
  count: number;
  state: CrawlExecutionState;
}

@Component({
  selector: 'app-job-execution-status',
  templateUrl: './job-execution-status.component.html',
  styleUrls: ['./job-execution-status.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    AsyncPipe,
    DatePipe,
    DecimalPipe,
    FileSizePipe,
    JobexecutionTotalQueuePipe,
    JobNamePipe,
    MatCardModule,
    MatIcon,
    RouterLink,
  ],
  standalone: true,
})
export class JobExecutionStatusComponent {
  readonly ExtraStatusCodes = ExtraStatusCodes;
  readonly crawlStatePresentation = crawlExecutionStatePresentation;
  readonly statePresentation = jobExecutionStatePresentation;

  @Input({required: true})
  jobExecutionStatus: JobExecutionStatus;

  get executionStateCounts(): readonly CrawlExecutionStateCount[] {
    return [...this.jobExecutionStatus.executionsStateMap.entries()]
      .map(([key, count]) => ({count, state: this.toCrawlExecutionState(key)}))
      .filter(({count, state}) => count > 0 && state !== null)
      .sort((left, right) => left.state - right.state);
  }

  hasError(): boolean {
    const error = this.jobExecutionStatus.error;
    return !!error && (error.code !== 0 || !!error.msg?.trim() || !!error.detail?.trim());
  }

  private toCrawlExecutionState(key: string): CrawlExecutionState | null {
    const numericState = Number.parseInt(key, 10);
    if (!Number.isNaN(numericState)) {
      return numericState as CrawlExecutionState;
    }
    const state = CrawlExecutionState[key as keyof typeof CrawlExecutionState];
    return typeof state === 'number' ? state : null;
  }
}
