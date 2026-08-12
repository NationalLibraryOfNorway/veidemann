import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {MatIcon} from '@angular/material/icon';

import {CrawlExecutionStatus} from '../../../../shared/models/report';
import {
  CrawlExecutionMetricsSectionComponent
} from '../../../report/components/crawl-execution-metrics-section/crawl-execution-metrics-section.component';

@Component({
  selector: 'app-config-crawl-execution-status',
  templateUrl: './crawl-execution-status.component.html',
  styleUrls: ['./crawl-execution-status.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CrawlExecutionMetricsSectionComponent,
    MatIcon,
  ],
  standalone: true
})
export class CrawlExecutionStatusComponent {
  @Input({required: true})
  crawlExecutionStatus: CrawlExecutionStatus;
  @Input() crawlJobName = '';

  hasError(): boolean {
    const error = this.crawlExecutionStatus.error;
    return !!error && (error.code !== 0 || !!error.msg?.trim() || !!error.detail?.trim());
  }
}
