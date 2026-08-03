import {Component, ChangeDetectionStrategy} from '@angular/core';
import {AuthService} from '../../../../core';
import {NavigationListComponent} from '../../../../shared/components';
import {AbilityServiceSignal} from '@casl/angular';
import {MatIcon} from '@angular/material/icon';
import { MatListModule, MatNavList} from '@angular/material/list';
import {RouterLink, RouterLinkActive} from '@angular/router';
import {MatLineModule} from '@angular/material/core';


@Component({
  selector: 'app-report-navigation-list',
  templateUrl: './report-navigation-list.component.html',
  styleUrls: ['../../../../shared/components/navigation-list/navigation-list.component.scss'],
  imports: [
    MatIcon,
    MatLineModule,
    MatListModule,
    MatNavList,
    RouterLink,
    RouterLinkActive,
  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: true
})
export class ReportNavigationListComponent extends NavigationListComponent {
  protected readonly can: AbilityServiceSignal<any>['can'];

  constructor(protected override authService: AuthService,
              private abilityService: AbilityServiceSignal<any>) {
    super(authService);
    this.can = this.abilityService.can;
  }
}
