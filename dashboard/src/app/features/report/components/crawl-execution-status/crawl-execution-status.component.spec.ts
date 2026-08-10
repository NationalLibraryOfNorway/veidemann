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

  it('uses the seed and job names as heading and omits execution relationship identifiers', async () => {
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
    expect(fixture.nativeElement.querySelector('.detail-subtitle').textContent).toBe('Example job');
    expect(fixture.nativeElement.textContent).not.toContain('crawl-execution-id-that-can-wrap');
    expect(fixture.nativeElement.textContent).not.toContain('parent-execution-id-that-can-wrap');
    expect(fixture.nativeElement.querySelectorAll('.metric-card').length).toBe(6);
    expect(fixture.nativeElement.textContent.match(/Not available/g)?.length).toBe(1);
    expect(fixture.nativeElement.querySelector('.error-callout')).toBeNull();
  });

  it('places state metadata in the header and orders optional crawl statistics after duration', () => {
    render(new CrawlExecutionStatus({
      state: CrawlExecutionState.FETCHING,
      desiredState: CrawlExecutionState.ABORTED_MANUAL,
      documentsDenied: 1,
      documentsFailed: 2,
      documentsRetried: 3,
    }));

    const metadata = fixture.nativeElement.querySelector('app-execution-metadata') as HTMLElement;
    const badges = [...metadata.querySelectorAll('.state-badge')]
      .map((badge: HTMLElement) => badge.querySelector(':scope > span')?.textContent.trim());
    expect(badges).toEqual(['Fetching', 'Aborted']);
    expect(metadata.textContent).toContain('Desired state:');
    expect(metadata.textContent).not.toContain('Created');
    expect(metadata.textContent).not.toContain('Last changed');
    expect(metadata.textContent).not.toContain('Not available');
    const metricLabels = [...fixture.nativeElement.querySelectorAll('.metric-card > span')]
      .map((label: HTMLElement) => label.textContent.trim());
    expect(metricLabels).toEqual([
      'Queue size',
      'Documents crawled',
      'Bytes crawled',
      'Duration',
      'Documents denied',
      'Documents failed',
      'Documents retried',
      'Documents out of scope',
      'URIs crawled',
    ]);
  });

  it.each([
    [CrawlExecutionState.FETCHING, 'state-active'],
    [CrawlExecutionState.CREATED, 'state-waiting'],
    [CrawlExecutionState.SLEEPING, 'state-waiting'],
    [CrawlExecutionState.FINISHED, 'state-finished'],
    [CrawlExecutionState.ABORTED_TIMEOUT, 'state-error'],
    [CrawlExecutionState.ABORTED_SIZE, 'state-error'],
    [CrawlExecutionState.ABORTED_MANUAL, 'state-error'],
    [CrawlExecutionState.FAILED, 'state-error'],
    [CrawlExecutionState.DIED, 'state-error'],
    [CrawlExecutionState.UNDEFINED, 'state-neutral'],
  ])('uses the semantic badge treatment for state %s', (state, expectedClass) => {
    render(new CrawlExecutionStatus({state}));

    const badge = fixture.nativeElement.querySelector('.state-badge') as HTMLElement;
    expect(badge.classList.contains(expectedClass)).toBe(true);
    expect(badge.querySelector('mat-icon')).not.toBeNull();
    expect(badge.textContent.trim().length).toBeGreaterThan(0);
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
    expect(header.querySelector('.detail-header-helpers .projected-action')).toBeNull();
  });

  it('uses one full-width content pane for the header and crawl statistics', () => {
    render(new CrawlExecutionStatus());
    const grid = fixture.nativeElement.querySelector('.detail-grid') as HTMLElement;
    const primary = grid.querySelector(':scope > .primary-pane') as HTMLElement;
    const statistics = primary.querySelector('.statistics-section') as HTMLElement;

    expect(statistics.querySelector('h2')?.textContent).toBe('Crawl statistics');
    expect(grid.querySelector(':scope > .overview-aside')).toBeNull();
    expect(primary.querySelector('.detail-header app-execution-metadata')).not.toBeNull();
    const metrics = statistics.querySelector('.metric-grid') as HTMLElement;
    expect(metrics.querySelectorAll(':scope > mat-card.metric-card').length).toBe(6);
    expect(metrics.querySelectorAll(':scope > mat-card.metric-card[appearance="filled"]').length).toBe(6);
  });

  it.each([
    [CrawlExecutionState.FINISHED, 'Ended'],
    [CrawlExecutionState.FAILED, 'Failed'],
    [CrawlExecutionState.DIED, 'Died'],
    [CrawlExecutionState.ABORTED_TIMEOUT, 'Aborted'],
    [CrawlExecutionState.ABORTED_SIZE, 'Aborted'],
    [CrawlExecutionState.ABORTED_MANUAL, 'Aborted'],
  ])('uses terminal metadata for state %s', (state, expectedText) => {
    render(new CrawlExecutionStatus({
      state,
      endTime: '2026-08-10T13:57:00.000Z',
    }));

    const metadata = fixture.nativeElement.querySelector('app-execution-metadata') as HTMLElement;
    expect(metadata.textContent).toContain(expectedText);
    expect(metadata.textContent).not.toContain('Not available');
  });

  it('renders a queue count and distinguishes an unavailable count from zero', () => {
    fixture.componentRef.setInput('queueSize', 17);
    render(new CrawlExecutionStatus());
    let queueMetric = fixture.nativeElement.querySelector('.metric-card') as HTMLElement;
    expect(queueMetric.querySelector('strong')?.textContent.trim()).toBe('17');

    fixture.componentRef.setInput('queueSize', null);
    fixture.detectChanges();
    queueMetric = fixture.nativeElement.querySelector('.metric-card') as HTMLElement;
    expect(queueMetric.querySelector('strong')?.textContent.trim()).toBe('Not available');
  });
});
