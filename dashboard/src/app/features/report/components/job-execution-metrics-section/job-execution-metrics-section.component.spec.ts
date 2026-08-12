import {ComponentFixture, TestBed} from '@angular/core/testing';

import {
  ConfigObject,
  CrawlJob,
  JobExecutionState,
  JobExecutionStatus,
  Kind,
} from '../../../../shared/models';
import {
  JobExecutionMetricsSectionComponent
} from './job-execution-metrics-section.component';

describe('JobExecutionMetricsSectionComponent', () => {
  let fixture: ComponentFixture<JobExecutionMetricsSectionComponent>;
  const crawlJob = new ConfigObject({
    kind: Kind.CRAWLJOB,
    crawlJob: new CrawlJob({limits: {maxBytes: 5_000, maxDurationS: 7_200}}),
  });

  beforeEach(async () => {
    await TestBed.configureTestingModule({imports: [JobExecutionMetricsSectionComponent]})
      .compileComponents();
    fixture = TestBed.createComponent(JobExecutionMetricsSectionComponent);
  });

  function render(queueSize: number | null | undefined = undefined): void {
    fixture.componentRef.setInput('jobExecutionStatus', new JobExecutionStatus({
      state: JobExecutionState.RUNNING,
      desiredState: JobExecutionState.ABORTED_MANUAL,
      startTime: '2026-08-11T10:00:00.000Z',
      documentsCrawled: 12,
      urisCrawled: 20,
      bytesCrawled: 2_500,
    }));
    fixture.componentRef.setInput('crawlJob', crawlJob);
    fixture.componentRef.setInput('queueSize', queueSize);
    fixture.detectChanges();
  }

  it('renders the shared accessible lifecycle and metrics section without an omitted queue', () => {
    render();

    const section = fixture.nativeElement.querySelector('.statistics-section') as HTMLElement;
    const metadata = section.querySelector('app-execution-metadata') as HTMLElement;
    const metrics = section.querySelector('app-execution-metrics') as HTMLElement;
    expect(section.getAttribute('aria-label')).toBe('Execution metrics');
    expect(metadata.querySelector('.execution-lifecycle--metrics')).not.toBeNull();
    expect(metadata.querySelectorAll('dd')[1].textContent.trim()).toBe('Aborted manually');
    expect(metrics.textContent).not.toContain('Queue size');
    expect(metrics.textContent).toContain('Remaining bytes');
  });

  it('distinguishes an unavailable queue from an omitted queue', () => {
    render(null);

    const queueMetric = [...fixture.nativeElement.querySelectorAll('.metric')]
      .find((metric: HTMLElement) => metric.querySelector('dt')?.textContent.trim() === 'Queue size');
    expect(queueMetric?.querySelector('dd')?.textContent.trim()).toBe('Not available');
  });

  it('calculates remaining limits once for both consuming contexts', () => {
    render();
    const fixedNow = new Date('2026-08-11T11:30:00.000Z');

    expect(fixture.componentInstance.remainingBytes()).toBe(2_500);
    expect(fixture.componentInstance.remainingTime(fixedNow)).toBe('30 min');
  });
});
