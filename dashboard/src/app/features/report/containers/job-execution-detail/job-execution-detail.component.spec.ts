import {ErrorHandler} from '@angular/core';
import {TestBed} from '@angular/core/testing';
import {MatDialog} from '@angular/material/dialog';
import {ActivatedRoute, convertToParamMap, provideRouter, Router} from '@angular/router';
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
  let queueCountsForJobExecutions: ReturnType<typeof vi.fn>;
  let get: ReturnType<typeof vi.fn>;
  let queryParamMap: BehaviorSubject<ReturnType<typeof convertToParamMap>>;

  beforeEach(async () => {
    queueCountsForJobExecutions = vi.fn(() => of(new Map([[jobStatus.id, 23]])));
    get = vi.fn(() => of(jobStatus));
    queryParamMap = new BehaviorSubject(convertToParamMap({watch: 'false'}));

    await TestBed.configureTestingModule({
      imports: [JobExecutionDetailComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {
          provide: ActivatedRoute,
          useValue: {
            paramMap: new BehaviorSubject(convertToParamMap({id: jobStatus.id})),
            queryParamMap,
          },
        },
        {
          provide: JobExecutionService,
          useValue: {get, getJob: vi.fn(() => of(null))},
        },
        {
          provide: ControllerApiService,
          useValue: {queueCountsForJobExecutions},
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
    expect(queueCountsForJobExecutions).toHaveBeenCalledWith([jobStatus.id]);
  });

  it('maps a failed job count to unavailable', async () => {
    queueCountsForJobExecutions.mockReturnValue(EMPTY);
    const component = createComponent();

    expect(await firstValueFrom(component.queueSize$)).toBeNull();
  });

  it('uses a snapshot unless watch is explicitly enabled', async () => {
    const component = createComponent();
    await firstValueFrom(component.item$);
    expect(get).toHaveBeenCalledWith({id: jobStatus.id, watch: false});
    expect(get).not.toHaveBeenCalledWith({id: jobStatus.id, watch: true});

    queryParamMap.next(convertToParamMap({watch: 'true'}));
    await firstValueFrom(component.item$);
    expect(get).toHaveBeenCalledWith({id: jobStatus.id, watch: true});
  });

  it('places the stateful watch button immediately before the overflow menu', async () => {
    const fixture = TestBed.createComponent(JobExecutionDetailComponent);
    fixture.detectChanges();
    await new Promise(resolve => setTimeout(resolve));
    fixture.detectChanges();
    const actions = [...fixture.nativeElement.querySelector('.detail-header-actions').children] as HTMLElement[];

    expect(actions[0].classList).toContain('watch-toggle');
    expect(actions[0].getAttribute('aria-pressed')).toBe('false');
    expect(actions[1].tagName).toBe('APP-DETAIL-OVERFLOW');

    const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    actions[0].click();
    expect(navigate).toHaveBeenCalledWith([], expect.objectContaining({queryParams: {watch: true}}));
  });
});
