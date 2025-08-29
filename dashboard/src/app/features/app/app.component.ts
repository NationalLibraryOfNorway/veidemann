import {AfterViewInit, ChangeDetectionStrategy, Component, OnInit} from '@angular/core';
import {
  ActivatedRoute,
  RouteConfigLoadEnd,
  RouteConfigLoadStart,
  Router,
  RouterLink,
  RouterLinkActive,
  RouterOutlet
} from '@angular/router';
import {MatDialog} from '@angular/material/dialog';
import {Observable} from 'rxjs';
import {filter, map} from 'rxjs/operators';
import {AbilityService} from "@casl/angular";

import {AuthService, ControllerApiService, ErrorService, GuardService, SnackBarService} from '../../core';
import {AboutDialogComponent} from './about-dialog/about-dialog.component';
import {KeyboardShortcutsModule, ShortcutEventOutput, ShortcutInput} from 'ng-keyboard-shortcuts';
import {ScheduleOverviewComponent} from './schedule-overview/schedule-overview.component';
import {AsyncPipe} from '@angular/common';
import {MatToolbar} from '@angular/material/toolbar';
import {TimeComponent} from './time/time.component';
import {MatMenuModule} from '@angular/material/menu';
import {FlexLayoutModule} from '@angular/flex-layout';
import {MatIcon} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {MatTabsModule} from '@angular/material/tabs';
import {DialogComponent} from './dialog/dialog.component';
import {MatTooltip} from '@angular/material/tooltip';
import {MatProgressSpinner} from '@angular/material/progress-spinner';

@Component({
  selector: 'app',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    AsyncPipe,
    FlexLayoutModule,
    KeyboardShortcutsModule,
    MatButtonModule,
    MatIcon,
    MatMenuModule,
    MatTabsModule,
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
export class AppComponent implements OnInit, AfterViewInit {
  readonly ability$: Observable<any>;

  isModuleLoading$: Observable<boolean>;
  private moduleLoadSemaphore = 0;
  shortcuts: ShortcutInput[] = [];

  constructor(private authService: AuthService,
    private controllerApiService: ControllerApiService,
    private router: Router,
    private route: ActivatedRoute,
    private guardService: GuardService,
    private snackBarService: SnackBarService,
    private dialog: MatDialog,
    private errorService: ErrorService,
    private abilityService: AbilityService<any>) {
    this.ability$ = this.abilityService.ability$;
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
        const queryParams = {};
        // @ts-ignore
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

  ngAfterViewInit(): void {
    this.shortcuts.push(
      {
        key: 'c e',
        label: 'Configurations',
        description: 'Show entity view',
        command: (output: ShortcutEventOutput) => this.router.navigate(['/config/entity'])
      },
      {
        key: 'c s',
        label: 'Configurations',
        description: 'Show seed view',
        command: (output: ShortcutEventOutput) => this.router.navigate(['/config/seed'])
      },
      {
        key: 'c j',
        label: 'Configurations',
        description: 'Show crawljob view',
        command: (output: ShortcutEventOutput) => this.router.navigate(['/config/crawljobs'])
      },
      {
        key: 'c s c',
        label: 'Configurations',
        description: 'Show schedule view',
        command: (output: ShortcutEventOutput) => this.router.navigate(['/config/schedule'])
      },
      {
        key: 'c r',
        label: 'Configurations',
        description: 'Show crawlconfig view',
        command: (output: ShortcutEventOutput) => this.router.navigate(['/config/crawlconfig'])
      },
      {
        key: 'c o',
        label: 'Configurations',
        description: 'Show collection view',
        command: (output: ShortcutEventOutput) => this.router.navigate(['/config/collection'])
      },
      {
        key: 'c b',
        label: 'Configurations',
        description: 'Show browserconfig view',
        command: (output: ShortcutEventOutput) => this.router.navigate(['/config/browserconfig'])
      },
      {
        key: 'c b s',
        label: 'Configurations',
        description: 'Show browserscript view',
        command: (output: ShortcutEventOutput) => this.router.navigate(['/config/browserscript'])
      },
      {
        key: 'c p',
        label: 'Configurations',
        description: 'Show politeness view',
        command: (output: ShortcutEventOutput) => this.router.navigate(['/config/politenessconfig'])
      },
      {
        key: 'c h',
        label: 'Configurations',
        description: 'Show crawlhostgroup view',
        command: (output: ShortcutEventOutput) => this.router.navigate(['/config/crawlhostgroupconfig'])
      },
      {
        key: 'c u',
        label: 'Configurations',
        description: 'Show users view',
        command: (output: ShortcutEventOutput) => this.router.navigate(['/config/rolemapping'])
      },
      {
        key: 'r j',
        label: 'Reports',
        description: 'Show jobexecution reports',
        command: (output: ShortcutEventOutput) => this.router.navigate(['/report/jobexecution'])
      },
      {
        key: 'r c',
        label: 'Reports',
        description: 'Show crawlexecution reports',
        command: (output: ShortcutEventOutput) => this.router.navigate(['/report/crawlexecution'])
      },
      {
        key: 'r p',
        label: 'Reports',
        description: 'Show pagelog reports',
        command: (output: ShortcutEventOutput) => this.router.navigate(['/report/pagelog'])
      },
      {
        key: 'r c l',
        label: 'Reports',
        description: 'Show crawllog reports',
        command: (output: ShortcutEventOutput) => this.router.navigate(['/report/crawllog'])
      }
    );
  }

  get isLoggedIn(): boolean {
    return this.authService.isLoggedIn;
  }

  get name(): string {
    return this.authService.name;
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
