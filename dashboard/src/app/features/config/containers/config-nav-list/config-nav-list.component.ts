import { ChangeDetectionStrategy,Component,inject } from '@angular/core';
import { MatCardModule } from '@angular/material/card';
import { MatIcon } from '@angular/material/icon';
import { RouterLink } from '@angular/router';
import { MongoAbility } from '@casl/ability';
import { AbilityServiceSignal } from '@casl/angular';
import { NavigationListComponent } from '../../../../shared/components';
import { Kind } from '../../../../shared/models';
import { ConfigPath } from '../../func';

interface ConfigDestination {
  readonly kind: Kind;
  readonly icon: string;
  readonly label: string;
}

interface ConfigDestinationGroup {
  readonly label: string;
  readonly destinations: readonly ConfigDestination[];
}

@Component({
  selector: 'app-config-navigation-list',
  templateUrl: './config-nav-list.component.html',
  styleUrls: ['../../../../shared/components/navigation-grid/navigation-grid.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatIcon,
    MatCardModule,
    RouterLink,
  ],
  standalone: true
})
export class ConfigNavListComponent extends NavigationListComponent {
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  readonly ConfigPath = ConfigPath;
  readonly Kind = Kind;
  readonly destinationGroups: readonly ConfigDestinationGroup[] = [
    {
      label: $localize`:@@configurationGroupCrawlTargets:Crawl targets`,
      destinations: [
        {kind: Kind.CRAWLENTITY, icon: 'business', label: $localize`:@@configurationSidebarMenuEntity:Entity`},
        {kind: Kind.SEED, icon: 'link', label: $localize`:@@configurationSidebarMenuSeed:Seed`},
      ],
    },
    {
      label: $localize`:@@configurationGroupCrawlSetup:Crawl setup`,
      destinations: [
        {kind: Kind.CRAWLJOB, icon: 'work', label: $localize`:@@configurationSidebarMenuCrawljobs:Crawl jobs`},
        {
          kind: Kind.CRAWLSCHEDULECONFIG,
          icon: 'schedule',
          label: $localize`:@@configurationSidebarMenuSchedule:Schedule`,
        },
        {
          kind: Kind.CRAWLCONFIG,
          icon: 'settings_system_daydream',
          label: $localize`:@@configurationSidebarMenuCrawlconfig:Crawl config`,
        },
        {
          kind: Kind.COLLECTION,
          icon: 'collections_bookmark',
          label: $localize`:@@configurationSidebarMenuCollection:Collection`,
        },
        {
          kind: Kind.BROWSERCONFIG,
          icon: 'web',
          label: $localize`:@@configurationSidebarMenuBrowserconfig:Browser config`,
        },
        {
          kind: Kind.BROWSERSCRIPT,
          icon: 'web_asset',
          label: $localize`:@@configurationSidebarMenuBrowserscript:Browser script`,
        },
        {
          kind: Kind.POLITENESSCONFIG,
          icon: 'sentiment_very_satisfied',
          label: $localize`:@@configurationSidebarMenuPolitenessconfig:Politeness`,
        },
        {
          kind: Kind.CRAWLHOSTGROUPCONFIG,
          icon: 'group_work',
          label: $localize`:@@configurationSidebarMenuCrawlhostgroupconfig:Crawl host group`,
        },
      ],
    },
    {
      label: $localize`:@@configurationGroupAccessManagement:Access management`,
      destinations: [
        {
          kind: Kind.ROLEMAPPING,
          icon: 'people',
          label: $localize`:@@configurationSidebarMenuRolemapping:Users`,
        },
      ],
    },
  ];
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  constructor() {

    super();

    this.can = this.abilityService.can;
  }

  visibleDestinations(group: ConfigDestinationGroup): readonly ConfigDestination[] {
    return group.destinations.filter(destination => this.can('read', Kind[destination.kind]));
  }
}
