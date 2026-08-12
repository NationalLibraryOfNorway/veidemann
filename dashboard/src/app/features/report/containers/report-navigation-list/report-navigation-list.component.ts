import { ChangeDetectionStrategy,Component,inject } from '@angular/core';
import { MatCardModule } from '@angular/material/card';
import { MatIcon } from '@angular/material/icon';
import { RouterLink } from '@angular/router';
import { MongoAbility } from '@casl/ability';
import { AbilityServiceSignal } from '@casl/angular';
import { NavigationListComponent } from '../../../../shared/components';

interface ReportDestination {
  readonly route: string;
  readonly icon: string;
  readonly label: string;
  readonly permissionSubject: string;
}

interface ReportDestinationGroup {
  readonly label: string;
  readonly destinations: readonly ReportDestination[];
}

@Component({
  selector: 'app-report-navigation-list',
  templateUrl: './report-navigation-list.component.html',
  styleUrls: ['../../../../shared/components/navigation-grid/navigation-grid.scss'],
  imports: [
    MatIcon,
    MatCardModule,
    RouterLink,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class ReportNavigationListComponent extends NavigationListComponent {
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  readonly destinationGroups: readonly ReportDestinationGroup[] = [
    {
      label: $localize`:@@reportGroupExecutions:Executions`,
      destinations: [
        {
          route: 'jobexecution',
          icon: 'hdr_strong',
          label: $localize`:@@reportNavigationLinkJobExecution:Job execution`,
          permissionSubject: 'jobexecution',
        },
        {
          route: 'crawlexecution',
          icon: 'hdr_weak',
          label: $localize`:@@reportNavigationLinkCrawlExecution:Crawl execution`,
          permissionSubject: 'crawlexecution',
        },
      ],
    },
    {
      label: $localize`:@@reportGroupLogs:Logs`,
      destinations: [
        {
          route: 'pagelog',
          icon: 'art_track',
          label: $localize`:@@reportNavigationLinkPageLog:Page log`,
          permissionSubject: 'pagelog',
        },
        {
          route: 'crawllog',
          icon: 'event_note',
          label: $localize`:@@reportNavigationLinkCrawlLog:Crawl log`,
          permissionSubject: 'crawllog',
        },
      ],
    },
  ];
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  constructor() {

    super();

    this.can = this.abilityService.can;
  }

  visibleDestinations(group: ReportDestinationGroup): readonly ReportDestination[] {
    return group.destinations.filter(destination => this.can('read', destination.permissionSubject));
  }
}
