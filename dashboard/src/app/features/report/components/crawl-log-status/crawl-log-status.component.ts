import { ChangeDetectionStrategy, Component, Input, inject } from '@angular/core';
import {ExtraStatusCodes} from '../../../../shared/models/report';
import {CrawlLog} from '../../../../shared/models/log';
import {MatTableModule} from '@angular/material/table';
import {DatePipe} from '@angular/common';
import {FileSizePipe} from '../../../../shared/pipes/filesize.pipe';
import {UrlFormatPipe} from '../../../../shared/pipes/url-format.pipe';
import {MatIcon} from '@angular/material/icon';

interface CrawlLogReportEntry {
  key: string;
  value: string;
  url?: boolean;
}

@Component({
  selector: 'app-crawl-log-status',
  templateUrl: './crawl-log-status.component.html',
  styleUrls: ['../detail-status-layout.scss', './crawl-log-status.component.css'],
  providers: [DatePipe, FileSizePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatIcon,
    MatTableModule,
    UrlFormatPipe,
  ],
  standalone: true
})
export class CrawlLogStatusComponent {
  private datePipe = inject(DatePipe);
  private fileSizePipe = inject(FileSizePipe);

  readonly ExtraStatusCodes = ExtraStatusCodes;
  crawlLogReportDisplayedColumns: string[] = ['crawlLogEntry', 'value'];

  @Input()
  crawlLog: CrawlLog;

  getReport(): CrawlLogReportEntry[] {
    const reports: (CrawlLogReportEntry | null)[] = [
      this.entry($localize`:@@crawlLogReportRequestedUri:Requested URI`, this.crawlLog.requestedUri, true),
      this.entry($localize`:@@crawlLogReportReferrer:Referrer`, this.crawlLog.referrer, true),
      this.entry($localize`:@@crawlLogReportResponseUri:Response URI`, this.crawlLog.responseUri, true),
      this.entry($localize`:@@crawlLogReportDiscoveryPath:Discovery path`, this.crawlLog.discoveryPath),
      this.entry($localize`:@@crawlLogReportStatusCode:Status code`, this.crawlLog.statusCode),
      this.entry($localize`:@@crawlLogReportCollection:Collection`, this.crawlLog.collectionFinalName),
      this.entry($localize`:@@crawlLogReportTimestamp:Timestamp`, this.datePipe.transform(this.crawlLog.timeStamp, 'long')),
      this.entry($localize`:@@crawlLogReportFetchTimestamp:Fetch timestamp`, this.datePipe.transform(this.crawlLog.fetchTimeStamp, 'long')),
      this.entry($localize`:@@crawlLogReportFetchTime:Fetch time`, this.crawlLog.fetchTimeMs / 1000),
      this.entry($localize`:@@crawlLogReportBlockDigest:Block digest`, this.crawlLog.blockDigest),
      this.entry($localize`:@@crawlLogReportPayloadDigest:Payload digest`, this.crawlLog.payloadDigest),
      this.entry($localize`:@@crawlLogReportSize:Size`, this.fileSizePipe.transform(this.crawlLog.size)),
      this.entry($localize`:@@crawlLogReportContentType:Content type`, this.crawlLog.contentType),
      this.entry($localize`:@@crawlLogReportStorageReference:Storage reference`, this.crawlLog.storageRef),
      this.entry($localize`:@@crawlLogReportRecordType:Record type`, this.crawlLog.recordType),
      this.entry($localize`:@@crawlLogReportWarcRefersTo:WARC refers to`, this.crawlLog.warcRefersTo),
      this.entry($localize`:@@crawlLogReportIpAddress:IP address`, this.crawlLog.ipAddress),
      this.entry($localize`:@@crawlLogReportRetries:Retries`, this.crawlLog.retries),
      this.entry($localize`:@@crawlLogReportMethod:Method`, this.crawlLog.method),
    ];
    return reports.filter((report): report is CrawlLogReportEntry => report !== null);
  }

  private entry(key: string, value: string | number | null | undefined, url = false): CrawlLogReportEntry | null {
    if (value === null || value === undefined || value === '') {
      return null;
    }
    return {key, value: String(value), url};
  }
}
