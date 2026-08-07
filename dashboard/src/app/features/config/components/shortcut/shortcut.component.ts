import { Component, EventEmitter, Input, Output, ChangeDetectionStrategy, inject } from '@angular/core';
import {ConfigObject, Kind} from '../../../../shared/models/config';
import {Params, RouterLink} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {AsyncPipe, NgClass} from '@angular/common';
import {MatChipsModule} from '@angular/material/chips';
import {MatIcon} from '@angular/material/icon';
import {MatTooltipModule} from '@angular/material/tooltip';
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
import {CopyIdDirective} from '../../../../shared/directives';

type ShortcutAction = 'clone' | 'createSeed' | 'openScriptEditor' | 'runCrawl';

@Component({
  selector: 'app-config-shortcut-helpers',
  templateUrl: './shortcut.component.html',
  styleUrls: ['./shortcut.component.scss'],
  imports: [
    AsyncPipe,
    BrowserConfigNamePipe,
    BrowserScriptNamePipe,
    CollectionNamePipe,
    CopyIdDirective,
    CrawlConfigNamePipe,
    CrawlJobDisabledStatusPipe,
    CrawlScheduleNamePipe,
    EntityNamePipe,
    JobNamePipe,
    MatChipsModule,
    MatIcon,
    MatTooltipModule,
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

  @Input() showMainId = true;

  @Input() showReferenceKindLabels = false;

  @Input()
  showScriptEditorAction = false;

  @Output()
  createSeed = new EventEmitter<ConfigObject>();

  @Output()
  runCrawl = new EventEmitter<ConfigObject>();

  @Output()
  clone = new EventEmitter<ConfigObject>();

  @Output()
  openScriptEditor = new EventEmitter<void>();

  constructor() {
    this.can = this.abilityService.can;
  }

  get hasActions(): boolean {
    return this.showScriptEditorAction
      || this.can('create', Kind[this.configObject.kind])
      || (this.configObject.kind === Kind.CRAWLENTITY && this.can('create', Kind[Kind.SEED]))
      || (this.configObject.kind === Kind.SEED && this.can('runCrawl', Kind[Kind.SEED]))
      || (this.configObject.kind === Kind.CRAWLJOB && this.can('runCrawl', Kind[Kind.CRAWLJOB]));
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

  onOpenScriptEditor() {
    this.openScriptEditor.emit();
  }

  onActionKeydown(event: KeyboardEvent, action: ShortcutAction): void {
    if (event.key !== 'Enter' && event.key !== ' ') {
      return;
    }

    event.preventDefault();
    switch (action) {
      case 'clone':
        this.onClone();
        break;
      case 'createSeed':
        this.onCreateSeed();
        break;
      case 'runCrawl':
        this.onRunCrawl();
        break;
      case 'openScriptEditor':
        this.onOpenScriptEditor();
        break;
    }
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
