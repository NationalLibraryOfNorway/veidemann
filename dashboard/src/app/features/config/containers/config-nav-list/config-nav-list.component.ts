import {ChangeDetectionStrategy, Component} from '@angular/core';
import {ConfigPath} from '../../func';
import {Kind} from '../../../../shared/models';
import {AuthService} from '../../../../core';
import {NavigationListComponent} from '../../../../shared/components';
import {RouterLink} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {MatIcon} from '@angular/material/icon';
import {MatCardModule} from '@angular/material/card';

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
  readonly ConfigPath = ConfigPath;
  readonly Kind = Kind;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  constructor(protected override authService: AuthService, private abilityService: AbilityServiceSignal<MongoAbility>) {
    super(authService);
    this.can = this.abilityService.can;
  }
}
