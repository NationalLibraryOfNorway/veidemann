import {Component, ErrorHandler} from '@angular/core';
import {BreakpointObserver, BreakpointState} from '@angular/cdk/layout';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {NavigationEnd, provideRouter, ResolveFn, Router} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {MatDialog} from '@angular/material/dialog';
import {BehaviorSubject, filter, firstValueFrom, ReplaySubject, take} from 'rxjs';

import {AuthService, SnackBarService} from '../../core';
import {provideMaterialAnimationsDisabled} from '../../core/core.testing.module';
import {AppComponent} from './app.component';

@Component({template: '<p data-testid="route-content">Route content</p>', standalone: true})
class EmptyRouteComponent {}

const RAIL_BREAKPOINT = '(min-width: 840px)';
const PERSISTENT_DRAWER_BREAKPOINT = '(min-width: 1200px)';

describe('AppComponent navigation', () => {
  let fixture: ComponentFixture<AppComponent>;
  let router: Router;
  let breakpointState: BehaviorSubject<BreakpointState>;
  let seedResolution: ReplaySubject<unknown>;
  let dialog: {open: ReturnType<typeof vi.fn>};
  const can = vi.fn((_action: string, subject: string) => ['configs', 'SEED'].includes(subject));

  beforeEach(async () => {
    can.mockImplementation((_action: string, subject: string) => ['configs', 'SEED'].includes(subject));
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
            ],
          },
          {
            path: 'report',
            children: [
              {path: '', component: EmptyRouteComponent},
              {path: 'jobexecution', component: EmptyRouteComponent},
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
    expect(homeLink.classList).toContain('mat-mdc-icon-button');
    const grouseIcon = homeLink.querySelector('img.grouse-icon') as HTMLImageElement;
    expect(grouseIcon).not.toBeNull();
    expect(grouseIcon.getAttribute('src')).toContain('veidemann_grouse_black.png');

    const sectionTriggers = Array.from(
      fixture.nativeElement.querySelectorAll('.drawer-section-trigger')
    ) as HTMLButtonElement[];
    expect(sectionTriggers).toHaveLength(1);
    expect(sectionTriggers[0].textContent).toContain('Configuration');
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
      expect(chevron.classList).toContain('material-icons');
      expect(chevron.classList).not.toContain('mat-mdc-list-item-meta');
    }
  });

  it('shows the rail and keeps section navigation collapsed at middle widths', async () => {
    (fixture.nativeElement.querySelector('button.toolbar-leading') as HTMLButtonElement).click();
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('mat-sidenav.mat-drawer-opened')).not.toBeNull();

    setNavigationWidth(true, false);
    await router.navigateByUrl('/config/entity');
    fixture.detectChanges();
    await fixture.whenStable();

    expect(fixture.nativeElement.querySelector('.main-toolbar')).toBeNull();
    expect(fixture.nativeElement.querySelector('.navigation-rail')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('mat-sidenav.mat-drawer-opened')).toBeNull();
    expect(fixture.nativeElement.querySelector('.navigation-rail a[href="/report"]')).toBeNull();
    const activeLinks = fixture.nativeElement.querySelectorAll(
      '.navigation-rail a[href="/config"][aria-current="page"]'
    );
    expect(activeLinks.length).toBe(1);
    expect(activeLinks[0].classList).toContain('active-rail-destination');
    const homeLinks = fixture.nativeElement.querySelectorAll('.navigation-rail a[href="/"]');
    expect(homeLinks.length).toBe(1);
    expect(homeLinks[0].querySelector('.grouse-icon')).not.toBeNull();
  });

  it('drills into section navigation and returns to the main drawer', () => {
    const drawerTriggers = Array.from(
      fixture.nativeElement.querySelectorAll('.drawer-section-trigger')
    ) as HTMLButtonElement[];
    const configurationButton = drawerTriggers.find(
      element => element.textContent?.includes('Configuration')
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

    sectionTriggers.find(trigger => trigger.textContent?.includes('Configuration'))?.click();
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
    expect(menuButton.querySelector('mat-icon')?.textContent).toContain('menu');
    expect(fixture.nativeElement.querySelector('.main-toolbar a.brand-link[href="/"]')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.main-toolbar .toolbar-title')?.textContent).toContain('Veidemann');
    expect(fixture.nativeElement.querySelector('.main-toolbar a.toolbar-leading')).toBeNull();
    expect(fixture.nativeElement.querySelector('.main-toolbar mat-icon')?.textContent).not.toContain('arrow_back');

    menuButton.click();
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('mat-sidenav.mat-drawer-opened')).not.toBeNull();
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
    expect(fixture.nativeElement.querySelector('.main-toolbar .toolbar-title')?.textContent).toContain('Veidemann');
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
    const homeLinks = fixture.nativeElement.querySelectorAll('.navigation-rail a[href="/"]');
    expect(homeLinks.length).toBe(1);
    expect(homeLinks[0].querySelector('.grouse-icon')).not.toBeNull();
  });

  it('gives every rail action an icon and accessible label', () => {
    can.mockImplementation((_action: string, subject: string) =>
      ['configs', 'SEED', 'report'].includes(subject));
    setNavigationWidth(true, false);

    expect(fixture.nativeElement.querySelector('.navigation-rail a[href="/report"]')).not.toBeNull();
    const actions = Array.from(
      fixture.nativeElement.querySelectorAll('.rail-actions .rail-action')
    ) as HTMLButtonElement[];
    expect(actions).toHaveLength(3);
    expect(actions[0].getAttribute('aria-label')).toBe('Crawljob schedule');
    for (const action of actions) {
      expect(action.querySelector('mat-icon')).not.toBeNull();
      expect(action.getAttribute('aria-label')).not.toBe('');
    }
  });

  it('keeps actions in the compact toolbar overflow menu', async () => {
    can.mockImplementation((_action: string, subject: string) =>
      ['configs', 'SEED', 'report'].includes(subject));
    const overflowButton = fixture.nativeElement.querySelector('.toolbar-overflow') as HTMLButtonElement;
    expect(overflowButton.getAttribute('aria-label')).toBe('More actions');
    expect(overflowButton.querySelector('mat-icon')?.textContent).toContain('more_vert');

    overflowButton.click();
    fixture.detectChanges();
    await fixture.whenStable();

    const menuItems = Array.from(document.querySelectorAll('.mat-mdc-menu-item')) as HTMLElement[];
    expect(menuItems.map(item => item.textContent)).toEqual(expect.arrayContaining([
      expect.stringContaining('Crawljob schedule'),
      expect.stringContaining('Docs'),
      expect.stringContaining('About'),
      expect.stringContaining('LOGIN'),
    ]));
    for (const item of menuItems) {
      expect(item.querySelector('mat-icon')).not.toBeNull();
    }
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
