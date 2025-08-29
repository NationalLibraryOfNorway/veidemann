import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {CrawlExecutionState, CrawlExecutionStatus} from '../../../../shared/models/report';
import {MatExpansionModule} from '@angular/material/expansion';
import {DatePipe} from '@angular/common';
import {NgxFilesizeModule} from 'ngx-filesize';
import {RouterLink} from '@angular/router';
import {MatListModule} from '@angular/material/list';

@Component({
  selector: 'app-config-crawl-execution-status',
  templateUrl: './crawl-execution-status.component.html',
  styleUrls: ['./crawl-execution-status.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatExpansionModule,
    MatListModule,
    DatePipe,
    NgxFilesizeModule,
    RouterLink,
  ],
  standalone: true
})
export class CrawlExecutionStatusComponent {
  readonly CrawlExecutionState = CrawlExecutionState;

  @Input()
  crawlExecutionStatus: CrawlExecutionStatus;
}
