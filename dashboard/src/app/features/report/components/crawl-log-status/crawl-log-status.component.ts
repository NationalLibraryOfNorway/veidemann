import {ChangeDetectionStrategy, Component, Input, inject} from '@angular/core';
import {MatButtonModule} from '@angular/material/button';
import {MatIcon} from '@angular/material/icon';
import {ExtraStatusCodes} from '../../../../shared/models/report';
import {CrawlLog} from '../../../../shared/models/log';
import {MatTableModule} from '@angular/material/table';
import {DatePipe} from '@angular/common';
import {RouterLink} from '@angular/router';
import {MatTooltipModule} from '@angular/material/tooltip';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {FileSizePipe} from '../../../../shared/pipes/filesize.pipe';
import {UrlFormatPipe} from '../../../../shared/pipes/url-format.pipe';
import {CopyIdDirective} from '../../../../shared/directives';

type CrawlLogReportPresentation = 'plain' | 'url' | 'copy' | 'crawlExecution' | 'jobExecution';

interface CrawlLogReportEntry {
  key: string;
  value: string;
  presentation: CrawlLogReportPresentation;
  actionValue: string;
}

@Component({
  selector: 'app-crawl-log-status',
  templateUrl: './crawl-log-status.component.html',
  styleUrls: ['../detail-status-layout.scss', './crawl-log-status.component.css'],
  providers: [DatePipe, FileSizePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CopyIdDirective,
    MatButtonModule,
    MatIcon,
    MatTableModule,
    MatTooltipModule,
    RouterLink,
    UrlFormatPipe,
  ],
  standalone: true
})
export class CrawlLogStatusComponent {
  private readonly datePipe = inject(DatePipe);
  private readonly fileSizePipe = inject(FileSizePipe);
  private readonly abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  readonly crawlLogReportDisplayedColumns: string[] = ['crawlLogEntry', 'value'];
  readonly notAvailable = $localize`:@@commonNotAvailable:Not available`;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  @Input()
  crawlLog: CrawlLog;

  constructor() {
    this.can = this.abilityService.can;
  }

  getReport(): CrawlLogReportEntry[] {
    const error = this.crawlLog.error;
    const errorCode = error?.code
      ? `${error.code}: ${ExtraStatusCodes[error.code] ?? this.notAvailable}`
      : null;
    return [
      this.entry($localize`:@@crawlLogReportWarcId:WARC ID`, this.crawlLog.warcId, 'copy'),
      this.entry($localize`:@@crawlLogReportTimestamp:Timestamp`,
        this.datePipe.transform(this.crawlLog.timeStamp, 'long')),
      this.entry($localize`:@@crawlLogReportStatusCode:Status code`, this.crawlLog.statusCode || null),
      this.entry($localize`:@@crawlLogReportSize:Size`, this.fileSizePipe.transform(this.crawlLog.size)),
      this.entry($localize`:@@crawlLogReportRequestedUri:Requested URI`, this.crawlLog.requestedUri, 'url'),
      this.entry($localize`:@@crawlLogReportResponseUri:Response URI`, this.crawlLog.responseUri, 'url'),
      this.entry($localize`:@@crawlLogReportDiscoveryPath:Discovery path`, this.crawlLog.discoveryPath),
      this.entry($localize`:@@crawlLogReportReferrer:Referrer`, this.crawlLog.referrer, 'url'),
      this.entry($localize`:@@crawlLogReportContentType:Content type`, this.crawlLog.contentType),
      this.entry($localize`:@@crawlLogReportFetchTimestamp:Fetch timestamp`,
        this.datePipe.transform(this.crawlLog.fetchTimeStamp, 'long')),
      this.entry($localize`:@@crawlLogReportFetchTime:Fetch time`, this.crawlLog.fetchTimeMs / 1000),
      this.entry($localize`:@@crawlLogReportBlockDigest:Block digest`, this.crawlLog.blockDigest),
      this.entry($localize`:@@crawlLogReportPayloadDigest:Payload digest`, this.crawlLog.payloadDigest),
      this.entry($localize`:@@crawlLogReportStorageReference:Storage reference`, this.crawlLog.storageRef),
      this.entry($localize`:@@crawlLogReportRecordType:Record type`, this.crawlLog.recordType),
      this.entry($localize`:@@crawlLogReportWarcRefersTo:WARC refers to`, this.crawlLog.warcRefersTo),
      this.entry($localize`:@@crawlLogReportIpAddress:IP address`, this.crawlLog.ipAddress),
      this.entry($localize`:@@crawlLogReportExecutionId:Crawl execution ID`,
        this.crawlLog.executionId, 'crawlExecution'),
      this.entry($localize`:@@crawlLogReportRetries:Retries`, this.crawlLog.retries),
      this.entry($localize`:@@crawlLogReportErrorCode:Error code`, errorCode),
      this.entry($localize`:@@crawlLogReportErrorMessage:Error message`, error?.msg),
      this.entry($localize`:@@crawlLogReportErrorDetails:Error details`, error?.detail),
      this.entry($localize`:@@crawlLogReportJobExecutionId:Job execution ID`,
        this.crawlLog.jobExecutionId, 'jobExecution'),
      this.entry($localize`:@@crawlLogReportCollection:Collection`, this.crawlLog.collectionFinalName),
      this.entry($localize`:@@crawlLogReportMethod:Method`, this.crawlLog.method),
    ];
  }

  private entry(
    key: string,
    value: string | number | null | undefined,
    presentation: CrawlLogReportPresentation = 'plain'
  ): CrawlLogReportEntry {
    const actionValue = value === null || value === undefined || value === '' ? '' : String(value);
    return {key, value: actionValue || this.notAvailable, presentation, actionValue};
  }
}
