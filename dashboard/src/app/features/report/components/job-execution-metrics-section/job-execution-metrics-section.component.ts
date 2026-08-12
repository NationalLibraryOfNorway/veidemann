import {ChangeDetectionStrategy, Component, Input} from '@angular/core';

import {ExecutionMetricsComponent} from '../../../../shared/components';
import {ConfigObject, JobExecutionState, JobExecutionStatus} from '../../../../shared/models';
import {DurationFormatPipe} from '../../../../shared/pipes/duration-format.pipe';
import {jobExecutionStatePresentation} from '../../func';
import {ExecutionMetadataComponent} from '../execution-metadata/execution-metadata.component';

@Component({
  selector: 'app-job-execution-metrics-section',
  templateUrl: './job-execution-metrics-section.component.html',
  styleUrls: ['../execution-metrics-section.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ExecutionMetadataComponent,
    ExecutionMetricsComponent,
  ],
  standalone: true,
})
export class JobExecutionMetricsSectionComponent {
  private readonly durationFormat = new DurationFormatPipe();

  readonly statePresentation = jobExecutionStatePresentation;

  @Input({required: true})
  jobExecutionStatus: JobExecutionStatus;

  @Input({required: true})
  crawlJob: ConfigObject | null;

  @Input()
  queueSize: number | null | undefined = undefined;

  isRunning(): boolean {
    return this.jobExecutionStatus.state === JobExecutionState.RUNNING;
  }

  remainingBytes(): number | null {
    const limit = this.crawlJob?.crawlJob?.limits?.maxBytes;
    if (typeof limit !== 'number' || !Number.isFinite(limit) || limit <= 0) {
      return null;
    }
    return Math.max(0, limit - (this.jobExecutionStatus.bytesCrawled || 0));
  }

  remainingTime(now = new Date()): string | null {
    const limitSeconds = this.crawlJob?.crawlJob?.limits?.maxDurationS;
    if (typeof limitSeconds !== 'number' || !Number.isFinite(limitSeconds) || limitSeconds <= 0) {
      return null;
    }

    const start = this.jobExecutionStatus.startTime
      ? new Date(this.jobExecutionStatus.startTime)
      : null;
    const end = this.jobExecutionStatus.endTime
      ? new Date(this.jobExecutionStatus.endTime)
      : now;
    const elapsedSeconds = start && !Number.isNaN(start.getTime()) && !Number.isNaN(end.getTime())
      ? Math.max(0, Math.floor((end.getTime() - start.getTime()) / 1000))
      : 0;
    const remainingSeconds = Math.max(0, limitSeconds - elapsedSeconds);
    return this.durationFormat.transform(new Date(0), new Date(remainingSeconds * 1000));
  }
}
