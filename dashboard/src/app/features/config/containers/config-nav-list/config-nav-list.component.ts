import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {ConfigOptions, ConfigPath} from '../../func';
import {Kind} from '../../../../shared/models';
import {AuthService} from '../../../../core';
import {NavigationListComponent} from '../../../../shared/components';
import {Router, RouterLink, RouterLinkActive} from '@angular/router';
import {Observable} from 'rxjs';
import {AbilityService} from '@casl/angular';
import {AsyncPipe} from '@angular/common';
import {MatListModule} from '@angular/material/list';
import {MatIcon} from '@angular/material/icon';
import {MatLineModule} from '@angular/material/core';

@Component({
  selector: 'app-config-navigation-list',
  templateUrl: './config-nav-list.component.html',
  styleUrls: ['../../../../shared/components/navigation-list/navigation-list.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    AsyncPipe,
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
  readonly ability$: Observable<any>

  @Input()
  kind: Kind;

  @Input()
  options: ConfigOptions;

  constructor(protected override authService: AuthService, private router: Router, private abilityService: AbilityService<any>) {
    super(authService);
    this.ability$ = this.abilityService.ability$;
  }
}
