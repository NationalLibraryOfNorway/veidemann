import {DecimalPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {MatIcon} from '@angular/material/icon';
import {MatTooltip} from '@angular/material/tooltip';

import {DurationFormatPipe} from '../../pipes/duration-format.pipe';
import {FileSizePipe} from '../../pipes/filesize.pipe';

// A hint on the duration, shown as a pill after the value.
export interface DurationNotice {
  label: string;
  badge: string;
}

export interface ExecutionMetricSource {
  documentsCrawled: number;
  urisCrawled: number;
  bytesCrawled: number;
  documentsOutOfScope: number;
  documentsFailed: number;
  documentsDenied: number;
  documentsRetried: number;
  startTime?: string | Date | null;
  endTime?: string | Date | null;
}

@Component({
  selector: 'app-execution-metrics',
  templateUrl: './execution-metrics.component.html',
  styleUrls: ['./execution-metrics.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [DecimalPipe, DurationFormatPipe, FileSizePipe, MatIcon, MatTooltip],
  standalone: true,
})
export class ExecutionMetricsComponent {
  @Input({required: true}) source: ExecutionMetricSource;
  @Input() queueSize: number | null | undefined = undefined;
  @Input() remainingBytes: number | null = null;
  @Input() remainingTime: string | null = null;
  @Input() durationNotice: DurationNotice | null = null;
  @Input() currentUris: number | null = null;

  get hasSecondaryMetrics(): boolean {
    return this.queueSize !== undefined
      || this.remainingBytes !== null
      || this.remainingTime !== null
      || (this.currentUris ?? 0) > 0
      || this.source.documentsOutOfScope > 0
      || this.source.documentsFailed > 0
      || this.source.documentsDenied > 0
      || this.source.documentsRetried > 0;
  }
}
