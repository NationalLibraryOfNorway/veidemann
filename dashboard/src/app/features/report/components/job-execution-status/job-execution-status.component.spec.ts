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
    const metrics = [...fixture.nativeElement.querySelectorAll('.metric')] as HTMLElement[];
    expect(metrics.map(metric => metric.querySelector('dt')?.textContent.trim()).slice(0, 4)).toEqual([
      'Documents crawled', 'URIs crawled', 'Bytes crawled', 'Duration',
    ]);
    expect(metrics.at(-1)?.querySelector('dt')?.textContent.trim()).toBe('Queue size');
    expect(metrics.at(-1)?.querySelector('dd')?.textContent.trim()).toBe('0');
    expect(metrics.length).toBe(5);
    expect(fixture.nativeElement.querySelector('.error-callout')).toBeNull();
  });

  it('places state metadata before the crawl metrics and orders optional metrics after duration', () => {
    render(new JobExecutionStatus({
      state: JobExecutionState.RUNNING,
      desiredState: JobExecutionState.ABORTED_MANUAL,
      documentsDenied: 1,
      documentsFailed: 2,
      documentsRetried: 3,
    }));

    const metadata = fixture.nativeElement.querySelector('app-execution-metadata') as HTMLElement;
    expect([...metadata.querySelectorAll('dt')].map((term: HTMLElement) => term.textContent.trim()))
      .toEqual(['Started', 'Running']);
    expect(metadata.textContent).not.toContain('Requested:');
    expect(metadata.textContent).toContain('Aborted manually');
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

  it('shows configured remaining byte and time limits and clamps exhausted limits to zero', async () => {
    job = new ConfigObject({
      id: 'job-id',
      kind: Kind.CRAWLJOB,
      meta: new Meta({name: 'Limited crawl'}),
      crawlJob: new CrawlJob({limits: {maxBytes: 2_000, maxDurationS: 3_600}}),
    });
    render(new JobExecutionStatus({
      jobId: 'job-id',
      state: JobExecutionState.RUNNING,
      bytesCrawled: 2_500,
      startTime: '2026-08-09T10:00:00.000Z',
      endTime: '2026-08-09T11:30:00.000Z',
    }));
    await fixture.whenStable();
    fixture.detectChanges();

    const metrics = [...fixture.nativeElement.querySelectorAll('.metric')]
      .map((metric: HTMLElement) => ({
        label: metric.querySelector('dt')?.textContent.trim(),
        value: metric.querySelector('dd')?.textContent.trim(),
      }));
    expect(metrics).toEqual([
      {label: 'Documents crawled', value: '0'},
      {label: 'URIs crawled', value: '0'},
      {label: 'Bytes crawled', value: '2.5 kB'},
      {label: 'Duration', value: '1 h 30 min'},
      {label: 'Queue size', value: '0'},
      {label: 'Remaining time', value: '0 s'},
      {label: 'Remaining bytes', value: '0 B'},
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

    const queueMetric = [...fixture.nativeElement.querySelectorAll('.metric')]
      .find((metric: HTMLElement) => metric.querySelector('dt')?.textContent.trim() === 'Queue size');
    expect(queueMetric?.querySelector('dd')?.textContent.trim()).toBe('Not available');
  });

  it.each([
    [JobExecutionState.RUNNING, 'Running'],
    [JobExecutionState.CREATED, 'Created'],
    [JobExecutionState.FINISHED, 'Finished'],
    [JobExecutionState.ABORTED_MANUAL, 'Aborted manually'],
    [JobExecutionState.FAILED, 'Failed'],
    [JobExecutionState.DIED, 'Died'],
    [JobExecutionState.UNDEFINED, 'Ended'],
  ])('uses the state verb in the lifecycle for state %s', (state, expectedLabel) => {
    render(new JobExecutionStatus({state}));
    const terms = fixture.nativeElement.querySelectorAll('app-execution-metadata dt');
    expect(terms[1].textContent.trim()).toBe(expectedLabel);
    expect(fixture.nativeElement.querySelector('.state-badge')).toBeNull();
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
    expect(actions.querySelector('.projected-helper')).toBeNull();
  });

  it('shows a defined desired state in the running lifecycle value', () => {
    render(new JobExecutionStatus({
      state: JobExecutionState.RUNNING,
      desiredState: JobExecutionState.UNDEFINED,
    }));
    expect(fixture.nativeElement.querySelector('app-execution-metadata').textContent)
      .not.toContain('Requested:');

    render(new JobExecutionStatus({
      state: JobExecutionState.RUNNING,
      desiredState: JobExecutionState.ABORTED_MANUAL,
    }));
    const metadata = fixture.nativeElement.querySelector('app-execution-metadata') as HTMLElement;
    expect(metadata.textContent).not.toContain('Requested:');
    expect(metadata.querySelectorAll('dd')[1].textContent.trim()).toBe('Aborted manually');
  });

  it('uses one full-width content pane and a headerless lifecycle-and-metrics section', () => {
    render(new JobExecutionStatus());
    const grid = fixture.nativeElement.querySelector('.detail-grid') as HTMLElement;
    const primary = grid.querySelector(':scope > .primary-pane') as HTMLElement;
    const statistics = primary.querySelector('.statistics-section') as HTMLElement;
    const metrics = statistics.querySelector('app-execution-metrics') as HTMLElement;
    expect(primary).not.toBeNull();
    expect(grid.querySelector(':scope > .overview-aside')).toBeNull();
    expect(statistics.querySelector('h2')).toBeNull();
    expect(statistics.getAttribute('aria-label')).toBe('Execution metrics');
    expect(primary.querySelector('app-detail-header app-execution-metadata')).toBeNull();
    expect(statistics).not.toBeNull();
    expect(primary.querySelector('.crawl-executions-section')).not.toBeNull();
    const metadata = statistics.querySelector('app-execution-metadata') as HTMLElement;
    const sharedSection = statistics.closest('app-job-execution-metrics-section') as HTMLElement;
    expect([...metadata.querySelectorAll('dt')].map((term: HTMLElement) => term.textContent.trim()))
      .toEqual(['Started', 'Ended']);
    expect(sharedSection).not.toBeNull();
    expect(metadata.compareDocumentPosition(metrics) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(metrics.querySelector('.primary-metrics')).not.toBeNull();
    expect(metrics.querySelector('.secondary-metrics')).not.toBeNull();
    expect(getComputedStyle(metadata.querySelector('.execution-lifecycle')).borderBottomStyle).toBe('solid');
    expect(metrics.querySelectorAll('.primary-metrics > .metric').length).toBe(4);
    expect(metrics.querySelector('mat-card')).toBeNull();
  });

  it.each([
    [JobExecutionState.FINISHED, 'Finished'],
    [JobExecutionState.FAILED, 'Failed'],
    [JobExecutionState.DIED, 'Died'],
    [JobExecutionState.ABORTED_MANUAL, 'Aborted manually'],
  ])('uses terminal metadata for state %s', (state, expectedText) => {
    render(new JobExecutionStatus({
      state,
      startTime: '2026-08-10T10:19:00.000Z',
      endTime: '2026-08-10T13:57:00.000Z',
    }));

    const metadata = fixture.nativeElement.querySelector('app-execution-metadata') as HTMLElement;
    expect(metadata.textContent).toContain(expectedText);
    expect(metadata.textContent).not.toContain('Not available');
  });
});
