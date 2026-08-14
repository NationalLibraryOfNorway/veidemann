import {Clipboard} from '@angular/cdk/clipboard';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';

import {CrawlLogStatusComponent} from './crawl-log-status.component';
import {ApiError, CrawlLog} from '../../../../shared/models';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {SnackBarService} from '../../../../core';

describe('CrawlLogStatusComponent', () => {
  let component: CrawlLogStatusComponent;
  let fixture: ComponentFixture<CrawlLogStatusComponent>;
  let can: ReturnType<typeof vi.fn>;
  let copy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    can = vi.fn(() => true);
    copy = vi.fn(() => true);
    TestBed.configureTestingModule({
      imports: [CrawlLogStatusComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {provide: AbilityServiceSignal, useValue: {can}},
        {provide: Clipboard, useValue: {copy}},
        {provide: SnackBarService, useValue: {openSnackBar: vi.fn(), openError: vi.fn()}},
      ],
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlLogStatusComponent);
    component = fixture.componentInstance;
    component.crawlLog = new CrawlLog();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('renders every crawl-log field in one header-free full-width table', () => {
    fixture.componentRef.setInput('crawlLog', new CrawlLog({
      warcId: 'warc-1',
      executionId: 'crawl-execution-1',
      jobExecutionId: 'job-execution-1',
      timeStamp: '2026-08-14T10:00:00.000Z',
      requestedUri: 'https://example.org/request',
      responseUri: 'https://example.org/response',
      discoveryPath: 'L',
      statusCode: 200,
      collectionFinalName: 'archive_2026',
      blockDigest: 'sha1:block',
      error: new ApiError({code: -7, msg: 'Failed', detail: 'Connection reset'}),
    }));
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('app-detail-header')).toBeNull();
    expect(fixture.nativeElement.querySelector('app-detail-overflow')).toBeNull();
    expect(fixture.nativeElement.querySelector('app-crawl-log-shortcuts')).toBeNull();
    expect(fixture.nativeElement.querySelector('mat-card')).toBeNull();
    const tables = fixture.nativeElement.querySelectorAll('table') as NodeListOf<HTMLTableElement>;
    expect(tables.length).toBe(1);
    expect(tables[0].closest('.table-scroll')).not.toBeNull();
    expect(getComputedStyle(tables[0]).width).toBe(getComputedStyle(tables[0].closest('.table-scroll')).width);
    expect(getComputedStyle(tables[0]).backgroundColor).toBe('rgba(0, 0, 0, 0)');
    const labels = [...tables[0].querySelectorAll('tbody .mat-column-crawlLogEntry')]
      .map((cell: HTMLElement) => cell.textContent.trim());
    expect(labels).toEqual([
      'WARC ID', 'Timestamp', 'Status code', 'Size', 'Requested URI', 'Response URI', 'Discovery path',
      'Referrer', 'Content type', 'Fetch timestamp', 'Fetch time', 'Block digest', 'Payload digest',
      'Storage reference', 'Record type', 'WARC refers to', 'IP address', 'Crawl execution ID', 'Retries',
      'Error code', 'Error message', 'Error details', 'Job execution ID', 'Collection', 'Method',
    ]);
    expect(fixture.nativeElement.textContent).toContain('-7: ILLEGAL_URI');
    expect(fixture.nativeElement.textContent).toContain('Connection reset');
    expect(fixture.nativeElement.querySelector('.error-callout')).toBeNull();
  });

  it('copies WARC ID and links both execution identifiers to their details', () => {
    fixture.componentRef.setInput('crawlLog', new CrawlLog({
      warcId: 'warc-1', executionId: 'crawl-execution-1', jobExecutionId: 'job-execution-1',
    }));
    fixture.detectChanges();

    const copyButton = fixture.nativeElement.querySelector('button[aria-label="Copy WARC ID"]') as HTMLButtonElement;
    const links = [...fixture.nativeElement.querySelectorAll('tbody a')] as HTMLAnchorElement[];
    expect(copyButton.previousElementSibling?.textContent).toBe('warc-1');
    expect(links.map(link => link.getAttribute('href'))).toEqual([
      '/report/crawlexecution/crawl-execution-1',
      '/report/jobexecution/job-execution-1',
    ]);

    copyButton.click();
    expect(copy).toHaveBeenCalledWith('warc-1');
  });

  it('keeps execution IDs as plain text without read permission', () => {
    can.mockReturnValue(false);
    fixture.componentRef.setInput('crawlLog', new CrawlLog({
      executionId: 'crawl-execution-1', jobExecutionId: 'job-execution-1',
    }));
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('tbody a')).toBeNull();
    expect(fixture.nativeElement.textContent).toContain('crawl-execution-1');
    expect(fixture.nativeElement.textContent).toContain('job-execution-1');
  });

  it('renders missing fields while preserving valid zero values', () => {
    fixture.componentRef.setInput('crawlLog', new CrawlLog({size: 0, fetchTimeMs: 0, retries: 0}));
    fixture.detectChanges();

    const values = [...fixture.nativeElement.querySelectorAll('tbody .mat-column-value')]
      .map((cell: HTMLElement) => cell.textContent.trim());
    expect(values[3]).toBe('0 B');
    expect(values[10]).toBe('0');
    expect(values[18]).toBe('0');
    expect(values.filter(value => value === 'Not available').length).toBeGreaterThan(0);
  });

  it('keeps field names on one line and lets long URI values yield', () => {
    fixture.componentRef.setInput('crawlLog', new CrawlLog({
      requestedUri: `https://example.org/${'very-long-path/'.repeat(20)}`,
      discoveryPath: 'A very long discovery path',
    }));
    fixture.detectChanges();

    const table = fixture.nativeElement.querySelector('table.report-table') as HTMLTableElement;
    const fieldCell = table.querySelector('.mat-column-crawlLogEntry') as HTMLTableCellElement;
    const valueCell = table.querySelector('.mat-column-value') as HTMLTableCellElement;

    expect(getComputedStyle(table).tableLayout).toBe('fixed');
    expect(getComputedStyle(fieldCell).whiteSpace).toBe('nowrap');
    expect(getComputedStyle(valueCell).overflowWrap).toBe('anywhere');
  });

});
