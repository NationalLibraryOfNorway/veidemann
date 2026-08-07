import {
  ChangeDetectionStrategy,
  Component,
  computed,
  ErrorHandler,
  inject,
  OnInit,
  Signal,
  signal
} from '@angular/core';
import {BreakpointObserver} from '@angular/cdk/layout';
import {
  ActivatedRoute,
  NavigationEnd,
  Router,
  RouterLink,
  RouterLinkActive,
  RouterOutlet
} from '@angular/router';
import {MatDialog} from '@angular/material/dialog';
import {filter, map, startWith} from 'rxjs/operators';
import {AbilityServiceSignal} from "@casl/angular";
import {MongoAbility} from '@casl/ability';

import {AuthService, SnackBarService} from '../../core';
import {AboutDialogComponent} from './about-dialog/about-dialog.component';
import {ScheduleOverviewComponent} from './schedule-overview/schedule-overview.component';
import {MatToolbar} from '@angular/material/toolbar';
import {MatMenuModule} from '@angular/material/menu';
import {MatIcon} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {MatTooltip} from '@angular/material/tooltip';
import {MatProgressSpinner} from '@angular/material/progress-spinner';
import {MatSidenavModule} from '@angular/material/sidenav';
import {MatListModule} from '@angular/material/list';
import {toSignal} from '@angular/core/rxjs-interop';

type NavigationSection = 'config' | 'report';
type DrawerLevel = 'main' | NavigationSection;

const RAIL_BREAKPOINT = '(min-width: 840px)';
const PERSISTENT_DRAWER_BREAKPOINT = '(min-width: 1200px)';

interface NavigationDestination {
  readonly route: string;
  readonly label: string;
  readonly permissionSubject?: string;
}

interface PrimaryDestination extends NavigationDestination {
  readonly icon: string;
  readonly section?: NavigationSection;
}

@Component({
  selector: 'app-shell',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    MatButtonModule,
    MatIcon,
    MatListModule,
    MatMenuModule,
    MatSidenavModule,
    MatToolbar,
    RouterLink,
    RouterLinkActive,
    MatTooltip,
    MatProgressSpinner,
    RouterOutlet
  ]
})
export class AppComponent implements OnInit {
  private authService = inject(AuthService);
  private router = inject(Router);
  private route = inject(ActivatedRoute);
  private snackBarService = inject(SnackBarService);
  private dialog = inject(MatDialog);
  private errorHandler = inject(ErrorHandler);
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);
  private breakpointObserver = inject(BreakpointObserver);

  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  readonly primaryDestinations: readonly PrimaryDestination[] = [
    {
      route: '/',
      icon: 'home',
      label: $localize`:@@mainMenuHome:Home`,
    },
    {
      route: '/config',
      icon: 'settings',
      label: $localize`:@@mainMenuConfiguration:Configuration`,
      section: 'config',
      permissionSubject: 'configs',
    },
    {
      route: '/report',
      icon: 'assessment',
      label: $localize`:@@mainMenuReport:Reports`,
      section: 'report',
      permissionSubject: 'report',
    },
  ];

  readonly configDestinations: readonly NavigationDestination[] = [
    {route: '/config/entity', label: 'Entity', permissionSubject: 'CRAWLENTITY'},
    {route: '/config/seed', label: 'Seed', permissionSubject: 'SEED'},
    {route: '/config/crawljobs', label: 'Crawl jobs', permissionSubject: 'CRAWLJOB'},
    {route: '/config/schedule', label: 'Schedule', permissionSubject: 'CRAWLSCHEDULECONFIG'},
    {route: '/config/crawlconfig', label: 'Crawl config', permissionSubject: 'CRAWLCONFIG'},
    {route: '/config/collection', label: 'Collection', permissionSubject: 'COLLECTION'},
    {route: '/config/browserconfig', label: 'Browser config', permissionSubject: 'BROWSERCONFIG'},
    {route: '/config/browserscript', label: 'Browser script', permissionSubject: 'BROWSERSCRIPT'},
    {route: '/config/politenessconfig', label: 'Politeness', permissionSubject: 'POLITENESSCONFIG'},
    {route: '/config/crawlhostgroupconfig', label: 'Crawl host group', permissionSubject: 'CRAWLHOSTGROUPCONFIG'},
    {route: '/config/rolemapping', label: 'Users', permissionSubject: 'ROLEMAPPING'},
  ];

  readonly reportDestinations: readonly NavigationDestination[] = [
    {route: '/report/jobexecution', label: 'Job execution', permissionSubject: 'jobexecution'},
    {route: '/report/crawlexecution', label: 'Crawl execution', permissionSubject: 'crawlexecution'},
    {route: '/report/pagelog', label: 'Page log', permissionSubject: 'pagelog'},
    {route: '/report/crawllog', label: 'Crawl log', permissionSubject: 'crawllog'},
  ];

  readonly railNavigation: Signal<boolean>;
  readonly expandedNavigation: Signal<boolean>;
  readonly activeSection: Signal<NavigationSection | null>;
  readonly compactDrawerOpen = signal(false);
  readonly drawerLevel = signal<DrawerLevel>('main');
  readonly visibleDrawerLevel: Signal<DrawerLevel>;
  readonly isNavigating: Signal<boolean>;

  constructor() {
    this.can = this.abilityService.can;
    const navigationBreakpoints = toSignal(this.breakpointObserver.observe([
      RAIL_BREAKPOINT,
      PERSISTENT_DRAWER_BREAKPOINT,
    ]), {
      initialValue: {
        matches: false,
        breakpoints: {
          [RAIL_BREAKPOINT]: false,
          [PERSISTENT_DRAWER_BREAKPOINT]: false,
        },
      },
    });
    this.railNavigation = computed(() => navigationBreakpoints().breakpoints[RAIL_BREAKPOINT]);
    this.expandedNavigation = computed(() => navigationBreakpoints().breakpoints[PERSISTENT_DRAWER_BREAKPOINT]);
    const currentUrl = toSignal(this.router.events.pipe(
      filter(event => event instanceof NavigationEnd),
      map(() => this.router.url),
      startWith(this.router.url),
    ), {requireSync: true});
    const currentPath = computed(() => currentUrl().split(/[?#]/, 1)[0]);
    this.activeSection = computed(() => currentPath() === '/config' || currentPath().startsWith('/config/')
      ? 'config'
      : currentPath() === '/report' || currentPath().startsWith('/report/') ? 'report' : null);
    this.visibleDrawerLevel = computed(() => this.expandedNavigation()
      ? this.activeSection() ?? 'main'
      : this.drawerLevel());
    this.isNavigating = computed(() => this.router.currentNavigation() !== null);
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
          .catch(e => this.errorHandler.handleError(e));
      } catch (e) {
        this.errorHandler.handleError(e);
      }
    }

  }

  get isLoggedIn(): boolean {
    return this.authService.isLoggedIn;
  }

  get name(): string {
    return this.authService.name;
  }

  canNavigate(destination: NavigationDestination): boolean {
    return !destination.permissionSubject || this.can('read', destination.permissionSubject);
  }

  togglePrimaryDrawer(): void {
    const open = !this.compactDrawerOpen();
    if (open) {
      this.drawerLevel.set(this.activeSection() ?? 'main');
    }
    this.compactDrawerOpen.set(open);
  }

  closePrimaryDrawer(): void {
    this.compactDrawerOpen.set(false);
  }

  showDrawerLevel(level: DrawerLevel): void {
    this.drawerLevel.set(level);
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
