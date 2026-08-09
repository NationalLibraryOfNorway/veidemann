import {ComponentFixture, TestBed} from '@angular/core/testing';
import {Component} from '@angular/core';

import {CrawlLogStatusComponent} from './crawl-log-status.component';
import {CrawlLog} from '../../../../shared/models';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {ActivatedRoute} from '@angular/router';

@Component({
  template: `
    <app-crawl-log-status [crawlLog]="crawlLog">
      <span detailHeaderHelpers class="projected-helper">Helper</span>
    </app-crawl-log-status>
  `,
  imports: [CrawlLogStatusComponent],
  standalone: true,
})
class CrawlLogStatusHostComponent {
  crawlLog = new CrawlLog();
}

describe('CrawlLogStatusComponent', () => {
  let component: CrawlLogStatusComponent;
  let fixture: ComponentFixture<CrawlLogStatusComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        CrawlLogStatusComponent,
        CrawlLogStatusHostComponent,
      ],
      providers: [
        ...provideCoreTesting,
        {provide: ActivatedRoute, useValue: {}}
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

  it('renders all crawl-log fields in one transparent table', () => {
    fixture.componentRef.setInput('crawlLog', new CrawlLog({
      warcId: 'warc-1',
      executionId: 'crawl-execution-1',
      jobExecutionId: 'job-execution-1',
      requestedUri: 'https://example.org/request',
      responseUri: 'https://example.org/response',
      discoveryPath: 'L',
      statusCode: 200,
      collectionFinalName: 'archive_2026',
      blockDigest: 'sha1:block',
    }));
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.detail-header h1')?.textContent).toContain('Crawl log');
    expect(fixture.nativeElement.querySelector('mat-card')).toBeNull();
    const tables = fixture.nativeElement.querySelectorAll('table') as NodeListOf<HTMLTableElement>;
    expect(tables.length).toBe(1);
    expect(tables[0].closest('.table-scroll')).not.toBeNull();
    expect(getComputedStyle(tables[0]).backgroundColor).toBe('rgba(0, 0, 0, 0)');
    expect(fixture.nativeElement.textContent).toContain('Requested URI');
    expect(fixture.nativeElement.textContent).toContain('Status code');
    expect(fixture.nativeElement.textContent).toContain('200');
    expect(fixture.nativeElement.textContent).toContain('Collection');
    expect(fixture.nativeElement.textContent).toContain('archive_2026');
    expect(fixture.nativeElement.textContent).toContain('Block digest');
    expect(fixture.nativeElement.textContent).toContain('sha1:block');
    expect(fixture.nativeElement.textContent).not.toContain('Crawl execution id');
    expect(fixture.nativeElement.textContent).not.toContain('Job execution id');
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

  it('projects shortcuts into the standard page header', () => {
    const hostFixture = TestBed.createComponent(CrawlLogStatusHostComponent);
    hostFixture.detectChanges();

    expect(hostFixture.nativeElement.querySelector('.detail-header .projected-helper')).not.toBeNull();
  });
});
