import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {ConfigOptions, ConfigPath} from '../../func';
import {Kind} from '../../../../shared/models';
import {AuthService} from '../../../../core';
import {NavigationListComponent} from '../../../../shared/components';
import {Router, RouterLink, RouterLinkActive} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {MatListModule} from '@angular/material/list';
import {MatIcon} from '@angular/material/icon';
import {MatLineModule} from '@angular/material/core';

@Component({
  selector: 'app-config-navigation-list',
  templateUrl: './config-nav-list.component.html',
  styleUrls: ['../../../../shared/components/navigation-list/navigation-list.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatIcon,
    MatLineModule,
    MatListModule,
    RouterLink,
    RouterLinkActive,

  ],
  standalone: true
})
export class ConfigNavListComponent extends NavigationListComponent {
  readonly ConfigPath = ConfigPath;
  readonly Kind = Kind;
  protected readonly can: AbilityServiceSignal<any>['can'];

  @Input()
  kind: Kind;

  @Input()
  options: ConfigOptions;

  constructor(protected override authService: AuthService, private router: Router, private abilityService: AbilityServiceSignal<any>) {
    super(authService);
    this.can = this.abilityService.can;
  }
}
