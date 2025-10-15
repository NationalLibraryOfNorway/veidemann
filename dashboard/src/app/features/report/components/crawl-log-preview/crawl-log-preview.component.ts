import {Component, Input} from '@angular/core';
import {CrawlLog} from '../../../../shared/models/log';
import {NgxFilesizeModule} from 'ngx-filesize';
import {UrlFormatPipe} from '../../../../shared/pipes/url-format.pipe';
import {DurationFormatPipe} from '../../../../shared/pipes/duration-format.pipe';
import {MatCardModule} from '@angular/material/card';
import {FlexDirective, LayoutDirective} from '@ngbracket/ngx-layout';
import {LayoutGapDirective} from '@ngbracket/ngx-layout/flex';

@Component({
  selector: 'app-crawl-log-preview',
  templateUrl: './crawl-log-preview.component.html',
  styleUrls: ['./crawl-log-preview.component.css'],
  imports: [
    DurationFormatPipe,
    FlexDirective,
    LayoutDirective,
    LayoutGapDirective,
    MatCardModule,
    NgxFilesizeModule,
    UrlFormatPipe,

  ],
  standalone: true
})
export class CrawlLogPreviewComponent {
  @Input()
  crawlLog: CrawlLog;

  constructor() {
  }
}
