import {ErrorHandler} from '@angular/core';
import {TestBed} from '@angular/core/testing';
import {MatDialog} from '@angular/material/dialog';
import {ActivatedRoute, convertToParamMap} from '@angular/router';
import {BehaviorSubject, EMPTY, firstValueFrom, of} from 'rxjs';

import {ControllerApiService, SnackBarService} from '../../../../core';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {JobExecutionState, JobExecutionStatus} from '../../../../shared/models';
import {JobExecutionService} from '../../services';
import {JobExecutionDetailComponent} from './job-execution-detail.component';

describe('JobExecutionDetailComponent', () => {
  const jobStatus = new JobExecutionStatus({
    id: 'job-execution-1',
    jobId: 'job-1',
    state: JobExecutionState.RUNNING,
  });
  let queueCountForJobExecution: ReturnType<typeof vi.fn>;

  beforeEach(async () => {
    queueCountForJobExecution = vi.fn(() => of({count: 23}));

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
          provide: ControllerApiService,
          useValue: {queueCountForJobExecution},
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

  it('gets the aggregate count by job execution id', async () => {
    const component = createComponent();

    expect(await firstValueFrom(component.queueSize$)).toBe(23);
    expect(queueCountForJobExecution).toHaveBeenCalledOnce();
    expect(queueCountForJobExecution.mock.calls[0][0].id).toBe(jobStatus.id);
  });

  it('maps a failed job count to unavailable', async () => {
    queueCountForJobExecution.mockReturnValue(EMPTY);
    const component = createComponent();

    expect(await firstValueFrom(component.queueSize$)).toBeNull();
  });
});
