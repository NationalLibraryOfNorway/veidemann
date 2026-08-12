import {ComponentFixture, TestBed} from '@angular/core/testing';
import {Component} from '@angular/core';
import {of} from 'rxjs';

import {ApiError, ConfigObject, CrawlExecutionState, CrawlExecutionStatus, Meta} from '../../../../shared/models';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {CrawlExecutionService, JobExecutionService} from '../../services';
import {CrawlExecutionStatusComponent} from './crawl-execution-status.component';

@Component({
  template: `
    <app-crawl-execution-status [crawlExecutionStatus]="status">
      <span detailHeaderHelpers class="projected-helper">Helper</span>
      <span detailActions class="projected-action">Action</span>
    </app-crawl-execution-status>
  `,
  imports: [CrawlExecutionStatusComponent],
  standalone: true,
})
class CrawlExecutionStatusHostComponent {
  status = new CrawlExecutionStatus();
}

describe('CrawlExecutionStatusComponent', () => {
  let fixture: ComponentFixture<CrawlExecutionStatusComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CrawlExecutionStatusComponent, CrawlExecutionStatusHostComponent],
      providers: [
        ...provideCoreTesting,
        {
          provide: CrawlExecutionService,
          useValue: {
            getSeed: (id: string) => of(new ConfigObject({id, meta: new Meta({name: 'Example seed'})})),
          },
        },
        {
          provide: JobExecutionService,
          useValue: {
            getJob: (id: string) => of(new ConfigObject({id, meta: new Meta({name: 'Example job'})})),
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(CrawlExecutionStatusComponent);
  });

  function render(status: CrawlExecutionStatus): void {
    fixture.componentRef.setInput('crawlExecutionStatus', status);
    fixture.detectChanges();
  }

  it('uses the seed as heading and moves the job name into the lifecycle grid', async () => {
    render(new CrawlExecutionStatus({
      id: 'crawl-execution-id-that-can-wrap',
      seedId: 'seed-id-that-can-wrap',
      jobId: 'job-id-that-can-wrap',
      jobExecutionId: 'parent-execution-id-that-can-wrap',
      state: CrawlExecutionState.FINISHED,
    }));
    await fixture.whenStable();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('table')).toBeNull();
    expect(fixture.nativeElement.querySelector('.overview-aside')).toBeNull();
    expect(fixture.nativeElement.querySelector('h1').textContent).toBe('Example seed');
    expect(fixture.nativeElement.querySelector('.detail-header-description')).toBeNull();
    const metadata = fixture.nativeElement.querySelector('.statistics-section app-execution-metadata') as HTMLElement;
    expect([...metadata.querySelectorAll('dt')].map((term: HTMLElement) => term.textContent.trim()))
      .toEqual(['Started', 'Finished', 'Crawl job']);
    expect(metadata.querySelectorAll('dd')[2].textContent.trim()).toBe('Example job');
    expect(fixture.nativeElement.textContent).not.toContain('crawl-execution-id-that-can-wrap');
    expect(fixture.nativeElement.textContent).not.toContain('parent-execution-id-that-can-wrap');
    expect(fixture.nativeElement.querySelectorAll('.metric').length).toBe(5);
    expect(fixture.nativeElement.textContent.match(/Not available/g)?.length).toBe(3);
    expect(fixture.nativeElement.querySelector('.error-callout')).toBeNull();
  });

  it('places state metadata before the crawl metrics and orders optional metrics after duration', () => {
    render(new CrawlExecutionStatus({
      state: CrawlExecutionState.FETCHING,
      desiredState: CrawlExecutionState.ABORTED_MANUAL,
      documentsDenied: 1,
      documentsFailed: 2,
      documentsRetried: 3,
    }));

    const metadata = fixture.nativeElement.querySelector('app-execution-metadata') as HTMLElement;
    expect([...metadata.querySelectorAll('dt')].map((term: HTMLElement) => term.textContent.trim()))
      .toEqual(['Started', 'Fetching', 'Crawl job']);
    expect(metadata.textContent).not.toContain('Requested:');
    expect(metadata.textContent).toContain('Aborted manually');
    expect(metadata.textContent).not.toContain('Created');
    expect(metadata.textContent).not.toContain('Last changed');
    expect(metadata.textContent).not.toContain('Now');
    expect(metadata.querySelectorAll('dd')[1].textContent.trim()).toBe('Aborted manually');
    const metricLabels = [...fixture.nativeElement.querySelectorAll('.metric dt')]
      .map((label: HTMLElement) => label.textContent.trim());
    expect(metricLabels).toEqual([
      'Documents crawled',
      'URIs crawled',
      'Bytes crawled',
      'Duration',
      'Queue size',
      'Documents failed',
      'Documents denied',
      'Documents retried',
    ]);
  });

  it.each([
    [CrawlExecutionState.FETCHING, 'Fetching'],
    [CrawlExecutionState.CREATED, 'Created'],
    [CrawlExecutionState.SLEEPING, 'Sleeping'],
    [CrawlExecutionState.FINISHED, 'Finished'],
    [CrawlExecutionState.ABORTED_TIMEOUT, 'Aborted after timeout'],
    [CrawlExecutionState.ABORTED_SIZE, 'Aborted at size limit'],
    [CrawlExecutionState.ABORTED_MANUAL, 'Aborted manually'],
    [CrawlExecutionState.FAILED, 'Failed'],
    [CrawlExecutionState.DIED, 'Died'],
    [CrawlExecutionState.UNDEFINED, 'Ended'],
  ])('uses the state verb in the lifecycle for state %s', (state, expectedLabel) => {
    render(new CrawlExecutionStatus({state}));
    const terms = fixture.nativeElement.querySelectorAll('app-execution-metadata dt');
    expect(terms[1].textContent.trim()).toBe(expectedLabel);
    expect(fixture.nativeElement.querySelector('.state-badge')).toBeNull();
  });

  it('renders message-only errors and allows long details to retain wrapping whitespace', () => {
    render(new CrawlExecutionStatus({error: new ApiError()}));
    expect(fixture.nativeElement.querySelector('.error-callout')).toBeNull();

    render(new CrawlExecutionStatus({
      error: new ApiError({detail: 'first line\nsecond line'}),
    }));
    const detail = fixture.nativeElement.querySelector('.error-text') as HTMLElement;
    expect(detail.textContent).toContain('first line\nsecond line');
  });

  it('projects helpers and actions into their page-header rows', () => {
    const hostFixture = TestBed.createComponent(CrawlExecutionStatusHostComponent);
    hostFixture.detectChanges();

    const header = hostFixture.nativeElement.querySelector('.detail-header') as HTMLElement;
    const actions = header.querySelector('.detail-header-actions') as HTMLElement;
    expect(header.querySelector('.projected-helper')).not.toBeNull();
    expect(actions.querySelector('.projected-action')).not.toBeNull();
    expect(actions.querySelector('.projected-helper')).toBeNull();
  });

  it('uses one full-width content pane and a headerless lifecycle-and-metrics section', () => {
    render(new CrawlExecutionStatus());
    const grid = fixture.nativeElement.querySelector('.detail-grid') as HTMLElement;
    const primary = grid.querySelector(':scope > .primary-pane') as HTMLElement;
    const statistics = primary.querySelector('.statistics-section') as HTMLElement;

    expect(statistics.querySelector('h2')).toBeNull();
    expect(statistics.getAttribute('aria-label')).toBe('Execution metrics');
    expect(grid.querySelector(':scope > .overview-aside')).toBeNull();
    expect(primary.querySelector('app-detail-header app-execution-metadata')).toBeNull();
    const metadata = statistics.querySelector('app-execution-metadata') as HTMLElement;
    const metrics = statistics.querySelector('app-execution-metrics') as HTMLElement;
    expect(statistics.closest('app-crawl-execution-metrics-section')).not.toBeNull();
    expect(metadata.compareDocumentPosition(metrics) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(metrics.querySelector('.primary-metrics')).not.toBeNull();
    expect(metrics.querySelector('.secondary-metrics')).not.toBeNull();
    expect(getComputedStyle(metadata.querySelector('.execution-lifecycle')).borderBottomStyle).toBe('solid');
    expect(metrics.querySelectorAll('.primary-metrics > .metric').length).toBe(4);
    expect(metrics.querySelector('mat-card')).toBeNull();
  });

  it.each([
    [CrawlExecutionState.FINISHED, 'Finished'],
    [CrawlExecutionState.FAILED, 'Failed'],
    [CrawlExecutionState.DIED, 'Died'],
    [CrawlExecutionState.ABORTED_TIMEOUT, 'Aborted after timeout'],
    [CrawlExecutionState.ABORTED_SIZE, 'Aborted at size limit'],
    [CrawlExecutionState.ABORTED_MANUAL, 'Aborted manually'],
  ])('uses terminal metadata for state %s', (state, expectedText) => {
    render(new CrawlExecutionStatus({
      state,
      startTime: '2026-08-10T10:19:00.000Z',
      endTime: '2026-08-10T13:57:00.000Z',
    }));

    const metadata = fixture.nativeElement.querySelector('app-execution-metadata') as HTMLElement;
    expect(metadata.textContent).toContain(expectedText);
    expect(metadata.textContent).not.toContain('Not available');
  });

  it('renders a queue count and distinguishes an unavailable count from zero', () => {
    fixture.componentRef.setInput('queueSize', 17);
    render(new CrawlExecutionStatus());
    let queueMetric = [...fixture.nativeElement.querySelectorAll('.metric')]
      .find((metric: HTMLElement) => metric.querySelector('dt')?.textContent.trim() === 'Queue size');
    expect(queueMetric?.querySelector('dd')?.textContent.trim()).toBe('17');

    fixture.componentRef.setInput('queueSize', null);
    fixture.detectChanges();
    queueMetric = [...fixture.nativeElement.querySelectorAll('.metric')]
      .find((metric: HTMLElement) => metric.querySelector('dt')?.textContent.trim() === 'Queue size');
    expect(queueMetric?.querySelector('dd')?.textContent.trim()).toBe('Not available');
  });
});
