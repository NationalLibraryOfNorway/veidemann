import {AsyncPipe, DecimalPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, inject, Input} from '@angular/core';
import {MatIcon} from '@angular/material/icon';
import {RouterLink} from '@angular/router';
import {Observable, of} from 'rxjs';

import {
  CrawlExecutionState,
  ExtraStatusCodes,
  JobExecutionStatus
} from '../../../../shared/models/report';
import {ConfigObject} from '../../../../shared/models/config';
import {DetailHeaderComponent} from '../../../../shared/components';
import {crawlExecutionStatePresentation} from '../../func';
import {JobExecutionService} from '../../services';
import {
  JobExecutionMetricsSectionComponent
} from '../job-execution-metrics-section/job-execution-metrics-section.component';

interface CrawlExecutionStateCount {
  count: number;
  state: CrawlExecutionState;
}

@Component({
  selector: 'app-job-execution-status',
  templateUrl: './job-execution-status.component.html',
  styleUrls: ['../detail-status-layout.scss', './job-execution-status.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    AsyncPipe,
    DecimalPipe,
    DetailHeaderComponent,
    JobExecutionMetricsSectionComponent,
    MatIcon,
    RouterLink,
  ],
  standalone: true,
})
export class JobExecutionStatusComponent {
  private readonly jobExecutionService = inject(JobExecutionService);

  readonly ExtraStatusCodes = ExtraStatusCodes;
  readonly crawlStatePresentation = crawlExecutionStatePresentation;

  private status: JobExecutionStatus;
  job$: Observable<ConfigObject | null> = of(null);

  @Input({required: true})
  set jobExecutionStatus(value: JobExecutionStatus) {
    if (value?.jobId !== this.status?.jobId) {
      this.job$ = value?.jobId ? this.jobExecutionService.getJob(value.jobId) : of(null);
    }
    this.status = value;
  }

  get jobExecutionStatus(): JobExecutionStatus {
    return this.status;
  }

  @Input()
  queueSize: number | null = null;

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
