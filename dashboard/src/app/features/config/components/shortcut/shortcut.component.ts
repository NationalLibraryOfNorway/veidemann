import { Component, EventEmitter, Input, Output, ChangeDetectionStrategy, inject } from '@angular/core';
import {ConfigObject, Kind} from '../../../../shared/models/config';
import {Params, RouterLink} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {AsyncPipe, NgClass} from '@angular/common';
import {MatChipsModule} from '@angular/material/chips';
import {MatIcon} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
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
  selector: 'app-config-shortcut-helpers',
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
    MatChipsModule,
    MatIcon,
    MatMenuModule,
    NgClass,
    PolitenessConfigNamePipe,
    RouterLink,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class ConfigShortcutHelpersComponent {
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  readonly Kind = Kind;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];
  @Input()
  configObject: ConfigObject;

  @Input() showReferenceKindLabels = false;
  @Input() showClone = true;
  @Input() showDelete = true;
  @Input() showCreateSeed = true;
  @Input() seedContext: ConfigObject | null = null;

  @Output()
  createSeed = new EventEmitter<ConfigObject>();

  @Output()
  runCrawl = new EventEmitter<ConfigObject>();

  @Output()
  runSeedInCrawlJob = new EventEmitter<{seed: ConfigObject; crawlJob: ConfigObject}>();

  @Output()
  clone = new EventEmitter<ConfigObject>();

  @Output()
  delete = new EventEmitter<ConfigObject>();

  constructor() {
    this.can = this.abilityService.can;
  }

  get hasActions(): boolean {
    return this.canCreateSeed
      || this.canRunCrawl
      || (this.showClone && this.can('create', Kind[this.configObject.kind]))
      || (this.showDelete && !!this.configObject.id && this.can('delete', Kind[this.configObject.kind]));
  }

  get canCreateSeed(): boolean {
    return this.showCreateSeed
      && this.configObject.kind === Kind.CRAWLENTITY
      && this.can('create', Kind[Kind.SEED]);
  }

  get canRunCrawl(): boolean {
    if (this.configObject.kind === Kind.SEED) {
      return this.can('runCrawl', Kind[Kind.SEED]);
    }
    if (this.configObject.kind === Kind.CRAWLJOB) {
      return this.seedContext?.kind === Kind.SEED
        ? this.can('runCrawl', Kind[Kind.SEED])
        : this.can('runCrawl', Kind[Kind.CRAWLJOB]);
    }
    return false;
  }

  onClone() {
    this.clone.emit(this.configObject);
  }

  onDelete() {
    this.delete.emit(this.configObject);
  }

  onCreateSeed() {
    this.createSeed.emit(this.configObject);
  }

  onRunCrawl() {
    if (this.configObject.kind === Kind.CRAWLJOB && this.seedContext?.kind === Kind.SEED) {
      this.runSeedInCrawlJob.emit({seed: this.seedContext, crawlJob: this.configObject});
      return;
    }
    this.runCrawl.emit(this.configObject);
  }

  get crawlJobActionLabel(): string {
    return this.seedContext?.kind === Kind.SEED
      ? $localize`:@@configDetailRunSeedWithCrawljobAction:Crawl seed with this crawljob`
      : $localize`:@@configDetailRunCrawljobAction:Run crawljob`;
  }

  getJobRefListQueryParams(configObject: ConfigObject): Params {
    return {crawl_job_id: configObject.seed.jobRefList.map(jobRef => jobRef.id)};
  }

  referenceKindLabel(kind: Kind): string {
    switch (kind) {
      case Kind.CRAWLENTITY: return $localize`:@@configReferenceTypeEntity:Entity`;
      case Kind.SEED: return $localize`:@@configReferenceTypeSeed:Seed`;
      case Kind.CRAWLJOB: return $localize`:@@configReferenceTypeCrawlJob:Crawl job`;
      case Kind.CRAWLCONFIG: return $localize`:@@configReferenceTypeCrawlConfig:Crawl config`;
      case Kind.CRAWLSCHEDULECONFIG: return $localize`:@@configReferenceTypeSchedule:Schedule`;
      case Kind.BROWSERCONFIG: return $localize`:@@configReferenceTypeBrowserConfig:Browser config`;
      case Kind.POLITENESSCONFIG: return $localize`:@@configReferenceTypePolitenessConfig:Politeness config`;
      case Kind.BROWSERSCRIPT: return $localize`:@@configReferenceTypeBrowserScript:Browser script`;
      case Kind.COLLECTION: return $localize`:@@configReferenceTypeCollection:Collection`;
      default: return $localize`:@@configReferenceTypeConfiguration:Configuration`;
    }
  }

}
