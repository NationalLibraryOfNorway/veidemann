import {AsyncPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {MatIcon} from '@angular/material/icon';

import {
  CrawlExecutionStatus,
  ExtraStatusCodes,
} from '../../../../shared/models/report';
import {DetailHeaderComponent} from '../../../../shared/components';
import {JobNamePipe, SeedNamePipe} from '../../pipe';
import {
  CrawlExecutionMetricsSectionComponent
} from '../crawl-execution-metrics-section/crawl-execution-metrics-section.component';

@Component({
  selector: 'app-crawl-execution-status',
  templateUrl: './crawl-execution-status.component.html',
  styleUrls: ['../detail-status-layout.scss', './crawl-execution-status.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    AsyncPipe,
    DetailHeaderComponent,
    CrawlExecutionMetricsSectionComponent,
    JobNamePipe,
    MatIcon,
    SeedNamePipe,
  ],
})
export class CrawlExecutionStatusComponent {
  readonly ExtraStatusCodes = ExtraStatusCodes;

  @Input({required: true})
  crawlExecutionStatus: CrawlExecutionStatus;

  @Input()
  queueSize: number | null = null;

  hasError(): boolean {
    const error = this.crawlExecutionStatus.error;
    return !!error && (error.code !== 0 || !!error.msg?.trim() || !!error.detail?.trim());
  }

}
