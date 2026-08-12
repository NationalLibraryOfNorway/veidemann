import {ChangeDetectionStrategy, Component, Input} from '@angular/core';

import {ExecutionMetricsComponent} from '../../../../shared/components';
import {CrawlExecutionStatus} from '../../../../shared/models';
import {crawlExecutionStatePresentation} from '../../func';
import {ExecutionMetadataComponent} from '../execution-metadata/execution-metadata.component';

@Component({
  selector: 'app-crawl-execution-metrics-section',
  templateUrl: './crawl-execution-metrics-section.component.html',
  styleUrls: ['../execution-metrics-section.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ExecutionMetadataComponent,
    ExecutionMetricsComponent,
  ],
  standalone: true,
})
export class CrawlExecutionMetricsSectionComponent {
  readonly statePresentation = crawlExecutionStatePresentation;

  @Input({required: true})
  crawlExecutionStatus: CrawlExecutionStatus;

  @Input()
  crawlJobDisplayValue = '';

  @Input()
  queueSize: number | null | undefined = undefined;
}
