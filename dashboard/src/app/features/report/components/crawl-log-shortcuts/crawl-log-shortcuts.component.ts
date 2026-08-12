import { Component, Input, ChangeDetectionStrategy, inject } from '@angular/core';
import {CrawlLog} from '../../../../shared/models';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {RouterLink} from '@angular/router';
import {MatIcon} from '@angular/material/icon';
import {MatChipsModule} from '@angular/material/chips';
import {MatTooltipModule} from '@angular/material/tooltip';
import {MatMenuModule} from '@angular/material/menu';
import {CopyIdDirective} from '../../../../shared/directives';

@Component({
  selector: 'app-crawl-log-shortcuts',
  templateUrl: './crawl-log-shortcuts.component.html',
  styleUrls: ['../execution-shortcut-helpers.scss'],
  imports: [
    CopyIdDirective,
    RouterLink,
    MatIcon,
    MatChipsModule,
    MatMenuModule,
    MatTooltipModule,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class CrawlLogShortcutsComponent {
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];
  @Input() crawlLog: CrawlLog;
  @Input() presentation: 'chips' | 'menu' = 'chips';

  constructor() {
    this.can = this.abilityService.can;
  }
}
