import {ComponentFixture, TestBed} from '@angular/core/testing';
import {Component} from '@angular/core';
import {provideRouter} from '@angular/router';
import {of} from 'rxjs';

import {
  ApiError,
  ConfigObject,
  CrawlJob,
  CrawlExecutionState,
  JobExecutionState,
  JobExecutionStatus,
  Kind,
  Meta
} from '../../../../shared/models';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {JobExecutionService} from '../../services';
import {JobExecutionStatusComponent} from './job-execution-status.component';

@Component({
  template: `
    <app-job-execution-status [jobExecutionStatus]="status">
      <span detailHeaderHelpers class="projected-helper">Helper</span>
      <span detailActions class="projected-action">Action</span>
    </app-job-execution-status>
  `,
  imports: [JobExecutionStatusComponent],
  standalone: true,
})
class JobExecutionStatusHostComponent {
  status = new JobExecutionStatus();
}

describe('JobExecutionStatusComponent', () => {
  let fixture: ComponentFixture<JobExecutionStatusComponent>;
  let job: ConfigObject;

  beforeEach(async () => {
    job = new ConfigObject({id: 'job-id', meta: new Meta({name: 'News crawl'})});
    await TestBed.configureTestingModule({
      imports: [JobExecutionStatusComponent, JobExecutionStatusHostComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {
          provide: JobExecutionService,
          useValue: {
            getJob: () => of(job),
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(JobExecutionStatusComponent);
  });

  function render(status: JobExecutionStatus, queueSize: number | null = 0): void {
    fixture.componentRef.setInput('jobExecutionStatus', status);
    fixture.componentRef.setInput('queueSize', queueSize);
    fixture.detectChanges();
  }

  it('uses the job name as title and places queue size first in crawl statistics', async () => {
    render(new JobExecutionStatus({
      id: 'execution-id-that-can-wrap',
      jobId: 'job-id-that-can-wrap',
      state: JobExecutionState.FINISHED,
    }));
    await fixture.whenStable();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('table')).toBeNull();
    expect(fixture.nativeElement.querySelector('.overview-aside')).toBeNull();
    expect(fixture.nativeElement.textContent).not.toContain('execution-id-that-can-wrap');
    expect(fixture.nativeElement.querySelector('h1').textContent).toBe('News crawl');
    expect(fixture.nativeElement.querySelector('.queue-badge')).toBeNull();
    const firstMetric = fixture.nativeElement.querySelector('.metric-card') as HTMLElement;
    expect(firstMetric.querySelector('span')?.textContent).toBe('Queue size');
    expect(firstMetric.querySelector('strong')?.textContent.trim()).toBe('0');
    expect(fixture.nativeElement.querySelectorAll('.metric-card').length).toBe(6);
    expect(fixture.nativeElement.querySelector('.error-callout')).toBeNull();
  });

  it('places state metadata in the header and orders optional crawl statistics after duration', () => {
    render(new JobExecutionStatus({
      state: JobExecutionState.RUNNING,
      desiredState: JobExecutionState.ABORTED_MANUAL,
      documentsDenied: 1,
      documentsFailed: 2,
      documentsRetried: 3,
    }));

    const metadata = fixture.nativeElement.querySelector('app-execution-metadata') as HTMLElement;
    const badges = [...metadata.querySelectorAll('.state-badge')]
      .map((badge: HTMLElement) => badge.querySelector(':scope > span')?.textContent.trim());
    expect(badges).toEqual(['Running', 'Aborted']);
    expect(metadata.textContent).toContain('Desired state:');
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

  it('shows configured remaining byte and time limits and clamps exhausted limits to zero', async () => {
    job = new ConfigObject({
      id: 'job-id',
      kind: Kind.CRAWLJOB,
      meta: new Meta({name: 'Limited crawl'}),
      crawlJob: new CrawlJob({limits: {maxBytes: 2_000, maxDurationS: 3_600}}),
    });
    render(new JobExecutionStatus({
      jobId: 'job-id',
      bytesCrawled: 2_500,
      startTime: '2026-08-09T10:00:00.000Z',
      endTime: '2026-08-09T11:30:00.000Z',
    }));
    await fixture.whenStable();
    fixture.detectChanges();

    const metrics = [...fixture.nativeElement.querySelectorAll('.metric-card')]
      .map((card: HTMLElement) => ({
        label: card.querySelector('span')?.textContent.trim(),
        value: card.querySelector('strong')?.textContent.trim(),
      }));
    expect(metrics).toEqual([
      {label: 'Queue size', value: '0'},
      {label: 'Documents crawled', value: '0'},
      {label: 'Bytes crawled', value: '2.5 kB'},
      {label: 'Remaining bytes', value: '0 B'},
      {label: 'Duration', value: '1 h 30 min'},
      {label: 'Remaining time', value: '0 s'},
      {label: 'Documents out of scope', value: '0'},
      {label: 'URIs crawled', value: '0'},
    ]);
  });

  it('shows compact state links with the job and execution filters', () => {
    render(new JobExecutionStatus({
      id: 'job-execution-1',
      jobId: 'job-1',
      state: JobExecutionState.FINISHED,
      executionsStateMap: new Map([
        [CrawlExecutionState[CrawlExecutionState.FETCHING], 3],
        [CrawlExecutionState[CrawlExecutionState.FINISHED], 7],
      ]),
    }));

    const links = fixture.nativeElement.querySelectorAll('.state-count-list a') as NodeListOf<HTMLAnchorElement>;
    expect(links.length).toBe(2);
    expect(links[0].getAttribute('href')).toContain('job_id=job-1');
    expect(links[0].getAttribute('href')).toContain('job_execution_id=job-execution-1');
    expect(links[0].getAttribute('href')).toContain(`state=${CrawlExecutionState.FETCHING}`);
  });

  it('renders an unavailable queue count distinctly from zero', () => {
    render(new JobExecutionStatus(), null);

    const queueMetric = fixture.nativeElement.querySelector('.metric-card') as HTMLElement;
    expect(queueMetric.querySelector('strong')?.textContent.trim()).toBe('Not available');
  });

  it.each([
    [JobExecutionState.RUNNING, 'state-active'],
    [JobExecutionState.CREATED, 'state-waiting'],
    [JobExecutionState.FINISHED, 'state-finished'],
    [JobExecutionState.ABORTED_MANUAL, 'state-error'],
    [JobExecutionState.FAILED, 'state-error'],
    [JobExecutionState.DIED, 'state-error'],
    [JobExecutionState.UNDEFINED, 'state-neutral'],
  ])('uses the semantic badge treatment for state %s', (state, expectedClass) => {
    render(new JobExecutionStatus({state}));

    const badge = fixture.nativeElement.querySelector('.state-badge') as HTMLElement;
    expect(badge.classList.contains(expectedClass)).toBe(true);
    expect(badge.querySelector('mat-icon')).not.toBeNull();
    expect(badge.textContent.trim().length).toBeGreaterThan(0);
  });

  it('renders an error callout only when the error contains meaningful data', () => {
    render(new JobExecutionStatus({error: new ApiError()}));
    expect(fixture.nativeElement.querySelector('.error-callout')).toBeNull();

    render(new JobExecutionStatus({
      error: new ApiError({msg: 'A long failure message', detail: 'A long\nwrapped detail'}),
    }));
    expect(fixture.nativeElement.querySelector('.error-callout')).not.toBeNull();
    expect(fixture.nativeElement.textContent).toContain('A long\nwrapped detail');
  });

  it('projects helpers and actions into their page-header rows', () => {
    const hostFixture = TestBed.createComponent(JobExecutionStatusHostComponent);
    hostFixture.detectChanges();

    const header = hostFixture.nativeElement.querySelector('.detail-header') as HTMLElement;
    const actions = header.querySelector('.detail-header-actions') as HTMLElement;
    expect(header.querySelector('.projected-helper')).not.toBeNull();
    expect(actions.querySelector('.projected-action')).not.toBeNull();
    expect(header.querySelector('.detail-header-helpers .projected-action')).toBeNull();
  });

  it('shows desired state only when it is defined', () => {
    render(new JobExecutionStatus({desiredState: JobExecutionState.UNDEFINED}));
    expect(fixture.nativeElement.querySelector('app-execution-metadata').textContent)
      .not.toContain('Desired state');

    render(new JobExecutionStatus({desiredState: JobExecutionState.ABORTED_MANUAL}));
    const metadata = fixture.nativeElement.querySelector('app-execution-metadata') as HTMLElement;
    expect(metadata.textContent).toContain('Desired state');
    expect(metadata.textContent).toContain('Aborted');
  });

  it('uses one full-width content pane for the header and crawl statistics', () => {
    render(new JobExecutionStatus());
    const grid = fixture.nativeElement.querySelector('.detail-grid') as HTMLElement;
    const primary = grid.querySelector(':scope > .primary-pane') as HTMLElement;
    const statistics = primary.querySelector('.statistics-section') as HTMLElement;
    const metrics = statistics.querySelector('.metric-grid') as HTMLElement;
    expect(primary).not.toBeNull();
    expect(grid.querySelector(':scope > .overview-aside')).toBeNull();
    expect(statistics.querySelector('h2')?.textContent).toBe('Crawl statistics');
    expect(primary.querySelector('.detail-header app-execution-metadata')).not.toBeNull();
    expect(statistics).not.toBeNull();
    expect(primary.querySelector('.crawl-executions-section')).not.toBeNull();
    expect(metrics.querySelectorAll(':scope > mat-card.metric-card').length).toBe(6);
    expect(metrics.querySelectorAll(':scope > mat-card.metric-card[appearance="filled"]').length).toBe(6);
  });

  it.each([
    [JobExecutionState.FINISHED, 'Ended'],
    [JobExecutionState.FAILED, 'Failed'],
    [JobExecutionState.DIED, 'Died'],
    [JobExecutionState.ABORTED_MANUAL, 'Aborted'],
  ])('uses terminal metadata for state %s', (state, expectedText) => {
    render(new JobExecutionStatus({
      state,
      endTime: '2026-08-10T13:57:00.000Z',
    }));

    const metadata = fixture.nativeElement.querySelector('app-execution-metadata') as HTMLElement;
    expect(metadata.textContent).toContain(expectedText);
    expect(metadata.textContent).not.toContain('Not available');
  });
});
