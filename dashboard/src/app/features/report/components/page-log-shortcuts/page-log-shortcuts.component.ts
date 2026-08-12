import { Component, Input, ChangeDetectionStrategy, inject } from '@angular/core';
import {PageLog} from '../../../../shared/models';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import { MatIconModule } from '@angular/material/icon';
import { RouterModule } from '@angular/router';
import {MatChipsModule} from '@angular/material/chips';
import {MatTooltipModule} from '@angular/material/tooltip';
import {MatMenuModule} from '@angular/material/menu';
import {CopyIdDirective} from '../../../../shared/directives';

@Component({
    selector: 'app-page-log-shortcuts',
    templateUrl: './page-log-shortcuts.component.html',
    styleUrls: ['../execution-shortcut-helpers.scss'],
    standalone: true,
    changeDetection: ChangeDetectionStrategy.OnPush,
    imports: [
      CopyIdDirective,
      MatChipsModule,
      MatIconModule,
      MatMenuModule,
      MatTooltipModule,
      RouterModule,
    ]
})
export class PageLogShortcutsComponent {
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  @Input() pageLog: PageLog;
  @Input() presentation: 'chips' | 'menu' = 'chips';

  constructor() {
    this.can = this.abilityService.can;
  }

  get referrerHref(): string | null {
    try {
      const url = new URL(this.pageLog.referrer);
      return url.protocol === 'http:' || url.protocol === 'https:' ? url.href : null;
    } catch {
      return null;
    }
  }
}
