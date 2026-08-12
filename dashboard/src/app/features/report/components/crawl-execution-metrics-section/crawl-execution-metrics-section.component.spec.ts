import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlExecutionState, CrawlExecutionStatus} from '../../../../shared/models';
import {
  CrawlExecutionMetricsSectionComponent
} from './crawl-execution-metrics-section.component';

describe('CrawlExecutionMetricsSectionComponent', () => {
  let fixture: ComponentFixture<CrawlExecutionMetricsSectionComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({imports: [CrawlExecutionMetricsSectionComponent]})
      .compileComponents();
    fixture = TestBed.createComponent(CrawlExecutionMetricsSectionComponent);
  });

  function render(queueSize: number | null | undefined = undefined): void {
    fixture.componentRef.setInput('crawlExecutionStatus', new CrawlExecutionStatus({
      state: CrawlExecutionState.FETCHING,
      desiredState: CrawlExecutionState.ABORTED_MANUAL,
      startTime: '2026-08-11T10:00:00.000Z',
      documentsCrawled: 12,
      urisCrawled: 20,
      bytesCrawled: 2_500,
      currentUriIdList: ['uri-1', 'uri-2'],
    }));
    fixture.componentRef.setInput('crawlJobDisplayValue', 'Daily crawl');
    fixture.componentRef.setInput('queueSize', queueSize);
    fixture.detectChanges();
  }

  it('renders crawl-job context and current URIs without an omitted queue', () => {
    render();

    const section = fixture.nativeElement.querySelector('.statistics-section') as HTMLElement;
    const labels = [...section.querySelectorAll('app-execution-metadata dt')]
      .map((label: HTMLElement) => label.textContent.trim());
    const values = [...section.querySelectorAll('app-execution-metadata dd')]
      .map((value: HTMLElement) => value.textContent.trim());
    expect(section.getAttribute('aria-label')).toBe('Execution metrics');
    expect(labels).toEqual(['Started', 'Fetching', 'Crawl job']);
    expect(values[1]).toBe('Aborted manually');
    expect(values[2]).toBe('Daily crawl');
    expect(section.textContent).not.toContain('Queue size');
    expect(section.textContent).toContain('Current URIs');
    expect(section.textContent).toContain('2');
  });

  it('distinguishes an unavailable queue from an omitted queue', () => {
    render(null);

    const queueMetric = [...fixture.nativeElement.querySelectorAll('.metric')]
      .find((metric: HTMLElement) => metric.querySelector('dt')?.textContent.trim() === 'Queue size');
    expect(queueMetric?.querySelector('dd')?.textContent.trim()).toBe('Not available');
  });
});
