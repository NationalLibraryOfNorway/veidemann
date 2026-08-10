import {DatePipe, DecimalPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {MatExpansionModule} from '@angular/material/expansion';
import {MatIcon} from '@angular/material/icon';
import {MatChipsModule} from '@angular/material/chips';
import {MatTooltip} from '@angular/material/tooltip';
import {RouterLink} from '@angular/router';

import {CrawlExecutionStatus} from '../../../../shared/models/report';
import {FileSizePipe} from '../../../../shared/pipes/filesize.pipe';
import {crawlExecutionStatePresentation} from '../../../report/func';
import {DurationFormatPipe} from '../../../../shared/pipes/duration-format.pipe';

@Component({
  selector: 'app-config-crawl-execution-status',
  templateUrl: './crawl-execution-status.component.html',
  styleUrls: ['./crawl-execution-status.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    DatePipe,
    DecimalPipe,
    DurationFormatPipe,
    FileSizePipe,
    MatExpansionModule,
    MatIcon,
    MatChipsModule,
    MatTooltip,
    RouterLink,
  ],
  standalone: true
})
export class CrawlExecutionStatusComponent {
  readonly statePresentation = crawlExecutionStatePresentation;

  @Input({required: true})
  crawlExecutionStatus: CrawlExecutionStatus;
  @Input() crawlJobName = '';
  @Input() canReadCrawlExecution = true;
  @Input() canReadCrawlJob = true;
  @Input() canReadJobExecution = true;

  hasError(): boolean {
    const error = this.crawlExecutionStatus.error;
    return !!error && (error.code !== 0 || !!error.msg?.trim() || !!error.detail?.trim());
  }
}
