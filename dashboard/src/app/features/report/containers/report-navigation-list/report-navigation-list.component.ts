import { ChangeDetectionStrategy,Component,inject } from '@angular/core';
import { MatCardModule } from '@angular/material/card';
import { MatIcon } from '@angular/material/icon';
import { RouterLink } from '@angular/router';
import { MongoAbility } from '@casl/ability';
import { AbilityServiceSignal } from '@casl/angular';
import { NavigationListComponent } from '../../../../shared/components';


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

  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  constructor() {

    super();

    this.can = this.abilityService.can;
  }
}
