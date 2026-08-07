import {ErrorHandler} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter, Router} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {MatDialog} from '@angular/material/dialog';
import {EMPTY, Observable, of, Subject} from 'rxjs';

import {ControllerApiService} from '../../../core';
import {provideMaterialAnimationsDisabled} from '../../../core/core.testing.module';
import {CrawlerStatus} from '../../../shared/models/controller';
import {ConfigObject, JobExecutionState, JobExecutionStatus, ListRange, Meta} from '../../../shared/models';
import {JobExecutionService, JobExecutionStatusQuery} from '../../report/services';
import {DashboardComponent} from './dashboard.component';

describe('DashboardComponent', () => {
  let canReadJobExecutions: boolean;
  let search: ReturnType<typeof vi.fn<
    (query: JobExecutionStatusQuery, range: ListRange) => Observable<JobExecutionStatus>
  >>;
  let getCrawlerStatus: ReturnType<typeof vi.fn<() => Observable<CrawlerStatus>>>;
  let pauseCrawler: ReturnType<typeof vi.fn<() => Observable<void>>>;
  let unpauseCrawler: ReturnType<typeof vi.fn<() => Observable<void>>>;
  let dialogResult: Subject<boolean>;

  beforeEach(async () => {
    vi.useFakeTimers();
    canReadJobExecutions = true;
    search = vi.fn(() => of());
    getCrawlerStatus = vi.fn(() => of(new CrawlerStatus()));
    pauseCrawler = vi.fn(() => of(undefined));
    unpauseCrawler = vi.fn(() => of(undefined));
    dialogResult = new Subject<boolean>();

    await TestBed.configureTestingModule({
      imports: [DashboardComponent],
      providers: [
        provideMaterialAnimationsDisabled(),
        provideRouter([]),
        {
          provide: AbilityServiceSignal,
          useValue: {
            can: (action: string, subject: string) =>
              action === 'read' && (subject !== 'jobexecution' || canReadJobExecutions),
          },
        },
        {
          provide: ControllerApiService,
          useValue: {getCrawlerStatus, pauseCrawler, unpauseCrawler},
        },
        {
          provide: MatDialog,
          useValue: {open: vi.fn(() => ({afterClosed: () => dialogResult.asObservable()}))},
        },
        {
          provide: ErrorHandler,
          useValue: {handleError: vi.fn()},
        },
        {
          provide: JobExecutionService,
          useValue: {
            getJob: (id: string) => of(new ConfigObject({id, meta: new Meta({name: id})})),
            search,
          },
        },
      ],
    }).compileComponents();
  });

  afterEach(() => vi.useRealTimers());

  function createDashboard(): ComponentFixture<DashboardComponent> {
    const fixture = TestBed.createComponent(DashboardComponent);
    fixture.detectChanges();
    return fixture;
  }

  it('renders the dashboard hero, embedded crawler status, and full-width latest crawl jobs', () => {
    const fixture = createDashboard();
    const grid = fixture.nativeElement.querySelector('.dashboard-grid') as HTMLElement;
    const introCard = grid.querySelector('.dashboard-intro-card') as HTMLElement;
    const artworkCard = grid.querySelector('.dashboard-artwork-card') as HTMLElement;
    const heading = introCard.querySelector('h1') as HTMLElement;
    const subtitle = introCard.querySelector('.dashboard-intro-copy p') as HTMLElement;
    const runningCrawls = grid.querySelector('.dashboard-running-crawls') as HTMLElement;

    expect(heading.textContent).toBe('Dashboard');
    expect(subtitle.textContent.trim()).toBe('Monitor crawler activity and follow running jobs in one place.');
    expect(grid.firstElementChild).toBe(introCard);
    expect(introCard.nextElementSibling).toBe(artworkCard);
    expect(introCard.querySelector('app-crawlerstatus')).not.toBeNull();
    expect(introCard.querySelector('mat-card')).toBeNull();
    expect(artworkCard.getAttribute('aria-hidden')).toBe('true');
    expect(artworkCard.querySelector('.dashboard-brand-logo')?.getAttribute('src'))
      .toBe('public/logo/veidemann_logo_inline_black.png');
    expect(artworkCard.querySelector('source')?.getAttribute('srcset'))
      .toBe('public/logo/veidemann_horizontal_white.png');
    expect(runningCrawls.tagName).toBe('APP-RUNNING-CRAWLS');
    expect(runningCrawls.parentElement).toBe(grid);
    expect(runningCrawls.textContent).toContain('Latest crawl jobs');
    expect(runningCrawls.querySelector('mat-card')).toBeNull();
  });

  it('loads crawler status once on initial render', () => {
    createDashboard();

    expect(getCrawlerStatus).toHaveBeenCalledTimes(1);
  });

  it.each([
    ['pause', true],
    ['resume', false],
  ])('refreshes crawler status once after a successful %s action', (_action, shouldPause) => {
    const actionResult = new Subject<void>();
    const action = shouldPause ? pauseCrawler : unpauseCrawler;
    action.mockReturnValue(actionResult.asObservable());
    const fixture = createDashboard();

    fixture.componentInstance.onChangeRunStatus(shouldPause);
    dialogResult.next(true);

    expect(action).toHaveBeenCalledTimes(1);
    expect(getCrawlerStatus).toHaveBeenCalledTimes(1);

    actionResult.next();
    actionResult.next();
    actionResult.complete();

    expect(getCrawlerStatus).toHaveBeenCalledTimes(2);
  });

  it('does not change or refresh crawler status when the dialog is cancelled', () => {
    const fixture = createDashboard();

    fixture.componentInstance.onChangeRunStatus(true);
    dialogResult.next(false);

    expect(pauseCrawler).not.toHaveBeenCalled();
    expect(unpauseCrawler).not.toHaveBeenCalled();
    expect(getCrawlerStatus).toHaveBeenCalledTimes(1);
  });

  it('does not refresh crawler status when the action fails', () => {
    pauseCrawler.mockReturnValue(EMPTY);
    const fixture = createDashboard();

    fixture.componentInstance.onChangeRunStatus(true);
    dialogResult.next(true);

    expect(pauseCrawler).toHaveBeenCalledTimes(1);
    expect(getCrawlerStatus).toHaveBeenCalledTimes(1);
  });

  it('queries running executions newest-first in 100-row pages and polls every 15 seconds', async () => {
    search.mockReturnValue(of(...Array.from({length: 12}, (_, index) => new JobExecutionStatus({
      id: `execution-${index}`,
      state: JobExecutionState.RUNNING,
    }))));
    const fixture = createDashboard();

    await fixture.whenStable();
    fixture.detectChanges();

    const [query, range] = search.mock.calls[0] as [JobExecutionStatusQuery, {offset: number; pageSize: number}];
    expect(query).toEqual(expect.objectContaining({
      active: 'startTime',
      direction: 'desc',
      stateList: [JobExecutionState.RUNNING],
      watch: false,
    }));
    expect(range).toEqual({offset: 0, pageSize: 100});
    expect(fixture.nativeElement.querySelectorAll('.item-row').length).toBe(12);

    await vi.advanceTimersByTimeAsync(15_000);
    expect(search).toHaveBeenCalledTimes(2);
  });

  it('does not render or subscribe to running crawls without permission', async () => {
    canReadJobExecutions = false;
    const fixture = createDashboard();

    await vi.advanceTimersByTimeAsync(30_000);

    expect(fixture.nativeElement.querySelector('app-running-crawls')).toBeNull();
    expect(search).not.toHaveBeenCalled();
  });

  it('updates the list query from state chips and links rows to their crawl executions', () => {
    const fixture = createDashboard();
    const router = TestBed.inject(Router);
    const navigate = vi.spyOn(router, 'navigate').mockResolvedValue(true);

    fixture.componentInstance.onSelectedStatesChange([
      JobExecutionState.RUNNING,
      JobExecutionState.ABORTED_MANUAL,
    ]);
    const [query] = search.mock.calls.at(-1) as [JobExecutionStatusQuery, ListRange];
    expect(query.stateList).toEqual([JobExecutionState.RUNNING, JobExecutionState.ABORTED_MANUAL]);

    fixture.componentInstance.onSelectedStatesChange([]);
    const [unfilteredQuery] = search.mock.calls.at(-1) as [JobExecutionStatusQuery, ListRange];
    expect(unfilteredQuery.stateList).toEqual([]);

    fixture.componentInstance.onJobExecutionClick(new JobExecutionStatus({id: 'execution-1'}));
    expect(navigate).toHaveBeenCalledWith(['/report', 'crawlexecution'], {
      queryParams: {job_execution_id: 'execution-1'},
    });
  });
});
