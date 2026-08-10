import {ErrorHandler} from '@angular/core';
import {TestBed} from '@angular/core/testing';
import {MatDialog} from '@angular/material/dialog';
import {ActivatedRoute, convertToParamMap} from '@angular/router';
import {BehaviorSubject, EMPTY, firstValueFrom, of} from 'rxjs';

import {ControllerApiService, ReportApiService, SnackBarService} from '../../../../core';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {CrawlExecutionStatus, JobExecutionState, JobExecutionStatus} from '../../../../shared/models';
import {JobExecutionService} from '../../services';
import {JobExecutionDetailComponent} from './job-execution-detail.component';

describe('JobExecutionDetailComponent', () => {
  const jobStatus = new JobExecutionStatus({
    id: 'job-execution-1',
    jobId: 'job-1',
    state: JobExecutionState.RUNNING,
  });
  let listCrawlExecutionsUnchecked: ReturnType<typeof vi.fn>;
  let queueCountForCrawlExecutions: ReturnType<typeof vi.fn>;
  let queueCountForCrawlExecution: ReturnType<typeof vi.fn>;

  beforeEach(async () => {
    listCrawlExecutionsUnchecked = vi.fn(() => of(
      new CrawlExecutionStatus({id: 'crawl-1'}),
      new CrawlExecutionStatus({id: 'crawl-1'}),
      new CrawlExecutionStatus({id: 'crawl-2'}),
    ));
    queueCountForCrawlExecutions = vi.fn(() => of({count: 23}));
    queueCountForCrawlExecution = vi.fn();

    await TestBed.configureTestingModule({
      imports: [JobExecutionDetailComponent],
      providers: [
        ...provideCoreTesting,
        {
          provide: ActivatedRoute,
          useValue: {
            paramMap: new BehaviorSubject(convertToParamMap({id: jobStatus.id})),
            queryParamMap: new BehaviorSubject(convertToParamMap({watch: 'false'})),
          },
        },
        {
          provide: JobExecutionService,
          useValue: {get: vi.fn(() => of(jobStatus))},
        },
        {
          provide: ReportApiService,
          useValue: {listCrawlExecutionsUnchecked},
        },
        {
          provide: ControllerApiService,
          useValue: {queueCountForCrawlExecutions, queueCountForCrawlExecution},
        },
        {provide: MatDialog, useValue: {}},
        {provide: SnackBarService, useValue: {}},
        {provide: ErrorHandler, useValue: {handleError: vi.fn()}},
      ],
    }).compileComponents();
  });

  function createComponent(): JobExecutionDetailComponent {
    const component = TestBed.createComponent(JobExecutionDetailComponent).componentInstance;
    component.ngOnInit();
    return component;
  }

  it('deduplicates active execution ids and makes one aggregate request', async () => {
    const component = createComponent();

    expect(await firstValueFrom(component.queueSize$)).toBe(23);
    expect(queueCountForCrawlExecutions).toHaveBeenCalledOnce();
    expect(queueCountForCrawlExecutions).toHaveBeenCalledWith(['crawl-1', 'crawl-2']);
    expect(queueCountForCrawlExecution).not.toHaveBeenCalled();
  });

  it('returns zero without a controller request when no active executions exist', async () => {
    listCrawlExecutionsUnchecked.mockReturnValue(of());
    const component = createComponent();

    expect(await firstValueFrom(component.queueSize$)).toBe(0);
    expect(queueCountForCrawlExecutions).not.toHaveBeenCalled();
  });

  it('maps a failed batch count to unavailable', async () => {
    queueCountForCrawlExecutions.mockReturnValue(EMPTY);
    const component = createComponent();

    expect(await firstValueFrom(component.queueSize$)).toBeNull();
  });
});
