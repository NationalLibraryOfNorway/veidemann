import {Component, ErrorHandler} from '@angular/core';
import {BreakpointObserver, BreakpointState} from '@angular/cdk/layout';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {NavigationEnd, provideRouter, ResolveFn, Router} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {MatDialog} from '@angular/material/dialog';
import {BehaviorSubject, filter, firstValueFrom, ReplaySubject, take} from 'rxjs';

import {AuthService, SnackBarService} from '../../core';
import {configureMaterialSymbols} from '../../app.config';
import {provideMaterialAnimationsDisabled} from '../../core/core.testing.module';
import {AppComponent} from './app.component';

@Component({template: '<p data-testid="route-content">Route content</p>', standalone: true})
class EmptyRouteComponent {}

const RAIL_BREAKPOINT = '(min-width: 840px)';
const PERSISTENT_DRAWER_BREAKPOINT = '(min-width: 1921px)';

describe('AppComponent navigation', () => {
  let fixture: ComponentFixture<AppComponent>;
  let router: Router;
  let breakpointState: BehaviorSubject<BreakpointState>;
  let seedResolution: ReplaySubject<unknown>;
  let dialog: {open: ReturnType<typeof vi.fn>};
  const can = vi.fn((_action: string, subject: string) =>
    ['configs', 'CRAWLENTITY', 'SEED'].includes(subject));

  beforeEach(async () => {
    can.mockImplementation((_action: string, subject: string) =>
      ['configs', 'CRAWLENTITY', 'SEED'].includes(subject));
    breakpointState = new BehaviorSubject<BreakpointState>({
      matches: false,
      breakpoints: {
        [RAIL_BREAKPOINT]: false,
        [PERSISTENT_DRAWER_BREAKPOINT]: false,
      },
    });
    seedResolution = new ReplaySubject<unknown>(1);
    dialog = {
      open: vi.fn(() => ({})),
    };
    const seedResolver: ResolveFn<unknown> = () => seedResolution;

    await TestBed.configureTestingModule({
      imports: [AppComponent],
      providers: [
        provideMaterialAnimationsDisabled(),
        {provide: BreakpointObserver, useValue: {observe: () => breakpointState.asObservable()}},
        provideRouter([
          {
            path: 'config',
            children: [
              {path: '', component: EmptyRouteComponent},
              {path: 'entity', component: EmptyRouteComponent},
              {path: 'seed', component: EmptyRouteComponent, resolve: {options: seedResolver}},
              {path: 'crawljobs', component: EmptyRouteComponent},
              {path: 'browserscript', component: EmptyRouteComponent},
            ],
          },
          {
            path: 'report',
            children: [
              {path: '', component: EmptyRouteComponent},
              {path: 'jobexecution', component: EmptyRouteComponent},
              {path: 'jobexecution/:id', component: EmptyRouteComponent},
              {path: 'crawlexecution', component: EmptyRouteComponent},
              {path: 'crawlexecution/:id', component: EmptyRouteComponent},
            ],
          },
        ]),
        {provide: AbilityServiceSignal, useValue: {can}},
        {
          provide: AuthService,
          useValue: {
            isLoggedIn: false,
            requestedUri: '',
            name: '',
            login: vi.fn(),
            logout: vi.fn(),
          },
        },
        {provide: ErrorHandler, useValue: {handleError: vi.fn()}},
        {provide: SnackBarService, useValue: {openSnackBar: vi.fn()}},
        {provide: MatDialog, useValue: dialog},
      ],
    }).compileComponents();

    TestBed.runInInjectionContext(configureMaterialSymbols);
    router = TestBed.inject(Router);
    fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    await fixture.whenStable();
  });

  function setNavigationWidth(rail: boolean, persistentDrawer: boolean): void {
    breakpointState.next({
      matches: rail || persistentDrawer,
      breakpoints: {
        [RAIL_BREAKPOINT]: rail,
        [PERSISTENT_DRAWER_BREAKPOINT]: persistentDrawer,
      },
    });
    fixture.detectChanges();
  }

  it('renders the compact toolbar and a clean permission-filtered main drawer', () => {
    expect(fixture.nativeElement.querySelector('.main-toolbar')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.navigation-rail')).toBeNull();
    expect(fixture.nativeElement.querySelector('app-dialog')).toBeNull();
    expect(fixture.nativeElement.textContent).not.toContain('Main menu');

    const homeLink = fixture.nativeElement.querySelector('.main-toolbar a[href="/"]') as HTMLAnchorElement;
    expect(homeLink.classList).not.toContain('mat-mdc-icon-button');
    expect(getComputedStyle(homeLink).backgroundColor).toBe('rgba(0, 0, 0, 0)');
    const brandLogo = homeLink.querySelector('img.toolbar-brand-logo') as HTMLImageElement;
    expect(brandLogo.getAttribute('src')).toContain('veidemann_logo_inline_black.png');
    expect(homeLink.querySelector('source')?.getAttribute('srcset')).toContain('veidemann_horizontal_white.png');
    expect(fixture.nativeElement.querySelector('.main-toolbar .toolbar-title')).toBeNull();

    const sectionTriggers = Array.from(
      fixture.nativeElement.querySelectorAll('.drawer-section-trigger')
    ) as HTMLButtonElement[];
    expect(sectionTriggers).toHaveLength(1);
    expect(sectionTriggers[0].textContent.trim()).toContain('Config');
    expect(sectionTriggers[0].getAttribute('type')).toBe('button');
  });

  it('renders section chevrons as icons inside trailing metadata', () => {
    can.mockImplementation((_action: string, subject: string) =>
      ['configs', 'report'].includes(subject));
    fixture.destroy();
    fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();

    const sectionTriggers = Array.from(
      fixture.nativeElement.querySelectorAll('.drawer-section-trigger')
    ) as HTMLButtonElement[];
    expect(sectionTriggers).toHaveLength(2);

    for (const trigger of sectionTriggers) {
      const meta = trigger.querySelector('.drawer-section-chevron') as HTMLSpanElement;
      const chevron = meta.querySelector('mat-icon') as HTMLElement;
      expect(meta.classList).toContain('mat-mdc-list-item-meta');
      expect(chevron.textContent).toContain('chevron_right');
      expect(chevron.classList).toContain('material-symbols-outlined');
      expect(chevron.classList).toContain('mat-ligature-font');
      expect(chevron.classList).not.toContain('mat-mdc-list-item-meta');
    }
  });

  it('shows the rail and keeps section navigation collapsed at middle widths', async () => {
    setNavigationWidth(true, false);
    await router.navigateByUrl('/config/entity');
    fixture.detectChanges();
    await fixture.whenStable();

    expect(fixture.nativeElement.querySelector('.main-toolbar')).toBeNull();
    expect(fixture.nativeElement.querySelector('.navigation-rail')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('mat-sidenav.mat-drawer-opened')).toBeNull();
    const menuButton = fixture.nativeElement.querySelector('.rail-menu') as HTMLButtonElement;
    expect(menuButton).not.toBeNull();
    expect(menuButton.getAttribute('aria-label')).toBe('Open navigation');
    expect(menuButton.getAttribute('aria-expanded')).toBe('false');
    expect(menuButton.getAttribute('aria-controls')).toBe('primary-navigation-drawer');
    expect(fixture.nativeElement.querySelector('.navigation-rail a[href="/config"]')).toBeNull();
    expect(fixture.nativeElement.querySelector('.navigation-rail a[href="/report"]')).toBeNull();
    const activeLinks = fixture.nativeElement.querySelectorAll(
      '.navigation-rail a[href="/config/entity"][aria-current="page"]'
    );
    expect(activeLinks.length).toBe(1);
    expect(activeLinks[0].classList).toContain('active-rail-destination');
    expect(activeLinks[0].querySelector('.rail-label').textContent.trim()).toBe('Entities');
    const homeLinks = fixture.nativeElement.querySelectorAll('.navigation-rail a[href="/"]');
    expect(homeLinks.length).toBe(1);
    expect(homeLinks[0].querySelector('.grouse-icon')).not.toBeNull();
  });

  it('shows direct rail destinations in order with descending execution links', () => {
    can.mockImplementation((_action: string, subject: string) =>
      ['configs', 'CRAWLENTITY', 'SEED', 'CRAWLJOB', 'report', 'jobexecution', 'crawlexecution']
        .includes(subject));
    fixture.destroy();
    fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    setNavigationWidth(true, false);

    const links = Array.from(
      fixture.nativeElement.querySelectorAll('.rail-destinations .rail-destination')
    ) as HTMLAnchorElement[];
    expect(links.map(link => link.querySelector('.rail-label')?.textContent.trim())).toEqual([
      'Home',
      'Entities',
      'Seeds',
      'Crawl jobs',
      'Jobs',
      'Crawls',
    ]);
    expect(links.map(link => link.querySelector('mat-icon')?.textContent.trim() ?? 'logo')).toEqual([
      'logo',
      'business',
      'link',
      'work',
      'hdr_strong',
      'hdr_weak',
    ]);
    expect(new URL(links[1].href).pathname).toBe('/config/entity');
    expect(new URL(links[2].href).pathname).toBe('/config/seed');
    expect(new URL(links[3].href).pathname).toBe('/config/crawljobs');
    expect(new URL(links[4].href).pathname).toBe('/report/jobexecution');
    expect(new URL(links[4].href).searchParams.get('sort')).toBe('startTime:desc');
    expect(new URL(links[5].href).pathname).toBe('/report/crawlexecution');
    expect(new URL(links[5].href).searchParams.get('sort')).toBe('startTime:desc');
    expect(links.some(link => ['/config', '/report'].includes(new URL(link.href).pathname))).toBe(false);
  });

  it('requires section and destination permissions for direct rail links', () => {
    can.mockImplementation((_action: string, subject: string) =>
      ['configs', 'CRAWLENTITY', 'jobexecution'].includes(subject));
    fixture.destroy();
    fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    setNavigationWidth(true, false);

    const labels = (Array.from(
      fixture.nativeElement.querySelectorAll('.rail-destinations .rail-label')
    ) as HTMLElement[]).map(label => label.textContent.trim());
    expect(labels).toEqual(['Home', 'Entities']);
  });

  it('keeps execution rail links active across filters and detail routes', async () => {
    can.mockImplementation((_action: string, subject: string) =>
      ['report', 'jobexecution', 'crawlexecution'].includes(subject));
    fixture.destroy();
    fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    setNavigationWidth(true, false);

    await router.navigateByUrl('/report/jobexecution/execution-1?sort=state:asc&job_id=job-1');
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector(
      '.navigation-rail a[href^="/report/jobexecution"].active-rail-destination[aria-current="page"]'
    )).not.toBeNull();

    await router.navigateByUrl('/report/crawlexecution/execution-2?sort=endTime:asc&seed_id=seed-1');
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector(
      '.navigation-rail a[href^="/report/crawlexecution"].active-rail-destination[aria-current="page"]'
    )).not.toBeNull();
  });

  it('closes the overlay and resets execution filters when a rail destination is selected', async () => {
    can.mockImplementation((_action: string, subject: string) =>
      ['report', 'jobexecution'].includes(subject));
    fixture.destroy();
    fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    setNavigationWidth(true, false);

    (fixture.nativeElement.querySelector('.rail-menu') as HTMLButtonElement).click();
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('mat-sidenav.mat-drawer-opened')).not.toBeNull();

    (fixture.nativeElement.querySelector(
      '.navigation-rail a[href^="/report/jobexecution"]'
    ) as HTMLAnchorElement).click();
    fixture.detectChanges();
    await fixture.whenStable();

    const url = new URL(router.url, 'http://localhost');
    expect(url.pathname).toBe('/report/jobexecution');
    expect(url.searchParams.get('sort')).toBe('startTime:desc');
    expect([...url.searchParams.keys()]).toEqual(['sort']);
    expect(fixture.nativeElement.querySelector('mat-sidenav.mat-drawer-opened')).toBeNull();
  });

  it('opens contextual configuration navigation from the rail and closes after selection', async () => {
    can.mockImplementation((_action: string, subject: string) =>
      ['configs', 'SEED', 'BROWSERSCRIPT'].includes(subject));
    fixture.destroy();
    fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    setNavigationWidth(true, false);
    seedResolution.next({});
    seedResolution.complete();
    await router.navigateByUrl('/config/seed');
    fixture.detectChanges();
    await fixture.whenStable();

    const menuButton = fixture.nativeElement.querySelector('.rail-menu') as HTMLButtonElement;
    menuButton.click();
    fixture.detectChanges();
    await fixture.whenStable();

    expect(fixture.nativeElement.querySelector('mat-sidenav.mat-drawer-opened')).not.toBeNull();
    expect(menuButton.getAttribute('aria-label')).toBe('Close navigation');
    expect(menuButton.getAttribute('aria-expanded')).toBe('true');
    expect(menuButton.querySelector('mat-icon')?.textContent.trim()).toBe('menu_open');
    const browserScriptLink = fixture.nativeElement.querySelector(
      '.primary-navigation a[href="/config/browserscript"]'
    ) as HTMLAnchorElement;
    expect(browserScriptLink).not.toBeNull();

    menuButton.click();
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('mat-sidenav.mat-drawer-opened')).toBeNull();
    expect(menuButton.getAttribute('aria-label')).toBe('Open navigation');
    expect(menuButton.getAttribute('aria-expanded')).toBe('false');

    menuButton.click();
    fixture.detectChanges();
    await fixture.whenStable();
    (fixture.nativeElement.querySelector(
      '.primary-navigation a[href="/config/browserscript"]'
    ) as HTMLAnchorElement).click();
    fixture.detectChanges();
    await fixture.whenStable();

    expect(router.url).toBe('/config/browserscript');
    expect(fixture.nativeElement.querySelector('mat-sidenav.mat-drawer-opened')).toBeNull();
  });

  it('opens the main menu from home and contextual report navigation from report routes', async () => {
    can.mockImplementation((_action: string, subject: string) =>
      ['configs', 'report', 'jobexecution'].includes(subject));
    fixture.destroy();
    fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    setNavigationWidth(true, false);

    let menuButton = fixture.nativeElement.querySelector('.rail-menu') as HTMLButtonElement;
    menuButton.click();
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('.drawer-section-trigger')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.primary-navigation a[href="/report/jobexecution"]')).toBeNull();

    menuButton.click();
    fixture.detectChanges();
    await fixture.whenStable();
    await router.navigateByUrl('/report/jobexecution');
    fixture.detectChanges();
    await fixture.whenStable();

    menuButton = fixture.nativeElement.querySelector('.rail-menu') as HTMLButtonElement;
    menuButton.click();
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector(
      '.primary-navigation a[href="/report/jobexecution"]'
    )).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.primary-navigation a[href="/config/seed"]')).toBeNull();
  });

  it('drills into section navigation and returns to the main drawer', () => {
    const drawerTriggers = Array.from(
      fixture.nativeElement.querySelectorAll('.drawer-section-trigger')
    ) as HTMLButtonElement[];
    const configurationButton = drawerTriggers.find(
      element => element.textContent?.includes('Config')
    ) as HTMLButtonElement;
    configurationButton.click();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.primary-navigation a[href="/config"]')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.primary-navigation a[href="/config/seed"]')).not.toBeNull();

    const backButton = fixture.nativeElement.querySelector('.drawer-back') as HTMLButtonElement;
    expect(backButton.getAttribute('aria-label')).toBe('Back to main navigation');
    expect(backButton.textContent).not.toContain('Main menu');
    backButton.click();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.drawer-section-trigger')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.drawer-back')).toBeNull();
  });

  it('renders configuration and report child destinations without menu icons', () => {
    can.mockImplementation((_action: string, subject: string) =>
      ['configs', 'SEED', 'report', 'jobexecution'].includes(subject));
    fixture.destroy();
    fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();

    const sectionTriggers = Array.from(
      fixture.nativeElement.querySelectorAll('.drawer-section-trigger')
    ) as HTMLButtonElement[];
    expect(sectionTriggers).toHaveLength(2);
    for (const trigger of sectionTriggers) {
      expect(trigger.querySelector('mat-icon.mat-mdc-list-item-icon')).not.toBeNull();
    }

    sectionTriggers.find(trigger => trigger.textContent?.includes('Config'))?.click();
    fixture.detectChanges();
    const seedLink = fixture.nativeElement.querySelector(
      '.primary-navigation a[href="/config/seed"]'
    ) as HTMLAnchorElement;
    expect(seedLink.textContent).toContain('Seed');
    expect(seedLink.querySelector('mat-icon')).toBeNull();

    (fixture.nativeElement.querySelector('.drawer-back') as HTMLButtonElement).click();
    fixture.detectChanges();
    const reportTriggers = Array.from(
      fixture.nativeElement.querySelectorAll('.drawer-section-trigger')
    ) as HTMLButtonElement[];
    const reportTrigger = reportTriggers.find(
      trigger => trigger.textContent?.includes('Reports')
    ) as HTMLButtonElement;
    reportTrigger.click();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.primary-navigation a[href="/report"]')).not.toBeNull();
    const jobExecutionLink = fixture.nativeElement.querySelector(
      '.primary-navigation a[href="/report/jobexecution"]'
    ) as HTMLAnchorElement;
    expect(jobExecutionLink.textContent).toContain('Job execution');
    expect(jobExecutionLink.querySelector('mat-icon')).toBeNull();
  });

  it('keeps the main compact toolbar on section child routes', async () => {
    can.mockImplementation((_action: string, subject: string) =>
      ['configs', 'SEED', 'report', 'jobexecution'].includes(subject));
    await router.navigateByUrl('/config/entity');
    fixture.detectChanges();
    await fixture.whenStable();

    let menuButton = fixture.nativeElement.querySelector(
      '.main-toolbar button.toolbar-leading'
    ) as HTMLButtonElement;
    expect(menuButton.getAttribute('aria-label')).toBe('Open navigation');
    expect(menuButton.getAttribute('aria-expanded')).toBe('false');
    expect(menuButton.getAttribute('aria-controls')).toBe('primary-navigation-drawer');
    expect(menuButton.querySelector('mat-icon')?.textContent).toContain('menu');
    expect(fixture.nativeElement.querySelector('.main-toolbar a.brand-link[href="/"]')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.main-toolbar .toolbar-brand-logo')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.main-toolbar a.toolbar-leading')).toBeNull();
    expect(fixture.nativeElement.querySelector('.main-toolbar mat-icon')?.textContent).not.toContain('arrow_back');

    menuButton.click();
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('mat-sidenav.mat-drawer-opened')).not.toBeNull();
    expect(menuButton.getAttribute('aria-label')).toBe('Close navigation');
    expect(menuButton.getAttribute('aria-expanded')).toBe('true');
    expect(fixture.nativeElement.querySelector('.primary-navigation a[href="/config/seed"]')).not.toBeNull();

    menuButton.click();
    fixture.detectChanges();
    await fixture.whenStable();

    await router.navigateByUrl('/report/jobexecution');
    fixture.detectChanges();
    await fixture.whenStable();

    menuButton = fixture.nativeElement.querySelector(
      '.main-toolbar button.toolbar-leading'
    ) as HTMLButtonElement;
    expect(menuButton).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.main-toolbar a.brand-link[href="/"]')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.main-toolbar .toolbar-brand-logo')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.main-toolbar a.toolbar-leading')).toBeNull();

    menuButton.click();
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('mat-sidenav.mat-drawer-opened')).not.toBeNull();
    expect(fixture.nativeElement.querySelector(
      '.primary-navigation a[href="/report/jobexecution"]'
    )).not.toBeNull();
  });

  it('shows persistent section navigation in the expanded layout', async () => {
    setNavigationWidth(true, true);
    await router.navigateByUrl('/config');
    fixture.detectChanges();
    await fixture.whenStable();

    expect(fixture.nativeElement.querySelector('mat-sidenav.mat-drawer-opened')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.primary-navigation a[href="/config"]')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.primary-navigation a[href="/config/seed"]')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.drawer-back')).toBeNull();
    expect(fixture.nativeElement.querySelector('.main-toolbar')).toBeNull();
    expect(fixture.nativeElement.querySelector('.rail-menu')).toBeNull();
    const homeLinks = fixture.nativeElement.querySelectorAll('.navigation-rail a[href="/"]');
    expect(homeLinks.length).toBe(1);
    expect(homeLinks[0].querySelector('.grouse-icon')).not.toBeNull();
  });

  it('gives every rail action an icon and accessible label', () => {
    can.mockImplementation((_action: string, subject: string) =>
      ['configs', 'SEED', 'report'].includes(subject));
    setNavigationWidth(true, false);

    expect(fixture.nativeElement.querySelector('.navigation-rail a[href="/report"]')).toBeNull();
    const actions = Array.from(
      fixture.nativeElement.querySelectorAll('.rail-actions .rail-action')
    ) as HTMLButtonElement[];
    expect(actions).toHaveLength(2);
    expect(actions[0].getAttribute('aria-label')).toBe('Crawljob schedule');
    for (const action of actions) {
      expect(action.querySelector('mat-icon')).not.toBeNull();
      expect(action.getAttribute('aria-label')).not.toBe('');
    }
  });

  it('omits destinations and actions from the main drawer when the rail already provides them', async () => {
    can.mockImplementation((_action: string, subject: string) =>
      ['configs', 'SEED', 'report'].includes(subject));
    fixture.destroy();
    fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    setNavigationWidth(true, false);

    (fixture.nativeElement.querySelector('.rail-menu') as HTMLButtonElement).click();
    fixture.detectChanges();
    await fixture.whenStable();

    let drawerText = (fixture.nativeElement.querySelector('.primary-navigation') as HTMLElement).textContent;
    expect(drawerText).not.toContain('Home');
    expect(drawerText).not.toContain('Crawljob schedule');
    expect(drawerText).not.toContain('LOGIN');
    expect(drawerText).not.toContain('Log out');
    expect(drawerText).toContain('Config');
    expect(drawerText).toContain('Reports');

    const authService = TestBed.inject(AuthService) as unknown as {isLoggedIn: boolean};
    authService.isLoggedIn = true;
    fixture.destroy();
    fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.rail-menu') as HTMLButtonElement).click();
    fixture.detectChanges();
    await fixture.whenStable();

    drawerText = (fixture.nativeElement.querySelector('.primary-navigation') as HTMLElement).textContent;
    expect(drawerText).not.toContain('Log out');
    expect(fixture.nativeElement.querySelector('.rail-actions [aria-label="Log out"]')).not.toBeNull();
  });

  it('places compact actions in the main drawer without a toolbar overflow menu', async () => {
    can.mockImplementation((_action: string, subject: string) =>
      ['configs', 'SEED', 'report'].includes(subject));
    fixture.destroy();
    fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.toolbar-overflow')).toBeNull();
    expect(fixture.nativeElement.querySelector('.main-toolbar mat-icon')?.textContent)
      .not.toContain('more_vert');

    const drawerItems = Array.from(
      fixture.nativeElement.querySelectorAll('.primary-navigation > .mat-mdc-list-item')
    ) as HTMLElement[];
    expect(drawerItems.map(item => item.textContent.trim())).toEqual([
      expect.stringContaining('Home'),
      expect.stringContaining('Config'),
      expect.stringContaining('Reports'),
      expect.stringContaining('Crawljob schedule'),
      expect.stringContaining('LOGIN'),
    ]);
    expect(drawerItems.at(-2)?.querySelector('mat-icon')?.textContent).toContain('calendar_month');
    expect(drawerItems.at(-1)?.querySelector('mat-icon')?.textContent).toContain('account_box');
    expectDrawerActionStyle(drawerItems.at(-2) as HTMLButtonElement);
    expectDrawerActionStyle(drawerItems.at(-1) as HTMLButtonElement);
  });

  it('shows log out below the primary destinations when signed in', () => {
    can.mockImplementation((_action: string, subject: string) =>
      ['configs', 'SEED', 'report'].includes(subject));
    const authService = TestBed.inject(AuthService) as unknown as {isLoggedIn: boolean};
    authService.isLoggedIn = true;
    fixture.destroy();
    fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();

    const drawerItems = Array.from(
      fixture.nativeElement.querySelectorAll('.primary-navigation > .mat-mdc-list-item')
    ) as HTMLElement[];
    expect(drawerItems.at(-1)?.textContent.trim()).toContain('Log out');
    expect(drawerItems.at(-1)?.querySelector('mat-icon')?.textContent).toContain('logout');
    expect(getComputedStyle(drawerItems.at(-1) as HTMLElement).marginTop).not.toBe('auto');
    expect(drawerItems.some(item => item.textContent.includes('LOGIN'))).toBe(false);
    expectDrawerActionStyle(drawerItems.at(-2) as HTMLButtonElement);
    expectDrawerActionStyle(drawerItems.at(-1) as HTMLButtonElement);
  });

  it('shows loading while the selected destination resolver is pending', async () => {
    await router.navigateByUrl('/config');
    fixture.detectChanges();
    await fixture.whenStable();

    const menuButton = fixture.nativeElement.querySelector('button.toolbar-leading') as HTMLButtonElement;
    menuButton.click();
    fixture.detectChanges();
    await fixture.whenStable();

    const seedLink = fixture.nativeElement.querySelector('a[href="/config/seed"]') as HTMLAnchorElement;
    const navigationEnd = firstValueFrom(router.events.pipe(
      filter(event => event instanceof NavigationEnd),
      take(1),
    ));
    seedLink.click();
    await Promise.resolve();
    fixture.detectChanges();

    const loader = fixture.nativeElement.querySelector('.module-loader') as HTMLElement;
    expect(router.currentNavigation()).not.toBeNull();
    expect(loader.getAttribute('role')).toBe('status');
    expect(loader.textContent).toContain('Loading...');

    seedResolution.next({});
    seedResolution.complete();
    await navigationEnd;
    await fixture.whenStable();
    fixture.detectChanges();

    expect(router.url).toBe('/config/seed');
    expect(fixture.nativeElement.querySelector('[data-testid="route-content"]')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.module-loader')).toBeNull();
  });

  it('clears loading when a pending navigation is superseded', async () => {
    await router.navigateByUrl('/config');
    fixture.detectChanges();
    await fixture.whenStable();

    const pendingNavigation = router.navigateByUrl('/config/seed');
    await Promise.resolve();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.module-loader')).not.toBeNull();

    const replacementNavigation = router.navigateByUrl('/report');
    await Promise.all([pendingNavigation, replacementNavigation]);
    await fixture.whenStable();
    fixture.detectChanges();

    expect(router.url).toBe('/report');
    expect(router.currentNavigation()).toBeNull();
    expect(fixture.nativeElement.querySelector('.module-loader')).toBeNull();
  });
});

function expectDrawerActionStyle(button: HTMLButtonElement): void {
  const style = getComputedStyle(button);

  expect(style.appearance).toBe('none');
  expect(style.borderStyle).toBe('none');
  expect(style.backgroundColor).toBe('rgba(0, 0, 0, 0)');
  expect(style.textAlign).toBe('left');
}
