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
    expect(fixture.nativeElement.querySelector('dl.description-list')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('h1').textContent).toBe('Example seed');
    expect(fixture.nativeElement.querySelector('.detail-subtitle').textContent).toBe('Example job');
    expect(fixture.nativeElement.textContent).not.toContain('crawl-execution-id-that-can-wrap');
    expect(fixture.nativeElement.textContent).not.toContain('parent-execution-id-that-can-wrap');
    expect(fixture.nativeElement.querySelectorAll('.metric-card').length).toBe(6);
    expect(fixture.nativeElement.textContent.match(/Not available/g)?.length).toBe(5);
    expect(fixture.nativeElement.querySelector('.error-callout')).toBeNull();
  });

  it('pairs states and dates and orders optional crawl statistics after duration', () => {
    render(new CrawlExecutionStatus({
      state: CrawlExecutionState.FETCHING,
      desiredState: CrawlExecutionState.ABORTED_MANUAL,
      documentsDenied: 1,
      documentsFailed: 2,
      documentsRetried: 3,
    }));

    const rows = [...fixture.nativeElement.querySelectorAll('.overview-card .description-row')]
      .map((row: HTMLElement) => [...row.querySelectorAll('dt')]
        .map(term => term.textContent.trim()));
    expect(rows).toEqual([
      ['State', 'Desired state'],
      ['Created', 'Last changed'],
      ['Started', 'Ended'],
    ]);
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

  it('projects helpers into the page header and actions into the overview card actions', () => {
    const hostFixture = TestBed.createComponent(CrawlExecutionStatusHostComponent);
    hostFixture.detectChanges();

    const header = hostFixture.nativeElement.querySelector('.detail-header') as HTMLElement;
    const cardActions = hostFixture.nativeElement.querySelector('.overview-card mat-card-actions') as HTMLElement;
    expect(header.querySelector('.projected-helper')).not.toBeNull();
    expect(header.querySelector('.projected-action')).toBeNull();
    expect(cardActions.querySelector('.projected-action')).not.toBeNull();
  });

  it('places crawl statistics in the primary pane and textual state in a filled supporting card', () => {
    render(new CrawlExecutionStatus());
    const grid = fixture.nativeElement.querySelector('.detail-grid') as HTMLElement;
    const primary = grid.querySelector(':scope > .primary-pane') as HTMLElement;
    const statistics = primary.querySelector('.statistics-section') as HTMLElement;
    const aside = grid.querySelector(':scope > .overview-aside') as HTMLElement;
    const overviewCard = aside.querySelector(':scope > mat-card.overview-card') as HTMLElement;

    expect(statistics.querySelector('h2')?.textContent).toBe('Crawl statistics');
    expect(aside.tagName).toBe('ASIDE');
    expect(overviewCard.getAttribute('appearance')).toBe('filled');
    expect(overviewCard.querySelector('dl.description-list')).not.toBeNull();
    expect(overviewCard.querySelector('mat-card-actions.detail-actions')).not.toBeNull();
    const metrics = statistics.querySelector('.metric-grid') as HTMLElement;
    expect(metrics.querySelectorAll(':scope > mat-card.metric-card').length).toBe(6);
    expect(metrics.querySelectorAll(':scope > mat-card.metric-card[appearance="filled"]').length).toBe(6);
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
