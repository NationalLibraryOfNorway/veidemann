import {ErrorHandler} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter, Router} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {MatDialog} from '@angular/material/dialog';
import {EMPTY, Observable, of, Subject} from 'rxjs';

import {AuthService, ControllerApiService} from '../../../core';
import {provideMaterialAnimationsDisabled} from '../../../core/core.testing.module';
import {CrawlerStatus} from '../../../shared/models/controller';
import {ConfigObject, JobExecutionState, JobExecutionStatus, ListRange, Meta} from '../../../shared/models';
import {JobExecutionService, JobExecutionStatusQuery} from '../../report/services';
import {CrawlerStatusDialogComponent} from '../crawlerstatus-dialog/crawlerstatus-dialog.component';
import {DashboardComponent} from './dashboard.component';

describe('DashboardComponent', () => {
  let isLoggedIn: boolean;
  let canReadJobExecutions: boolean;
  let login: ReturnType<typeof vi.fn>;
  let search: ReturnType<typeof vi.fn<
    (query: JobExecutionStatusQuery, range: ListRange) => Observable<JobExecutionStatus>
  >>;
  let getCrawlerStatus: ReturnType<typeof vi.fn<() => Observable<CrawlerStatus>>>;
  let pauseCrawler: ReturnType<typeof vi.fn<() => Observable<void>>>;
  let unpauseCrawler: ReturnType<typeof vi.fn<() => Observable<void>>>;
  let dialogResult: Subject<boolean>;

  beforeEach(async () => {
    vi.useFakeTimers();
    isLoggedIn = true;
    canReadJobExecutions = true;
    login = vi.fn();
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
              isLoggedIn && action === 'read' && (subject !== 'jobexecution' || canReadJobExecutions),
          },
        },
        {
          provide: AuthService,
          useValue: {
            get isLoggedIn() { return isLoggedIn; },
            login,
          },
        },
        {
          provide: ControllerApiService,
          useValue: {
            getCrawlerStatus,
            pauseCrawler,
            unpauseCrawler,
          },
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

  it('renders the hero, centered status above its metrics, and full-width crawl jobs', () => {
    const fixture = createDashboard();
    const page = fixture.nativeElement.querySelector('.dashboard-page') as HTMLElement;
    const heroHeader = page.querySelector('.destination-hero') as HTMLElement;
    const introCard = heroHeader.querySelector('.destination-intro-card') as HTMLElement;
    const artworkCard = heroHeader.querySelector('.destination-artwork-card') as HTMLElement;
    const heading = introCard.querySelector('h1') as HTMLElement;
    const subtitle = introCard.querySelector('.destination-intro-copy p') as HTMLElement;
    const crawlerStatus = page.querySelector('.dashboard-crawler-status') as HTMLElement;
    const runningCrawls = page.querySelector('.dashboard-running-crawls') as HTMLElement;

    expect(heading.textContent).toBe('Dashboard');
    expect(subtitle.textContent.trim()).toBe('Monitor crawler activity and follow running jobs in one place.');
    expect(heroHeader.firstElementChild).toBe(introCard);
    expect(introCard.nextElementSibling).toBe(artworkCard);
    expect(introCard.querySelector('app-crawlerstatus')).toBeNull();
    expect(heroHeader.nextElementSibling).toBe(crawlerStatus);
    expect(crawlerStatus.nextElementSibling).toBe(runningCrawls);
    const statusControl = crawlerStatus.querySelector('.status-control') as HTMLElement;
    const metricList = crawlerStatus.querySelector('.status-metrics') as HTMLElement;
    expect(statusControl.compareDocumentPosition(metricList) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(getComputedStyle(statusControl).justifyContent).toBe('center');
    const metrics = [...crawlerStatus.querySelectorAll('.metric')] as HTMLElement[];
    expect(metrics.map(metric => metric.querySelector('dt').textContent.trim()))
      .toEqual(['Total queue size', 'Busy crawl host groups']);
    expect(getComputedStyle(metricList).display).toBe('flex');
    expect(getComputedStyle(metricList).justifyContent).toBe('center');
    expect(crawlerStatus.querySelector('.metric-divider.mat-divider-vertical')).not.toBeNull();
    expect(introCard.querySelector('mat-card')).toBeNull();
    expect(artworkCard.getAttribute('aria-hidden')).toBe('true');
    expect(artworkCard.classList).toContain('mat-mdc-card-filled');
    expect(artworkCard.classList).not.toContain('mat-mdc-card-outlined');
    for (const heroCard of [introCard, artworkCard]) {
      const style = getComputedStyle(heroCard);
      expect(style.backgroundColor).toBe('rgba(0, 0, 0, 0)');
      expect(style.borderTopWidth).toBe('0px');
      expect(style.boxShadow).toBe('none');
    }
    expect(artworkCard.querySelector('.destination-brand-logo')?.getAttribute('src'))
      .toBe('public/logo/veidemann_logo_inline_black.png');
    expect(artworkCard.querySelector('source')?.getAttribute('srcset'))
      .toBe('public/logo/veidemann_horizontal_white.png');
    expect(runningCrawls.tagName).toBe('APP-RUNNING-CRAWLS');
    expect(runningCrawls.parentElement).toBe(page);
    expect(runningCrawls.querySelector('h2')).toBeNull();
    expect(runningCrawls.querySelector('.latest-crawl-jobs')?.getAttribute('aria-label')).toBe('Jobs');
    expect(runningCrawls.querySelector('mat-card')).toBeNull();
  });

  it('uses the shared full-width hero container without a bottom margin', () => {
    const fixture = createDashboard();
    const page = fixture.nativeElement.querySelector('.dashboard-page') as HTMLElement;
    const heroHeader = page.querySelector('.destination-hero') as HTMLElement;

    expect(getComputedStyle(page).width).toBe('100%');
    expect(getComputedStyle(page).marginTop).toBe('8px');
    expect(getComputedStyle(page).marginRight).toBe('0px');
    expect(getComputedStyle(page).padding).toBe('0px');
    expect(getComputedStyle(heroHeader).paddingInline).toBe('8px');
    expect(getComputedStyle(heroHeader).gap).toBe('8px');
    expect(getComputedStyle(heroHeader).marginBottom).toBe('0');
    expect(getComputedStyle(heroHeader.querySelector('.destination-hero-card') as HTMLElement).minHeight)
      .toBe('320px');
  });

  it('renders the logo and a welcome login card when logged out', () => {
    isLoggedIn = false;
    const fixture = createDashboard();
    const page = fixture.nativeElement.querySelector('.dashboard-page') as HTMLElement;
    const heroHeader = page.querySelector('.destination-hero') as HTMLElement;
    const welcomeCard = heroHeader.querySelector('.dashboard-welcome-card') as HTMLElement;
    const artworkCard = heroHeader.querySelector('.destination-artwork-card') as HTMLElement;
    const loginButton = welcomeCard.querySelector('.dashboard-login-button') as HTMLButtonElement;

    expect(welcomeCard.querySelector('h1').textContent).toBe('Signed out');
    expect(welcomeCard.querySelector('p').textContent).toBe('Sign in to access dashboard.');
    expect(welcomeCard.classList).toContain('destination-intro-card');
    expect(heroHeader.firstElementChild).toBe(welcomeCard);
    expect(welcomeCard.nextElementSibling).toBe(artworkCard);
    expect(artworkCard.classList).toContain('mat-mdc-card-filled');
    expect(artworkCard.querySelector('.destination-brand-logo')?.getAttribute('src'))
      .toBe('public/logo/veidemann_logo_inline_black.png');
    expect(loginButton.textContent).toContain('LOGIN');
    expect(getComputedStyle(loginButton).minHeight).toBe('40px');
    expect(page.querySelector('app-crawlerstatus')).toBeNull();
    expect(page.querySelector('app-running-crawls')).toBeNull();

    loginButton.click();

    expect(login).toHaveBeenCalledWith('/');
    expect(getCrawlerStatus).not.toHaveBeenCalled();
    expect(search).not.toHaveBeenCalled();
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
    expect(TestBed.inject(MatDialog).open).toHaveBeenLastCalledWith(
      CrawlerStatusDialogComponent,
      expect.objectContaining({disableClose: false}),
    );
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

  it('updates the list query from state chips and links rows to job execution details', () => {
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
    expect(navigate).toHaveBeenCalledWith(['/report', 'jobexecution', 'execution-1']);
  });
});
