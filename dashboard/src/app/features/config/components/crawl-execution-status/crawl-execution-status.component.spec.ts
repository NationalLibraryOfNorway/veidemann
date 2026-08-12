import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';

import {CrawlExecutionStatusComponent} from './crawl-execution-status.component';
import {ApiError, CrawlExecutionState, CrawlExecutionStatus} from '../../../../shared/models';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('CrawlExecutionStatusComponent', () => {
  let component: CrawlExecutionStatusComponent;
  let fixture: ComponentFixture<CrawlExecutionStatusComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CrawlExecutionStatusComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlExecutionStatusComponent);
    component = fixture.componentInstance;
    component.crawlExecutionStatus = new CrawlExecutionStatus({
      id: 'crawl-execution-1',
      state: CrawlExecutionState.FINISHED,
      jobId: 'crawl-job-1',
      jobExecutionId: 'job-execution-1',
      startTime: '2025-01-01T10:00:00Z',
      endTime: '2025-01-01T10:10:00Z',
      urisCrawled: 42,
      documentsCrawled: 40,
      bytesCrawled: 1024,
    });
    component.crawlJobName = 'Daily crawl';
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('renders a static execution summary with semantic KPI groups', () => {
    const summary = fixture.nativeElement.querySelector('.execution-summary') as HTMLElement;
    expect(summary.querySelector('app-crawl-execution-metrics-section')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('mat-expansion-panel')).toBeNull();
    expect(summary.querySelector('.state-badge')).toBeNull();
    expect(fixture.nativeElement.querySelector('mat-chip-set')).toBeNull();

    const timingLabels = [...summary.querySelectorAll('app-execution-metadata dt')]
      .map((label: HTMLElement) => label.textContent.trim());
    const timingValues = [...summary.querySelectorAll('app-execution-metadata dd')]
      .map((value: HTMLElement) => value.textContent.trim());
    expect(timingLabels).toEqual(['Started', 'Finished', 'Crawl job']);
    expect(timingValues[2]).toBe('Daily crawl');

    const metrics = [...summary.querySelectorAll('.metric')]
      .map((metric: HTMLElement) => ({
        label: metric.querySelector('dt')?.textContent.trim(),
        value: metric.querySelector('dd')?.textContent.trim(),
      }));
    expect(metrics).toEqual([
      {label: 'Documents crawled', value: '40'},
      {label: 'URIs crawled', value: '42'},
      {label: 'Bytes crawled', value: '1.02 kB'},
      {label: 'Duration', value: '10 min'},
    ]);
    expect(summary.querySelector('mat-card')).toBeNull();
    expect(summary.querySelector('.primary-metrics')?.tagName).toBe('DL');
    expect(summary.textContent).not.toContain('Queue size');
    expect(summary.textContent).not.toContain('Documents out of scope');
    expect(summary.textContent).not.toContain('Documents failed');
    expect(summary.textContent).not.toContain('Documents denied');
    expect(summary.textContent).not.toContain('Documents retried');

    expect(summary.textContent).not.toContain('crawl-execution-1');
    expect(summary.textContent).not.toContain('crawl-job-1');
    expect(summary.textContent).not.toContain('job-execution-1');
  });

  it('does not render a duplicate summary action', () => {
    expect(fixture.nativeElement.querySelector('.summary-actions')).toBeNull();
    expect(fixture.nativeElement.querySelector('a')).toBeNull();
  });

  it('falls back without exposing the crawl job id when the job name is unavailable', () => {
    fixture.componentRef.setInput('crawlJobName', '');
    fixture.detectChanges();

    const crawlJobValue = fixture.nativeElement.querySelectorAll('app-execution-metadata dd')[2] as HTMLElement;
    expect(crawlJobValue.textContent.trim()).toBe('Not available');
    expect(crawlJobValue.textContent).not.toContain('crawl-job-1');
  });

  it('does not expose navigation or raw identifiers inside the summary', () => {
    expect(fixture.nativeElement.querySelector('.summary-actions')).toBeNull();
    expect(fixture.nativeElement.querySelectorAll('a').length).toBe(0);
    expect(fixture.nativeElement.textContent).not.toContain('crawl-execution-1');
    expect(fixture.nativeElement.textContent).not.toContain('crawl-job-1');
    expect(fixture.nativeElement.textContent).not.toContain('job-execution-1');
  });

  it('keeps the summary action-free when the execution has no id', () => {
    fixture.componentRef.setInput('crawlExecutionStatus', new CrawlExecutionStatus({
      state: CrawlExecutionState.CREATED,
    }));
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.summary-actions')).toBeNull();
  });

  it('shows the current URI count for a live execution', () => {
    fixture.componentRef.setInput('crawlExecutionStatus', new CrawlExecutionStatus({
      ...component.crawlExecutionStatus,
      state: CrawlExecutionState.FETCHING,
      currentUriIdList: ['uri-1', 'uri-2'],
    }));
    fixture.detectChanges();

    const metrics = [...fixture.nativeElement.querySelectorAll('.metric')]
      .map((metric: HTMLElement) => ({
        label: metric.querySelector('dt')?.textContent.trim(),
        value: metric.querySelector('dd')?.textContent.trim(),
      }));
    expect(metrics.at(-1)).toEqual({label: 'Current URIs', value: '2'});
  });

  it('shows the desired state in place of Now for the latest active crawl execution', () => {
    fixture.componentRef.setInput('crawlExecutionStatus', new CrawlExecutionStatus({
      ...component.crawlExecutionStatus,
      state: CrawlExecutionState.FETCHING,
      desiredState: CrawlExecutionState.ABORTED_MANUAL,
      endTime: '',
    }));
    fixture.detectChanges();

    const metadata = fixture.nativeElement.querySelector('app-execution-metadata') as HTMLElement;
    expect(metadata.querySelectorAll('dd')[1].textContent.trim()).toBe('Aborted manually');
    expect(metadata.textContent).not.toContain('Now');
    expect(metadata.textContent).not.toContain('Requested:');
  });

  it('preserves crawl execution errors', () => {
    fixture.componentRef.setInput('crawlExecutionStatus', new CrawlExecutionStatus({
      ...component.crawlExecutionStatus,
      error: new ApiError({code: 12, msg: 'Crawl failed', detail: 'Connection lost'}),
    }));
    fixture.detectChanges();

    const errorSummary = fixture.nativeElement.querySelector('.error-summary') as HTMLElement;
    expect(errorSummary.getAttribute('role')).toBe('alert');
    expect(errorSummary.textContent).toContain('12');
    expect(errorSummary.textContent).toContain('Crawl failed');
    expect(errorSummary.textContent).toContain('Connection lost');
  });
});
