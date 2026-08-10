import {TestBed} from '@angular/core/testing';
import {MatDialog} from '@angular/material/dialog';
import {ActivatedRoute, convertToParamMap} from '@angular/router';
import {BehaviorSubject, EMPTY, firstValueFrom, of} from 'rxjs';

import {ControllerApiService, SnackBarService} from '../../../../core';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {CrawlExecutionState, CrawlExecutionStatus} from '../../../../shared/models';
import {CrawlExecutionService} from '../../services';
import {CrawlExecutionDetailComponent} from './crawl-execution-detail.component';

describe('CrawlExecutionDetailComponent', () => {
  const status = new CrawlExecutionStatus({id: 'crawl-1', state: CrawlExecutionState.FETCHING});
  let queueCountForCrawlExecution: ReturnType<typeof vi.fn>;

  beforeEach(async () => {
    queueCountForCrawlExecution = vi.fn(() => of({count: 7}));
    await TestBed.configureTestingModule({
      imports: [CrawlExecutionDetailComponent],
      providers: [
        ...provideCoreTesting,
        {
          provide: ActivatedRoute,
          useValue: {
            paramMap: new BehaviorSubject(convertToParamMap({id: status.id})),
            queryParamMap: new BehaviorSubject(convertToParamMap({watch: 'false'})),
          },
        },
        {
          provide: CrawlExecutionService,
          useValue: {get: vi.fn(() => of(status))},
        },
        {
          provide: ControllerApiService,
          useValue: {queueCountForCrawlExecution},
        },
        {provide: MatDialog, useValue: {}},
        {provide: SnackBarService, useValue: {}},
      ],
    }).compileComponents();
  });

  it('uses one single-execution queue-count request', async () => {
    const component = TestBed.createComponent(CrawlExecutionDetailComponent).componentInstance;
    component.ngOnInit();

    expect(await firstValueFrom(component.queueSize$)).toBe(7);
    expect(queueCountForCrawlExecution).toHaveBeenCalledOnce();
    expect(queueCountForCrawlExecution.mock.calls[0][0].id).toBe(status.id);
  });

  it('maps a failed queue count to unavailable', async () => {
    queueCountForCrawlExecution.mockReturnValue(EMPTY);
    const component = TestBed.createComponent(CrawlExecutionDetailComponent).componentInstance;
    component.ngOnInit();

    expect(await firstValueFrom(component.queueSize$)).toBeNull();
  });
});
