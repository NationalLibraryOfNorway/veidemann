import {TestBed} from '@angular/core/testing';
import {MatDialog} from '@angular/material/dialog';
import {ActivatedRoute, convertToParamMap, provideRouter, Router} from '@angular/router';
import {BehaviorSubject, EMPTY, firstValueFrom, of} from 'rxjs';

import {ControllerApiService, SnackBarService} from '../../../../core';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {CrawlExecutionState, CrawlExecutionStatus} from '../../../../shared/models';
import {CrawlExecutionService, JobExecutionService} from '../../services';
import {CrawlExecutionDetailComponent} from './crawl-execution-detail.component';

describe('CrawlExecutionDetailComponent', () => {
  const status = new CrawlExecutionStatus({id: 'crawl-1', state: CrawlExecutionState.FETCHING});
  let queueCountsForCrawlExecutions: ReturnType<typeof vi.fn>;
  let get: ReturnType<typeof vi.fn>;
  let queryParamMap: BehaviorSubject<ReturnType<typeof convertToParamMap>>;

  beforeEach(async () => {
    queueCountsForCrawlExecutions = vi.fn(() => of(new Map([[status.id, 7]])));
    get = vi.fn(() => of(status));
    queryParamMap = new BehaviorSubject(convertToParamMap({watch: 'false'}));
    await TestBed.configureTestingModule({
      imports: [CrawlExecutionDetailComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {
          provide: ActivatedRoute,
          useValue: {
            paramMap: new BehaviorSubject(convertToParamMap({id: status.id})),
            queryParamMap,
          },
        },
        {
          provide: CrawlExecutionService,
          useValue: {get, getSeed: vi.fn(() => of(null))},
        },
        {provide: JobExecutionService, useValue: {getJob: vi.fn(() => of(null))}},
        {
          provide: ControllerApiService,
          useValue: {queueCountsForCrawlExecutions},
        },
        {provide: MatDialog, useValue: {}},
        {provide: SnackBarService, useValue: {}},
      ],
    }).compileComponents();
  });

  it('uses one bounded queue-count request', async () => {
    const component = TestBed.createComponent(CrawlExecutionDetailComponent).componentInstance;
    component.ngOnInit();

    expect(await firstValueFrom(component.queueSize$)).toBe(7);
    expect(queueCountsForCrawlExecutions).toHaveBeenCalledWith([status.id]);
  });

  it('maps a failed queue count to unavailable', async () => {
    queueCountsForCrawlExecutions.mockReturnValue(EMPTY);
    const component = TestBed.createComponent(CrawlExecutionDetailComponent).componentInstance;
    component.ngOnInit();

    expect(await firstValueFrom(component.queueSize$)).toBeNull();
  });

  it('does not open a crawl changefeed until watch is enabled', async () => {
    const component = TestBed.createComponent(CrawlExecutionDetailComponent).componentInstance;
    component.ngOnInit();
    await firstValueFrom(component.item$);
    expect(get).toHaveBeenCalledWith({id: status.id, watch: false});
    expect(get).not.toHaveBeenCalledWith({id: status.id, watch: true});

    queryParamMap.next(convertToParamMap({watch: 'true'}));
    await firstValueFrom(component.item$);
    expect(get).toHaveBeenCalledWith({id: status.id, watch: true});
  });

  it('places the stateful watch button immediately before the overflow menu', async () => {
    const fixture = TestBed.createComponent(CrawlExecutionDetailComponent);
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

  it('hides the watch button when the execution is already finished', async () => {
    get.mockReturnValue(of(new CrawlExecutionStatus({
      id: status.id,
      state: CrawlExecutionState.FINISHED,
    })));
    const fixture = TestBed.createComponent(CrawlExecutionDetailComponent);
    fixture.detectChanges();
    await new Promise(resolve => setTimeout(resolve));
    fixture.detectChanges();

    const actions = [...fixture.nativeElement.querySelector('.detail-header-actions').children] as HTMLElement[];
    expect(fixture.nativeElement.querySelector('.watch-toggle')).toBeNull();
    expect(actions[0].tagName).toBe('APP-DETAIL-OVERFLOW');
  });
});
