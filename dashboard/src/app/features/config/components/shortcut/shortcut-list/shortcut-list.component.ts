import {Component, Input, ChangeDetectionStrategy} from '@angular/core';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {AbilityServiceSignal} from "@casl/angular";
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
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: true
})
export class ShortcutListComponent {
  readonly Kind = Kind;
  protected readonly can: AbilityServiceSignal<any>['can'];

  @Input()
  configObject: ConfigObject;

  constructor(private abilityService: AbilityServiceSignal<any>) {
    this.can = this.abilityService.can;
  }
}
