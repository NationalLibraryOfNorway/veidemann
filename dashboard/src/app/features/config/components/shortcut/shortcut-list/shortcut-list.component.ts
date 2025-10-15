import {Component, Input} from '@angular/core';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {AbilityService} from "@casl/angular";
import {PureAbility} from '@casl/ability';
import {Observable} from 'rxjs';
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
  standalone: true
})
export class ShortcutListComponent {
  readonly Kind = Kind;
  readonly ability$: Observable<PureAbility>;

  @Input()
  configObject: ConfigObject;

  constructor(private abilityService: AbilityService<any>) {
    this.ability$ = this.abilityService.ability$;
  }
}
