import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {EMPTY} from 'rxjs';

import {ControllerApiService, ReportApiService} from '../../../../core';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {
  ConfigObject,
  CrawlJob,
  JobExecutionState,
  JobExecutionStatus,
  Kind,
} from '../../../../shared/models';
import {JobStatusComponent} from './job-status.component';

describe('JobStatusComponent', () => {
  let fixture: ComponentFixture<JobStatusComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [JobStatusComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {
          provide: ReportApiService,
          useValue: {listCrawlExecutions: () => EMPTY},
        },
        {
          provide: ControllerApiService,
          useValue: {queueCountForCrawlExecution: vi.fn()},
        },
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

  it('shows state and start date in an initially collapsed panel header', () => {
    render();

    const panel = fixture.nativeElement.querySelector('mat-expansion-panel') as HTMLElement;
    const title = panel.querySelector('mat-panel-title') as HTMLElement;
    const description = panel.querySelector('mat-panel-description') as HTMLElement;
    expect(panel.classList.contains('mat-expanded')).toBe(false);
    expect(title.textContent.trim()).toBe('Finished');
    expect(description.textContent).toContain('Aug 9, 2026');
    expect(fixture.nativeElement.querySelector('table')).toBeNull();
  });

  it('falls back when the execution has no start date', () => {
    render(new JobExecutionStatus({state: JobExecutionState.CREATED}));

    const description = fixture.nativeElement.querySelector('mat-panel-description') as HTMLElement;
    expect(description.textContent.trim()).toBe('Not available');
  });

  it('reuses the complete crawl-statistics card set from the execution detail', () => {
    render();

    const metrics = [...fixture.nativeElement.querySelectorAll('.metric-card')]
      .map((card: HTMLElement) => ({
        label: card.querySelector('span')?.textContent.trim(),
        value: card.querySelector('strong')?.textContent.trim(),
      }));
    expect(metrics).toEqual([
      {label: 'Queue size', value: '0'},
      {label: 'Documents crawled', value: '12'},
      {label: 'Bytes crawled', value: '2.5 kB'},
      {label: 'Remaining bytes', value: '2.5 kB'},
      {label: 'Duration', value: '1 h 30 min'},
      {label: 'Remaining time', value: '30 min'},
      {label: 'Documents denied', value: '1'},
      {label: 'Documents failed', value: '2'},
      {label: 'Documents retried', value: '3'},
      {label: 'Documents out of scope', value: '4'},
      {label: 'URIs crawled', value: '20'},
    ]);
  });

  it('links to the job execution detail from the action row', () => {
    render();

    const actionRow = fixture.nativeElement.querySelector('mat-action-row') as HTMLElement;
    const link = actionRow.querySelector('a') as HTMLAnchorElement;
    expect(link.textContent.trim()).toBe('View job execution');
    expect(link.getAttribute('href')).toBe('/report/jobexecution/execution-1');
  });

  it('hides the navigation action without job-execution read permission', () => {
    fixture.componentRef.setInput('canReadJobExecution', false);
    render();

    expect(fixture.nativeElement.querySelector('mat-action-row')).toBeNull();
  });
});
