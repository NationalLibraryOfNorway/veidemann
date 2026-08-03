import { Component, Input, ChangeDetectionStrategy, inject } from '@angular/core';
import {CrawlLog} from '../../../../shared/models';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {RouterLink} from '@angular/router';
import {MatIcon} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {MatMenuModule} from '@angular/material/menu';

@Component({
  selector: 'app-crawl-log-shortcuts',
  templateUrl: './crawl-log-shortcuts.component.html',
  styleUrls: ['../shortcut-actions.scss'],
  imports: [
    RouterLink,
    MatIcon,
    MatButtonModule,
    MatMenuModule,

  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class CrawlLogShortcutsComponent {
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  @Input() crawlLog: CrawlLog;

  constructor() {
    this.can = this.abilityService.can;
  }
}
