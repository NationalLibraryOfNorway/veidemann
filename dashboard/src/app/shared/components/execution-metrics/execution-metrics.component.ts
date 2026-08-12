import {DecimalPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, Input} from '@angular/core';

import {DurationFormatPipe} from '../../pipes/duration-format.pipe';
import {FileSizePipe} from '../../pipes/filesize.pipe';

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
  imports: [DecimalPipe, DurationFormatPipe, FileSizePipe],
  standalone: true,
})
export class ExecutionMetricsComponent {
  @Input({required: true}) source: ExecutionMetricSource;
  @Input() queueSize: number | null | undefined = undefined;
  @Input() remainingBytes: number | null = null;
  @Input() remainingTime: string | null = null;
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
