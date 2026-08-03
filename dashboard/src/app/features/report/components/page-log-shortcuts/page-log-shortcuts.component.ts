import {Component, Input, ChangeDetectionStrategy} from '@angular/core';
import {PageLog} from '../../../../shared/models';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import { MatIconModule } from '@angular/material/icon';
import { RouterModule } from '@angular/router';
import {MatButtonModule} from '@angular/material/button';
import {MatMenuModule} from '@angular/material/menu';

@Component({
    selector: 'app-page-log-shortcuts',
    templateUrl: './page-log-shortcuts.component.html',
    styleUrls: ['../shortcut-actions.scss'],
    standalone: true,
    changeDetection: ChangeDetectionStrategy.OnPush,
    imports: [
      MatButtonModule,
      MatMenuModule,
      MatIconModule,
      RouterModule,
    ]
})
export class PageLogShortcutsComponent {
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  @Input() pageLog: PageLog;

  constructor(private abilityService: AbilityServiceSignal<MongoAbility>) {
    this.can = this.abilityService.can;
  }
}
