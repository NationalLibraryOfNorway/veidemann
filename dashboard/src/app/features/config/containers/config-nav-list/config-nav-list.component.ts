import { ChangeDetectionStrategy,Component,inject } from '@angular/core';
import { MatCardModule } from '@angular/material/card';
import { MatIcon } from '@angular/material/icon';
import { RouterLink } from '@angular/router';
import { MongoAbility } from '@casl/ability';
import { AbilityServiceSignal } from '@casl/angular';
import { NavigationListComponent } from '../../../../shared/components';
import { Kind } from '../../../../shared/models';
import { ConfigPath } from '../../func';

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
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  constructor() {

    super();

    this.can = this.abilityService.can;
  }
}
