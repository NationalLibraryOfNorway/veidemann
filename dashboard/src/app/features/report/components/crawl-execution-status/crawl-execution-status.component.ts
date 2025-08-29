import { ChangeDetectionStrategy, Component, Input, OnInit } from '@angular/core';
import { MatCardModule } from '@angular/material/card';
import { MatTableDataSource, MatTableModule } from '@angular/material/table';
import { FileSizePipe } from 'ngx-filesize';
import { CrawlExecutionState, CrawlExecutionStatus, ExtraStatusCodes } from '../../../../shared/models/report';
import {RouterLink} from '@angular/router';
import {JobNamePipe, SeedNamePipe} from '../../pipe';
import {AsyncPipe, DatePipe} from '@angular/common';
import {MatIcon} from '@angular/material/icon';
import {FlexLayoutModule} from '@angular/flex-layout';

@Component({
    selector: 'app-crawl-execution-status',
    templateUrl: './crawl-execution-status.component.html',
    styleUrls: ['./crawl-execution-status.component.scss'],
    providers: [FileSizePipe],
    changeDetection: ChangeDetectionStrategy.OnPush,
    standalone: true,
  imports: [
    AsyncPipe,
    DatePipe,
    FlexLayoutModule,
    JobNamePipe,
    MatCardModule,
    MatIcon,
    MatTableModule,
    RouterLink,
    SeedNamePipe
  ]
})
export class CrawlExecutionStatusComponent implements OnInit{
  readonly CrawlExecutionState = CrawlExecutionState;
  readonly ExtraStatusCodes = ExtraStatusCodes;

  @Input()
  crawlExecutionStatus: CrawlExecutionStatus;

  dataSource = new MatTableDataSource<CrawlExecutionStatus>();
  crawlExecDisplayedColumns: string[] = ['jobExecution', 'job', 'state'];
  crawlExecRuntimeDisplayedColumns: string[] = ['createdTime', 'startTime', 'endTime', 'lastChangeTime'];
  crawlExecStatisticsDisplayedColumns: string[] = ['statistics', 'count'];

  constructor(private fileSizePipe: FileSizePipe) {
  }

  ngOnInit(): void {
    this.dataSource = new MatTableDataSource<CrawlExecutionStatus>([this.crawlExecutionStatus]);
  }

  getStatistics(){
    const datasource = [];
    const stats = [
      {stat:'URIs crawled', count: this.crawlExecutionStatus.urisCrawled},
      {stat:'Bytes crawled', count: this.fileSizePipe.transform(this.crawlExecutionStatus.bytesCrawled)},
      {stat:'Documents crawled', count: this.crawlExecutionStatus.documentsCrawled},
      {stat:'Documents denied', count: this.crawlExecutionStatus.documentsDenied},
      {stat:'Documents failed', count: this.crawlExecutionStatus.documentsFailed},
      {stat:'Documents out of scope', count: this.crawlExecutionStatus.documentsOutOfScope},
      {stat:'Documents retried', count: this.crawlExecutionStatus.documentsRetried}
    ];
    for (let stat of stats) {
      if (stat.count !== 0) {
        datasource.push(stat);
      }
    }
    return datasource;
  }

}
