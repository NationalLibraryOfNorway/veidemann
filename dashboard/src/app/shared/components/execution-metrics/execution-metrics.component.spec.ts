import {ComponentFixture, TestBed} from '@angular/core/testing';

import {ExecutionMetricsComponent, ExecutionMetricSource} from './execution-metrics.component';

describe('ExecutionMetricsComponent', () => {
  let fixture: ComponentFixture<ExecutionMetricsComponent>;
  const source: ExecutionMetricSource = {
    documentsCrawled: 12,
    urisCrawled: 20,
    bytesCrawled: 2_500,
    documentsOutOfScope: 0,
    documentsFailed: 0,
    documentsDenied: 0,
    documentsRetried: 0,
    startTime: '2026-08-10T10:00:00Z',
    endTime: '2026-08-10T10:30:00Z',
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({imports: [ExecutionMetricsComponent]}).compileComponents();
    fixture = TestBed.createComponent(ExecutionMetricsComponent);
    fixture.componentRef.setInput('source', source);
    fixture.detectChanges();
  });

  function metrics(selector = '.metric'): {label: string; value: string}[] {
    return [...fixture.nativeElement.querySelectorAll(selector)].map((metric: HTMLElement) => ({
      label: metric.querySelector('dt')?.textContent.trim() ?? '',
      value: metric.querySelector('dd')?.textContent.trim() ?? '',
    }));
  }

  it('renders the four primary KPIs in fixed semantic order without cards', () => {
    expect(fixture.nativeElement.querySelector('.primary-metrics')?.tagName).toBe('DL');
    expect(metrics('.primary-metrics > .metric')).toEqual([
      {label: 'Documents crawled', value: '12'},
      {label: 'URIs crawled', value: '20'},
      {label: 'Bytes crawled', value: '2.5 kB'},
      {label: 'Duration', value: '30 min'},
    ]);
    expect(fixture.nativeElement.querySelector('mat-card')).toBeNull();
    expect(fixture.nativeElement.querySelector('.secondary-metrics')).toBeNull();
  });

  it('keeps zero remaining limits while omitting zero outcomes', () => {
    fixture.componentRef.setInput('remainingBytes', 0);
    fixture.componentRef.setInput('remainingTime', '0 s');
    fixture.detectChanges();

    expect(metrics('.secondary-metrics > .metric')).toEqual([
      {label: 'Remaining time', value: '0 s'},
      {label: 'Remaining bytes', value: '0 B'},
    ]);
    expect(fixture.nativeElement.textContent).not.toContain('Documents failed');
    expect(fixture.nativeElement.textContent).not.toContain('Documents denied');
    expect(fixture.nativeElement.textContent).not.toContain('Documents retried');
    expect(fixture.nativeElement.textContent).not.toContain('Documents out of scope');
  });

  it('renders queue, current URI, and non-zero outcomes as secondary metrics', () => {
    fixture.componentRef.setInput('source', {
      ...source,
      documentsOutOfScope: 1,
      documentsFailed: 2,
      documentsDenied: 3,
      documentsRetried: 4,
    });
    fixture.componentRef.setInput('queueSize', null);
    fixture.componentRef.setInput('currentUris', 5);
    fixture.detectChanges();

    expect(metrics('.secondary-metrics > .metric')).toEqual([
      {label: 'Queue size', value: 'Not available'},
      {label: 'Current URIs', value: '5'},
      {label: 'Documents out of scope', value: '1'},
      {label: 'Documents failed', value: '2'},
      {label: 'Documents denied', value: '3'},
      {label: 'Documents retried', value: '4'},
    ]);
  });
});
