import { Component, Input, ChangeDetectionStrategy, inject } from '@angular/core';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {AbilityServiceSignal} from "@casl/angular";
import {MongoAbility} from '@casl/ability';
import {AsyncPipe, NgClass} from '@angular/common';
import {MatListModule} from '@angular/material/list';
import {RouterLink} from '@angular/router';
import {
  BrowserConfigNamePipe,
  BrowserScriptNamePipe,
  CollectionNamePipe,
  CrawlConfigNamePipe,
  CrawlJobDisabledStatusPipe,
  CrawlScheduleNamePipe,
  EntityNamePipe,
  PolitenessConfigNamePipe
} from '../../../pipe';
import {JobNamePipe} from '../../../../report/pipe';
import {MatIcon} from '@angular/material/icon';

@Component({
  selector: 'app-shortcut-list',
  templateUrl: './shortcut-list.component.html',
  imports: [
    AsyncPipe,
    BrowserConfigNamePipe,
    BrowserScriptNamePipe,
    CollectionNamePipe,
    CrawlConfigNamePipe,
    CrawlJobDisabledStatusPipe,
    CrawlScheduleNamePipe,
    EntityNamePipe,
    JobNamePipe,
    MatIcon,
    MatListModule,
    NgClass,
    PolitenessConfigNamePipe,
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
