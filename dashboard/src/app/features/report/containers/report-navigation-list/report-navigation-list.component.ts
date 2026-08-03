import {Component, ChangeDetectionStrategy} from '@angular/core';
import {AuthService} from '../../../../core';
import {NavigationListComponent} from '../../../../shared/components';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {MatIcon} from '@angular/material/icon';
import {RouterLink} from '@angular/router';
import {MatCardModule} from '@angular/material/card';


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
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  constructor(protected override authService: AuthService,
              private abilityService: AbilityServiceSignal<MongoAbility>) {
    super(authService);
    this.can = this.abilityService.can;
  }
}
