import {ChangeDetectionStrategy, Component, computed, inject, Signal} from '@angular/core';
import {ActivatedRoute, NavigationEnd, Router, RouterOutlet} from '@angular/router';

import {filter, map, startWith, tap} from 'rxjs/operators';

import {
  BrowserScriptType,
  ConfigObject,
  Kind,
  RobotsPolicy,
  Role,
  RotationPolicy,
  SubCollectionType
} from '../../shared/models';
import {OptionsService} from './services';
import {configKindFromPath, ConfigPath} from './func';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {SectionHeaderComponent} from '../../shared/components/section-header/section-header.component';

export interface ConfigOptions {
  rotationPolicies?: RotationPolicy[];
  subCollectionTypes?: SubCollectionType[];
  crawlConfigs?: ConfigObject[];
  crawlScheduleConfigs?: ConfigObject[];
  browserConfigs?: ConfigObject[];
  collections?: ConfigObject[];
  politenessConfigs?: ConfigObject[];
  browserScripts?: ConfigObject[];
  browserScriptTypes?: BrowserScriptType[];
  robotsPolicies?: RobotsPolicy[];
  crawlJobs?: ConfigObject[];
  roles?: Role[];
  scopeScripts?: ConfigObject[];
}

@Component({
  templateUrl: './config.html',
  styleUrls: ['./config.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    SectionHeaderComponent,
    RouterOutlet
  ],
  standalone: true
})
export class ConfigComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly optionsService = inject(OptionsService);

  readonly kind: Signal<Kind>;
  readonly kindTitle: Signal<string>;
  readonly isDetail: Signal<boolean>;
  readonly backLink: Signal<string[]>;
  readonly listLink: Signal<string[]>;
  readonly configurationLabel = $localize`:@@configurationPageTitle:Configuration`;

  constructor() {
    this.kind = toSignal(
      this.route.paramMap.pipe(map(params => configKindFromPath(params.get('kind')))),
      {requireSync: true}
    );
    this.kindTitle = computed(() => this.getKindTitle(this.kind()));
    this.isDetail = toSignal(
      this.router.events.pipe(
        filter(event => event instanceof NavigationEnd),
        startWith(null),
        map(() => !!this.route.firstChild?.snapshot.paramMap.get('id')),
      ),
      {requireSync: true}
    );
    this.listLink = computed(() => ['/config', ConfigPath[this.kind()]]);
    this.backLink = computed(() => this.isDetail() ? this.listLink() : ['/config']);

    this.route.data.pipe(
      map(data => data['options']),
      tap(options => this.optionsService.next(options)),
      takeUntilDestroyed(),
    ).subscribe();
  }

  private getKindTitle(kind: Kind): string {
    switch (kind) {
      case Kind.CRAWLENTITY:
        return $localize`:@@configurationSidebarMenuEntity:Entity`;
      case Kind.SEED:
        return $localize`:@@configurationSidebarMenuSeed:Seed`;
      case Kind.CRAWLJOB:
        return $localize`:@@configurationSidebarMenuCrawljobs:Crawl jobs`;
      case Kind.CRAWLSCHEDULECONFIG:
        return $localize`:@@configurationSidebarMenuSchedule:Schedule`;
      case Kind.CRAWLCONFIG:
        return $localize`:@@configurationSidebarMenuCrawlconfig:Crawl config`;
      case Kind.COLLECTION:
        return $localize`:@@configurationSidebarMenuCollection:Collection`;
      case Kind.BROWSERCONFIG:
        return $localize`:@@configurationSidebarMenuBrowserconfig:Browser config`;
      case Kind.BROWSERSCRIPT:
        return $localize`:@@configurationSidebarMenuBrowserscript:Browser script`;
      case Kind.POLITENESSCONFIG:
        return $localize`:@@configurationSidebarMenuPolitenessconfig:Politeness`;
      case Kind.CRAWLHOSTGROUPCONFIG:
        return $localize`:@@configurationSidebarMenuCrawlhostgroupconfig:Crawl host group`;
      case Kind.ROLEMAPPING:
        return $localize`:@@configurationSidebarMenuRolemapping:Users`;
      default:
        return this.configurationLabel;
    }
  }
}
