import {Component, Input} from '@angular/core';
import {CrawlLog} from '../../../../shared/models/log';
import {NgxFilesizeModule} from 'ngx-filesize';
import {UrlFormatPipe} from '../../../../shared/pipes/url-format.pipe';
import {DurationFormatPipe} from '../../../../shared/pipes/duration-format.pipe';
import {MatCardModule} from '@angular/material/card';
import {FlexLayoutModule} from '@angular/flex-layout';

@Component({
  selector: 'app-crawl-log-preview',
  templateUrl: './crawl-log-preview.component.html',
  styleUrls: ['./crawl-log-preview.component.css'],
  imports: [
    DurationFormatPipe,
    FlexLayoutModule,
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
