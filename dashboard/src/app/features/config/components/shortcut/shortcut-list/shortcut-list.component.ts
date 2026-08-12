import { Component, Input, ChangeDetectionStrategy, inject } from '@angular/core';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {AbilityServiceSignal} from "@casl/angular";
import {MongoAbility} from '@casl/ability';
import {AsyncPipe, NgClass} from '@angular/common';
import {MatListModule} from '@angular/material/list';
import {RouterLink} from '@angular/router';
import {
  ConfigRefNamePipe,
  CrawlJobDisabledStatusPipe,
} from '../../../pipe';
import {MatIcon} from '@angular/material/icon';

@Component({
  selector: 'app-shortcut-list',
  templateUrl: './shortcut-list.component.html',
  styleUrls: ['./shortcut-list.component.scss'],
  imports: [
    AsyncPipe,
    ConfigRefNamePipe,
    CrawlJobDisabledStatusPipe,
    MatIcon,
    MatListModule,
    NgClass,
    RouterLink
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class ShortcutListComponent {
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  readonly Kind = Kind;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  @Input()
  configObject: ConfigObject;

  constructor() {
    this.can = this.abilityService.can;
  }
}
