import {AsyncPipe, DatePipe, DecimalPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {MatCardModule} from '@angular/material/card';
import {MatIcon} from '@angular/material/icon';

import {CrawlExecutionStatus, ExtraStatusCodes} from '../../../../shared/models/report';
import {FileSizePipe} from '../../../../shared/pipes/filesize.pipe';
import {DurationFormatPipe} from '../../../../shared/pipes/duration-format.pipe';
import {crawlExecutionStatePresentation} from '../../func';
import {JobNamePipe, SeedNamePipe} from '../../pipe';

@Component({
  selector: 'app-crawl-execution-status',
  templateUrl: './crawl-execution-status.component.html',
  styleUrls: ['../detail-status-layout.scss', './crawl-execution-status.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    AsyncPipe,
    DatePipe,
    DecimalPipe,
    DurationFormatPipe,
    FileSizePipe,
    JobNamePipe,
    MatCardModule,
    MatIcon,
    SeedNamePipe,
  ],
})
export class CrawlExecutionStatusComponent {
  readonly ExtraStatusCodes = ExtraStatusCodes;
  readonly statePresentation = crawlExecutionStatePresentation;

  @Input({required: true})
  crawlExecutionStatus: CrawlExecutionStatus;

  hasError(): boolean {
    const error = this.crawlExecutionStatus.error;
    return !!error && (error.code !== 0 || !!error.msg?.trim() || !!error.detail?.trim());
  }
}
