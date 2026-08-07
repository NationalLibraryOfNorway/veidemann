import {ComponentFixture, TestBed} from '@angular/core/testing';
import {Component} from '@angular/core';
import {provideRouter} from '@angular/router';
import {of} from 'rxjs';

import {ControllerApiService, ReportApiService} from '../../../../core';
import {
  ApiError,
  ConfigObject,
  CrawlExecutionState,
  JobExecutionState,
  JobExecutionStatus,
  Meta
} from '../../../../shared/models';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {JobExecutionService} from '../../services';
import {JobExecutionStatusComponent} from './job-execution-status.component';

@Component({
  template: `
    <app-job-execution-status [jobExecutionStatus]="status">
      <span cardHeaderHelpers class="projected-helper">Helper</span>
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

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [JobExecutionStatusComponent, JobExecutionStatusHostComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {
          provide: JobExecutionService,
          useValue: {
            getJob: (id: string) => of(new ConfigObject({id, meta: new Meta({name: 'News crawl'})})),
          },
        },
        {
          provide: ControllerApiService,
          useValue: {queueCountForCrawlExecution: () => of({count: 0})},
        },
        {
          provide: ReportApiService,
          useValue: {listCrawlExecutions: () => of()},
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(JobExecutionStatusComponent);
  });

  function render(status: JobExecutionStatus): void {
    fixture.componentRef.setInput('jobExecutionStatus', status);
    fixture.detectChanges();
  }

  it('uses semantic descriptions and always renders all seven metrics, including zeros', async () => {
    render(new JobExecutionStatus({
      id: 'execution-id-that-can-wrap',
      jobId: 'job-id-that-can-wrap',
      state: JobExecutionState.FINISHED,
    }));
    await fixture.whenStable();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('table')).toBeNull();
    expect(fixture.nativeElement.querySelector('dl.description-list')).not.toBeNull();
    expect(fixture.nativeElement.textContent).toContain('execution-id-that-can-wrap');
    expect(fixture.nativeElement.textContent).toContain('News crawl');
    expect(fixture.nativeElement.querySelectorAll('.metric-tile').length).toBe(7);
    expect(fixture.nativeElement.querySelectorAll('.metric-tile strong').length).toBe(7);
    expect(fixture.nativeElement.querySelector('.error-card')).toBeNull();
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

  it('renders an error card only when the error contains meaningful data', () => {
    render(new JobExecutionStatus({error: new ApiError()}));
    expect(fixture.nativeElement.querySelector('.error-card')).toBeNull();

    render(new JobExecutionStatus({
      error: new ApiError({msg: 'A long failure message', detail: 'A long\nwrapped detail'}),
    }));
    expect(fixture.nativeElement.querySelector('.error-card')).not.toBeNull();
    expect(fixture.nativeElement.textContent).toContain('A long\nwrapped detail');
  });

  it('projects helpers into the overview header and destructive actions below the card', () => {
    const hostFixture = TestBed.createComponent(JobExecutionStatusHostComponent);
    hostFixture.detectChanges();

    const header = hostFixture.nativeElement.querySelector('.overview-card mat-card-header') as HTMLElement;
    expect(header.querySelector('.projected-helper')).not.toBeNull();
    expect(header.querySelector('.projected-action')).toBeNull();
    expect(hostFixture.nativeElement.querySelector('.detail-actions .projected-action')).not.toBeNull();
  });
});
