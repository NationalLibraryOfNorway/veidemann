import {Component, EventEmitter, Input, Output, ChangeDetectionStrategy} from '@angular/core';
import {ConfigObject, Kind} from '../../../../shared/models/config';
import {Params, RouterLink} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {AsyncPipe, NgClass} from '@angular/common';
import {MatButtonModule} from '@angular/material/button';
import {MatIcon} from '@angular/material/icon';
import {MatMenuModule} from '@angular/material/menu';
import {
  BrowserConfigNamePipe,
  BrowserScriptNamePipe,
  CollectionNamePipe,
  CrawlConfigNamePipe,
  CrawlJobDisabledStatusPipe,
  CrawlScheduleNamePipe,
  EntityNamePipe,
  PolitenessConfigNamePipe
} from '../../pipe';
import {JobNamePipe} from '../../../report/pipe';

@Component({
  selector: 'app-shortcut',
  templateUrl: './shortcut.component.html',
  styleUrls: ['./shortcut.component.scss'],
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
    MatButtonModule,
    MatIcon,
    MatMenuModule,
    NgClass,
    PolitenessConfigNamePipe,
    RouterLink,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class ShortcutComponent {
  readonly Kind = Kind;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];
  @Input()
  configObject: ConfigObject;

  @Output()
  createSeed = new EventEmitter<ConfigObject>();

  @Output()
  runCrawl = new EventEmitter<ConfigObject>();

  @Output()
  clone = new EventEmitter<ConfigObject>();

  constructor(private abilityService: AbilityServiceSignal<MongoAbility>) {
    this.can = this.abilityService.can;
  }

  onClone() {
    this.clone.emit(this.configObject);
  }

  onCreateSeed() {
    this.createSeed.emit(this.configObject);
  }

  onRunCrawl() {
    this.runCrawl.emit(this.configObject);
  }

  getJobRefListQueryParams(configObject: ConfigObject): Params {
    return {crawl_job_id: configObject.seed.jobRefList.map(jobRef => jobRef.id)};
  }

}
