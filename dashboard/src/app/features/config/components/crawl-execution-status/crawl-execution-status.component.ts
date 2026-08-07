import {DatePipe, DecimalPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {MatButtonModule} from '@angular/material/button';
import {MatCardModule} from '@angular/material/card';
import {MatIcon} from '@angular/material/icon';
import {RouterLink} from '@angular/router';

import {CrawlExecutionStatus} from '../../../../shared/models/report';
import {FileSizePipe} from '../../../../shared/pipes/filesize.pipe';
import {crawlExecutionStatePresentation} from '../../../report/func';

@Component({
  selector: 'app-config-crawl-execution-status',
  templateUrl: './crawl-execution-status.component.html',
  styleUrls: ['./crawl-execution-status.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    DatePipe,
    DecimalPipe,
    FileSizePipe,
    MatButtonModule,
    MatCardModule,
    MatIcon,
    RouterLink,
  ],
  standalone: true
})
export class CrawlExecutionStatusComponent {
  readonly statePresentation = crawlExecutionStatePresentation;

  @Input({required: true})
  crawlExecutionStatus: CrawlExecutionStatus;

  hasError(): boolean {
    const error = this.crawlExecutionStatus.error;
    return !!error && (error.code !== 0 || !!error.msg?.trim() || !!error.detail?.trim());
  }
}
