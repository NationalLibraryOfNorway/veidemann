import { ChangeDetectionStrategy,Component,inject } from '@angular/core';
import { MatCardModule } from '@angular/material/card';
import { MatIcon } from '@angular/material/icon';
import { RouterLink } from '@angular/router';
import { MongoAbility } from '@casl/ability';
import { AbilityServiceSignal } from '@casl/angular';
import { NavigationListComponent } from '../../../../shared/components';
import { Kind } from '../../../../shared/models';
import {ConfigPath} from '../../func';
import {configKindIcon} from '../../func/config-kind-icon';

interface ConfigDestination {
  readonly kind: Kind;
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
  readonly configKindIcon = configKindIcon;
  readonly destinationGroups: readonly ConfigDestinationGroup[] = [
    {
      label: $localize`:@@configurationGroupCrawlTargets:Crawl targets`,
      destinations: [
        {kind: Kind.CRAWLENTITY, label: $localize`:@@configurationSidebarMenuEntity:Entity`},
        {kind: Kind.SEED, label: $localize`:@@configurationSidebarMenuSeed:Seed`},
      ],
    },
    {
      label: $localize`:@@configurationGroupCrawlSetup:Crawl setup`,
      destinations: [
        {kind: Kind.CRAWLJOB, label: $localize`:@@configurationSidebarMenuCrawljobs:Crawl jobs`},
        {
          kind: Kind.CRAWLSCHEDULECONFIG,
          label: $localize`:@@configurationSidebarMenuSchedule:Schedule`,
        },
        {
          kind: Kind.CRAWLCONFIG,
          label: $localize`:@@configurationSidebarMenuCrawlconfig:Crawl config`,
        },
        {
          kind: Kind.COLLECTION,
          label: $localize`:@@configurationSidebarMenuCollection:Collection`,
        },
        {
          kind: Kind.BROWSERCONFIG,
          label: $localize`:@@configurationSidebarMenuBrowserconfig:Browser config`,
        },
        {
          kind: Kind.BROWSERSCRIPT,
          label: $localize`:@@configurationSidebarMenuBrowserscript:Browser script`,
        },
        {
          kind: Kind.POLITENESSCONFIG,
          label: $localize`:@@configurationSidebarMenuPolitenessconfig:Politeness`,
        },
        {
          kind: Kind.CRAWLHOSTGROUPCONFIG,
          label: $localize`:@@configurationSidebarMenuCrawlhostgroupconfig:Crawl host group`,
        },
      ],
    },
    {
      label: $localize`:@@configurationGroupAccessManagement:Access management`,
      destinations: [
        {
          kind: Kind.ROLEMAPPING,
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
