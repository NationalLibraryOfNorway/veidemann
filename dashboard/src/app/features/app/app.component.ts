import { ChangeDetectionStrategy, Component, computed, OnInit, Signal, inject } from '@angular/core';
import {BreakpointObserver} from '@angular/cdk/layout';
import {
  ActivatedRoute,
  NavigationEnd,
  RouteConfigLoadEnd,
  RouteConfigLoadStart,
  Router,
  RouterLink,
  RouterLinkActive,
  RouterOutlet
} from '@angular/router';
import {MatDialog} from '@angular/material/dialog';
import {Observable} from 'rxjs';
import {filter, map, startWith} from 'rxjs/operators';
import {AbilityServiceSignal} from "@casl/angular";
import {MongoAbility} from '@casl/ability';

import {AuthService, ErrorService, SnackBarService} from '../../core';
import {AboutDialogComponent} from './about-dialog/about-dialog.component';
import {ScheduleOverviewComponent} from './schedule-overview/schedule-overview.component';
import {AsyncPipe} from '@angular/common';
import {MatToolbar} from '@angular/material/toolbar';
import {TimeComponent} from './time/time.component';
import {MatMenuModule} from '@angular/material/menu';
import {MatIcon} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {DialogComponent} from './dialog/dialog.component';
import {MatTooltip} from '@angular/material/tooltip';
import {MatProgressSpinner} from '@angular/material/progress-spinner';
import {MatSidenavModule} from '@angular/material/sidenav';
import {MatListModule} from '@angular/material/list';
import {toSignal} from '@angular/core/rxjs-interop';

interface PrimaryDestination {
  readonly route: string;
  readonly icon: string;
  readonly label: string;
  readonly permissionSubject?: string;
}

@Component({
  selector: 'app-shell',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    AsyncPipe,
    MatButtonModule,
    MatIcon,
    MatListModule,
    MatMenuModule,
    MatSidenavModule,
    MatToolbar,
    TimeComponent,
    RouterLink,
    RouterLinkActive,
    MatTooltip,
    MatProgressSpinner,
    RouterOutlet,
    DialogComponent
  ]
})
export class AppComponent implements OnInit {
  private authService = inject(AuthService);
  private router = inject(Router);
  private route = inject(ActivatedRoute);
  private snackBarService = inject(SnackBarService);
  private dialog = inject(MatDialog);
  private errorService = inject(ErrorService);
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);
  private breakpointObserver = inject(BreakpointObserver);

  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  readonly primaryDestinations: readonly PrimaryDestination[] = [
    {route: '/', icon: 'home', label: $localize`:@@mainMenuDashboard:Dashboard`},
    {
      route: '/config',
      icon: 'settings',
      label: $localize`:@@mainMenuConfiguration:CONFIGURATION`,
      permissionSubject: 'configs',
    },
    {
      route: '/report',
      icon: 'assessment',
      label: $localize`:@@mainMenuReport:REPORT`,
      permissionSubject: 'report',
    },
    {
      route: '/logconfig',
      icon: 'notes',
      label: $localize`:@@mainMenuLogConfiguration:LOG LEVEL`,
      permissionSubject: 'logconfig',
    },
  ];

  readonly configDestinations: readonly PrimaryDestination[] = [
    {route: '/config/entity', icon: 'business', label: 'Entity', permissionSubject: 'CRAWLENTITY'},
    {route: '/config/seed', icon: 'link', label: 'Seed', permissionSubject: 'SEED'},
    {route: '/config/crawljobs', icon: 'work', label: 'Crawljobs', permissionSubject: 'CRAWLJOB'},
    {route: '/config/schedule', icon: 'schedule', label: 'Schedule', permissionSubject: 'CRAWLSCHEDULECONFIG'},
    {route: '/config/crawlconfig', icon: 'settings_system_daydream', label: 'CrawlConfig', permissionSubject: 'CRAWLCONFIG'},
    {route: '/config/collection', icon: 'collections_bookmark', label: 'Collection', permissionSubject: 'COLLECTION'},
    {route: '/config/browserconfig', icon: 'web', label: 'BrowserConfig', permissionSubject: 'BROWSERCONFIG'},
    {route: '/config/browserscript', icon: 'web_asset', label: 'BrowserScript', permissionSubject: 'BROWSERSCRIPT'},
    {route: '/config/politenessconfig', icon: 'sentiment_satisfied', label: 'Politeness', permissionSubject: 'POLITENESSCONFIG'},
    {route: '/config/crawlhostgroupconfig', icon: 'group_work', label: 'CrawlHostGroup', permissionSubject: 'CRAWLHOSTGROUPCONFIG'},
    {route: '/config/rolemapping', icon: 'people', label: 'Users', permissionSubject: 'ROLEMAPPING'},
  ];

  readonly reportDestinations: readonly PrimaryDestination[] = [
    {route: '/report/jobexecution', icon: 'hdr_strong', label: 'JobExecution', permissionSubject: 'jobexecution'},
    {route: '/report/crawlexecution', icon: 'hdr_weak', label: 'CrawlExecution', permissionSubject: 'crawlexecution'},
    {route: '/report/pagelog', icon: 'art_track', label: 'PageLog', permissionSubject: 'pagelog'},
    {route: '/report/crawllog', icon: 'event_note', label: 'CrawlLog', permissionSubject: 'crawllog'},
  ];

  readonly expandedNavigation: Signal<boolean>;
  readonly activeSection: Signal<'config' | 'report' | null>;

  isModuleLoading$: Observable<boolean>;
  private moduleLoadSemaphore = 0;

  constructor() {
    this.can = this.abilityService.can;
    this.expandedNavigation = toSignal(
      this.breakpointObserver.observe('(min-width: 1200px)').pipe(map(state => state.matches)),
      {initialValue: false}
    );
    const currentUrl = toSignal(this.router.events.pipe(
      filter(event => event instanceof NavigationEnd),
      map(() => this.router.url),
      startWith(this.router.url),
    ), {requireSync: true});
    this.activeSection = computed(() => currentUrl().startsWith('/config')
      ? 'config'
      : currentUrl().startsWith('/report') ? 'report' : null);
    this.isModuleLoading$ = this.router.events.pipe(
      filter(event => event instanceof RouteConfigLoadStart || event instanceof RouteConfigLoadEnd),
      map(event => {
        if (event instanceof RouteConfigLoadStart) {
          this.moduleLoadSemaphore++;
        } else if (event instanceof RouteConfigLoadEnd) {
          this.moduleLoadSemaphore--;
        }
        return this.moduleLoadSemaphore > 0;
      }),
    );
  }

  ngOnInit(): void {
    if (this.isLoggedIn && this.authService.requestedUri) {
      try {
        const url: URL = new URL(this.authService.requestedUri, 'http://localhost');
        const queryParams: Record<string, string> = {};
        for (const [key, value] of url.searchParams) {
          queryParams[key] = value;
        }
        const fragment = url.hash.substring(1) || null;
        const commands = url.pathname.split('/');
        this.router.navigate(commands, { queryParams, fragment, replaceUrl: true })
          .catch(e => this.errorService.dispatch(e));
      } catch (e) {
        this.errorService.dispatch(e);
      }
    }

  }

  get isLoggedIn(): boolean {
    return this.authService.isLoggedIn;
  }

  get name(): string {
    return this.authService.name;
  }

  canNavigate(destination: PrimaryDestination): boolean {
    return !destination.permissionSubject || this.can('read', destination.permissionSubject);
  }

  onLogin() {
    this.authService.login(this.route.snapshot.url.join('/'));
  }

  onLogout() {
    this.authService.logout();
    this.router.navigate(['/'], { relativeTo: this.route.root })
      .then(() => this.snackBarService.openSnackBar($localize`:@snackBarMessage.loggedOut:You are now logged out`));
  }

  onAbout() {
    this.dialog.open(AboutDialogComponent);
  }

  onShowJobSchedule() {
    this.dialog.open(ScheduleOverviewComponent, {
      maxWidth: '95vw',
      maxHeight: '95vh',
      height: '95%',
      width: '95%',
      autoFocus: true,
      panelClass: 'calendar'
    });
  }
}
