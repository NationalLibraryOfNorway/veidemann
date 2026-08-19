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
  IsActiveMatchOptions,
  NavigationEnd,
  Router,
  RouterLink,
  RouterLinkActive,
  RouterOutlet
} from '@angular/router';
import {filter, map, startWith, tap} from 'rxjs/operators';
import {AbilityServiceSignal} from "@casl/angular";
import {MongoAbility} from '@casl/ability';

import {AuthService, SnackBarService} from '../../core';
import {MatToolbar} from '@angular/material/toolbar';
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
const PERSISTENT_DRAWER_BREAKPOINT = '(min-width: 1921px)';

interface NavigationDestination {
  readonly route: string;
  readonly label: string;
  readonly permissionSubject?: string;
}

interface PrimaryDestination extends NavigationDestination {
  readonly icon: string;
  readonly section?: NavigationSection;
}

interface RailDestination extends NavigationDestination {
  readonly icon: string;
  readonly sectionPermissionSubject?: string;
  readonly queryParams?: Readonly<Record<string, string>>;
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
  private errorHandler = inject(ErrorHandler);
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);
  private breakpointObserver = inject(BreakpointObserver);

  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];
  readonly openNavigationLabel = $localize`:@@primaryNavigationOpenLabel:Open navigation`;
  readonly closeNavigationLabel = $localize`:@@primaryNavigationCloseLabel:Close navigation`;
  readonly showMainNavigationLabel = $localize`:@@primaryNavigationShowMainLabel:Show main navigation`;
  readonly showSectionNavigationLabel = $localize`:@@primaryNavigationShowSectionLabel:Show section navigation`;
  readonly mainNavigationLabel = $localize`:@@primaryNavigationMainLabel:Main navigation`;

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

  readonly railDestinations: readonly RailDestination[] = [
    {
      route: '/',
      icon: 'home',
      label: $localize`:@@mainMenuHome:Home`,
    },
    {
      route: '/config/entity',
      icon: 'business',
      label: $localize`:@@railNavigationEntities:Entities`,
      sectionPermissionSubject: 'configs',
      permissionSubject: 'CRAWLENTITY',
    },
    {
      route: '/config/seed',
      icon: 'link',
      label: $localize`:@@railNavigationSeeds:Seeds`,
      sectionPermissionSubject: 'configs',
      permissionSubject: 'SEED',
    },
    {
      route: '/config/crawljobs',
      icon: 'work',
      label: $localize`:@@railNavigationCrawlJobs:Crawl jobs`,
      sectionPermissionSubject: 'configs',
      permissionSubject: 'CRAWLJOB',
    },
    {
      route: '/report/jobexecution',
      icon: 'hdr_strong',
      label: $localize`:@@railNavigationJobs:Jobs`,
      sectionPermissionSubject: 'report',
      permissionSubject: 'jobexecution',
      queryParams: {sort: 'startTime:desc'},
    },
    {
      route: '/report/crawlexecution',
      icon: 'hdr_weak',
      label: $localize`:@@railNavigationCrawls:Crawls`,
      sectionPermissionSubject: 'report',
      permissionSubject: 'crawlexecution',
      queryParams: {sort: 'startTime:desc'},
    },
  ];

  readonly exactRailLinkMatchOptions: IsActiveMatchOptions = {
    paths: 'exact',
    queryParams: 'ignored',
    matrixParams: 'ignored',
    fragment: 'ignored',
  };
  readonly childRailLinkMatchOptions: IsActiveMatchOptions = {
    ...this.exactRailLinkMatchOptions,
    paths: 'subset',
  };

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
  readonly expandedDrawerLevel = signal<DrawerLevel | null>(null);
  readonly visibleDrawerLevel: Signal<DrawerLevel>;
  readonly navigationMenuLabel: Signal<string>;
  readonly mainNavigationVisible: Signal<boolean>;
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
      tap(() => this.expandedDrawerLevel.set(null)),
    ), {requireSync: true});
    const currentPath = computed(() => currentUrl().split(/[?#]/, 1)[0]);
    this.activeSection = computed(() => currentPath() === '/config' || currentPath().startsWith('/config/')
      ? 'config'
      : currentPath() === '/report' || currentPath().startsWith('/report/') ? 'report' : null);
    this.visibleDrawerLevel = computed(() => this.expandedNavigation()
      ? this.expandedDrawerLevel() ?? this.activeSection() ?? 'main'
      : this.drawerLevel());
    this.mainNavigationVisible = computed(() => this.visibleDrawerLevel() === 'main');
    this.navigationMenuLabel = computed(() => {
      if (!this.expandedNavigation()) {
        return this.compactDrawerOpen() ? this.closeNavigationLabel : this.openNavigationLabel;
      }
      if (!this.activeSection()) {
        return this.mainNavigationLabel;
      }
      return this.mainNavigationVisible() ? this.showSectionNavigationLabel : this.showMainNavigationLabel;
    });
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

  canNavigateRail(destination: RailDestination): boolean {
    return (!destination.sectionPermissionSubject || this.can('read', destination.sectionPermissionSubject))
      && this.canNavigate(destination);
  }

  togglePrimaryDrawer(): void {
    if (this.expandedNavigation()) {
      const contextualLevel = this.activeSection() ?? 'main';
      this.expandedDrawerLevel.set(this.mainNavigationVisible() ? contextualLevel : 'main');
      return;
    }
    const open = !this.compactDrawerOpen();
    if (open) {
      this.drawerLevel.set(this.activeSection() ?? 'main');
    }
    this.compactDrawerOpen.set(open);
  }

  closePrimaryDrawer(): void {
    this.compactDrawerOpen.set(false);
    this.expandedDrawerLevel.set(null);
  }

  showDrawerLevel(level: DrawerLevel): void {
    if (this.expandedNavigation()) {
      this.expandedDrawerLevel.set(level);
    } else {
      this.drawerLevel.set(level);
    }
  }

  onLogin() {
    this.authService.login(this.route.snapshot.url.join('/'));
  }

  onLogout() {
    this.authService.logout();
    this.router.navigate(['/'], { relativeTo: this.route.root })
      .then(() => this.snackBarService.openSnackBar($localize`:@snackBarMessage.loggedOut:You are now logged out`));
  }

}
