import {ComponentFixture, TestBed} from '@angular/core/testing';
import {By} from '@angular/platform-browser';
import {provideRouter} from '@angular/router';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {
  ConfigObject,
  CrawlJob,
  JobExecutionState,
  JobExecutionStatus,
  Kind,
} from '../../../../shared/models';
import {JobStatusComponent} from './job-status.component';
import {
  JobExecutionMetricsSectionComponent
} from '../../../report/components/job-execution-metrics-section/job-execution-metrics-section.component';

describe('JobStatusComponent', () => {
  let fixture: ComponentFixture<JobStatusComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [JobStatusComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(JobStatusComponent);
  });

  function render(
    status = new JobExecutionStatus({
      id: 'execution-1',
      state: JobExecutionState.FINISHED,
      startTime: '2026-08-09T10:00:00.000Z',
      endTime: '2026-08-09T11:30:00.000Z',
      documentsCrawled: 12,
      bytesCrawled: 2_500,
      documentsDenied: 1,
      documentsFailed: 2,
      documentsRetried: 3,
      documentsOutOfScope: 4,
      urisCrawled: 20,
    }),
    crawlJob = new ConfigObject({
      id: 'job-1',
      kind: Kind.CRAWLJOB,
      crawlJob: new CrawlJob({limits: {maxBytes: 5_000, maxDurationS: 7_200}}),
    }),
  ): void {
    fixture.componentRef.setInput('jobExecutionStatus', status);
    fixture.componentRef.setInput('crawlJob', crawlJob);
    fixture.detectChanges();
  }

  function renderedMetrics(): {label: string; value: string}[] {
    return [...fixture.nativeElement.querySelectorAll('.metric')]
      .map((metric: HTMLElement) => ({
        label: metric.querySelector('dt')?.textContent.trim() ?? '',
        value: metric.querySelector('dd')?.textContent.trim() ?? '',
      }));
  }

  it('renders a static execution summary without metric cards', () => {
    render();

    const summary = fixture.nativeElement.querySelector('.execution-summary') as HTMLElement;
    expect(summary.querySelector('app-job-execution-metrics-section')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('mat-expansion-panel')).toBeNull();
    expect([...summary.querySelectorAll('app-execution-metadata dt')]
      .map((label: HTMLElement) => label.textContent.trim())).toEqual(['Started', 'Finished']);
    expect(summary.querySelector('app-execution-metadata dd')?.textContent).toContain('Aug 9, 2026');
    expect(summary.querySelector('.state-badge')).toBeNull();
    expect(summary.querySelector('mat-card')).toBeNull();
    expect(summary.textContent).not.toContain('Queue size');
  });

  it('falls back when the execution has no start date', () => {
    render(new JobExecutionStatus({state: JobExecutionState.CREATED}));

    const summary = fixture.nativeElement.querySelector('.execution-summary') as HTMLElement;
    expect([...summary.querySelectorAll('app-execution-metadata dd')]
      .map((value: HTMLElement) => value.textContent.trim())).toEqual(['Not available', 'Now']);
  });

  it('shows the desired state in place of Now for the latest running job execution', () => {
    render(new JobExecutionStatus({
      state: JobExecutionState.RUNNING,
      desiredState: JobExecutionState.ABORTED_MANUAL,
    }));

    const metadata = fixture.nativeElement.querySelector('app-execution-metadata') as HTMLElement;
    expect(metadata.querySelectorAll('dd')[1].textContent.trim()).toBe('Aborted manually');
    expect(metadata.textContent).not.toContain('Now');
    expect(metadata.textContent).not.toContain('Requested:');
  });

  it('shows finished execution timing and outcomes without progress estimates', () => {
    render();

    expect(renderedMetrics()).toEqual([
      {label: 'Documents crawled', value: '12'},
      {label: 'URIs crawled', value: '20'},
      {label: 'Bytes crawled', value: '2.5 kB'},
      {label: 'Duration', value: '1 h 30 min'},
      {label: 'Documents out of scope', value: '4'},
      {label: 'Documents failed', value: '2'},
      {label: 'Documents denied', value: '1'},
      {label: 'Documents retried', value: '3'},
    ]);
  });

  it('groups progress estimates with related metrics only while running', () => {
    const startTime = new Date(Date.now() - 30 * 60 * 1000).toISOString();
    render(new JobExecutionStatus({
      id: 'execution-1',
      state: JobExecutionState.RUNNING,
      startTime,
      documentsCrawled: 12,
      bytesCrawled: 2_500,
      documentsOutOfScope: 4,
      urisCrawled: 20,
    }));

    const metrics = renderedMetrics();
    expect(metrics.map(metric => metric.label)).toEqual([
      'Documents crawled',
      'URIs crawled',
      'Bytes crawled',
      'Duration',
      'Remaining time',
      'Remaining bytes',
      'Documents out of scope',
    ]);
    expect(metrics.find(metric => metric.label === 'Remaining bytes')?.value).toBe('2.5 kB');

    const fixedNow = new Date(new Date(startTime).getTime() + 90 * 60 * 1000);
    const metricsSection = fixture.debugElement
      .query(By.directive(JobExecutionMetricsSectionComponent))
      .componentInstance as JobExecutionMetricsSectionComponent;
    expect(metricsSection.remainingTime(fixedNow)).toBe('30 min');
  });

  it('does not render a duplicate summary action', () => {
    render();

    expect(fixture.nativeElement.querySelector('.summary-actions')).toBeNull();
    expect(fixture.nativeElement.querySelector('a')).toBeNull();
  });
});
