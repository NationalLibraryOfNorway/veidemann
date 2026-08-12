import {Component, EventEmitter, HostBinding, Input, Output, ChangeDetectionStrategy, inject, signal} from '@angular/core';
import {ConfigObject, Kind} from '../../../../shared/models/config';
import {Params, RouterLink} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {AsyncPipe} from '@angular/common';
import {MatIcon} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {MatMenuModule} from '@angular/material/menu';
import {
  ConfigRefObjectPipe,
} from '../../pipe';
import {configKindIcon} from '../../func/config-kind-icon';

@Component({
  selector: 'app-config-shortcut-helpers',
  templateUrl: './shortcut.component.html',
  styleUrls: ['./shortcut.component.scss'],
  imports: [
    AsyncPipe,
    ConfigRefObjectPipe,
    MatButtonModule,
    MatIcon,
    MatMenuModule,
    RouterLink,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class ConfigShortcutHelpersComponent {
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  readonly Kind = Kind;
  readonly configKindIcon = configKindIcon;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];
  @Input()
  configObject: ConfigObject;

  @Input() showReferences = true;
  @Input() showClone = true;
  @Input() showDelete = true;
  @Input() showEdit = false;
  @Input() showCreateSeed = true;
  @Input() showFilters = true;
  @Input() showActions = true;
  @Input() seedContext: ConfigObject | null = null;

  readonly activeMenuConfig = signal<ConfigObject | null>(null);
  readonly relationshipMenu = signal(false);

  @Output()
  createSeed = new EventEmitter<ConfigObject>();

  @Output()
  runCrawl = new EventEmitter<ConfigObject>();

  @Output()
  runSeedInCrawlJob = new EventEmitter<{seed: ConfigObject; crawlJob: ConfigObject}>();

  @Output()
  clone = new EventEmitter<ConfigObject>();

  @Output()
  edit = new EventEmitter<ConfigObject>();

  @Output()
  delete = new EventEmitter<ConfigObject>();

  constructor() {
    this.can = this.abilityService.can;
  }

  get hasActions(): boolean {
    return this.showActions && (this.showCreateSeed && this.canCreateSeedFor(this.configObject)
      || this.canRunCrawlFor(this.configObject, this.seedContext)
      || (this.showEdit && this.can('update', Kind[this.configObject.kind]))
      || (this.showClone && this.can('create', Kind[this.configObject.kind]))
      || (this.showDelete && !!this.configObject.id && this.can('delete', Kind[this.configObject.kind])));
  }

  get hasMenuItems(): boolean {
    return this.hasFilters || this.hasActions;
  }

  @HostBinding('class.empty-shortcut-layout')
  get empty(): boolean {
    return !this.hasMenuItems && !this.hasReferenceLinks;
  }

  get hasReferenceLinks(): boolean {
    if (!this.showReferences) return false;
    switch (this.configObject.kind) {
      case Kind.SEED:
        return !!this.configObject.seed.entityRef.id && this.can('read', Kind[Kind.CRAWLENTITY])
          || !!this.configObject.seed.jobRefList.length && this.can('read', Kind[Kind.CRAWLJOB]);
      case Kind.CRAWLJOB:
        return !!this.configObject.crawlJob.scheduleRef.id && this.can('read', Kind[Kind.CRAWLSCHEDULECONFIG])
          || !!this.configObject.crawlJob.crawlConfigRef.id && this.can('read', Kind[Kind.CRAWLCONFIG])
          || !!this.configObject.crawlJob.scopeScriptRef.id && this.can('read', Kind[Kind.BROWSERSCRIPT]);
      case Kind.CRAWLCONFIG:
        return !!this.configObject.crawlConfig.collectionRef.id && this.can('read', Kind[Kind.COLLECTION])
          || !!this.configObject.crawlConfig.browserConfigRef.id && this.can('read', Kind[Kind.BROWSERCONFIG])
          || !!this.configObject.crawlConfig.politenessRef.id && this.can('read', Kind[Kind.POLITENESSCONFIG]);
      case Kind.BROWSERCONFIG:
        return !!this.configObject.browserConfig.scriptRefList.length && this.can('read', Kind[Kind.BROWSERSCRIPT]);
      default:
        return false;
    }
  }

  get hasFilters(): boolean {
    return this.showFilters && this.hasFiltersFor(this.configObject);
  }

  hasFiltersFor(configObject: ConfigObject): boolean {
    switch (configObject.kind) {
      case Kind.CRAWLENTITY:
        return this.can('read', Kind[Kind.SEED]);
      case Kind.SEED:
        return this.can('read', 'crawlexecution') || this.can('read', Kind[Kind.SEED]);
      case Kind.CRAWLJOB:
        return this.can('read', 'reports')
          || this.can('read', Kind[Kind.SEED])
          || !!configObject.crawlJob.scheduleRef.id
            && this.can('read', Kind[Kind.CRAWLSCHEDULECONFIG])
          || !!configObject.crawlJob.crawlConfigRef.id
            && this.can('read', Kind[Kind.CRAWLCONFIG]);
      case Kind.CRAWLSCHEDULECONFIG:
        return this.can('read', Kind[Kind.CRAWLJOB]);
      case Kind.CRAWLCONFIG:
        return this.can('read', Kind[Kind.CRAWLJOB])
          || !!configObject.crawlConfig.collectionRef.id
            && this.can('read', Kind[Kind.COLLECTION])
          || !!configObject.crawlConfig.browserConfigRef.id
            && this.can('read', Kind[Kind.BROWSERCONFIG])
          || !!configObject.crawlConfig.politenessRef.id
            && this.can('read', Kind[Kind.POLITENESSCONFIG]);
      case Kind.COLLECTION:
      case Kind.BROWSERCONFIG:
      case Kind.POLITENESSCONFIG:
        return this.can('read', Kind[Kind.CRAWLCONFIG]);
      case Kind.BROWSERSCRIPT:
        return this.can('read', Kind[Kind.BROWSERCONFIG]);
      default:
        return false;
    }
  }

  canCreateSeedFor(configObject: ConfigObject): boolean {
    return configObject.kind === Kind.CRAWLENTITY
      && this.can('create', Kind[Kind.SEED]);
  }

  canRunCrawlFor(configObject: ConfigObject, seedContext: ConfigObject | null): boolean {
    if (configObject.kind === Kind.SEED) {
      return this.can('runCrawl', Kind[Kind.SEED]);
    }
    if (configObject.kind === Kind.CRAWLJOB) {
      return seedContext?.kind === Kind.SEED
        ? this.can('runCrawl', Kind[Kind.SEED])
        : this.can('runCrawl', Kind[Kind.CRAWLJOB]);
    }
    return false;
  }

  openMenu(configObject: ConfigObject, relationship: boolean): void {
    this.activeMenuConfig.set(configObject);
    this.relationshipMenu.set(relationship);
  }

  isDeactivated(configObject: ConfigObject | null): boolean {
    return !!configObject && (configObject.kind === Kind.SEED && configObject.seed.disabled
      || configObject.kind === Kind.CRAWLJOB && configObject.crawlJob.disabled);
  }

  menuSeedContext(): ConfigObject | null {
    return this.relationshipMenu() && this.configObject.kind === Kind.SEED
      ? this.configObject
      : this.seedContext;
  }

  onClone(configObject: ConfigObject) {
    this.clone.emit(configObject);
  }

  onEdit(configObject: ConfigObject) {
    this.edit.emit(configObject);
  }

  onDelete(configObject: ConfigObject) {
    this.delete.emit(configObject);
  }

  onCreateSeed(configObject: ConfigObject) {
    this.createSeed.emit(configObject);
  }

  onRunCrawl(configObject: ConfigObject, seedContext: ConfigObject | null) {
    if (configObject.kind === Kind.CRAWLJOB && seedContext?.kind === Kind.SEED) {
      this.runSeedInCrawlJob.emit({seed: seedContext, crawlJob: configObject});
      return;
    }
    this.runCrawl.emit(configObject);
  }

  crawlJobActionLabel(seedContext: ConfigObject | null): string {
    return seedContext?.kind === Kind.SEED
      ? $localize`:@@configDetailRunSeedWithCrawljobAction:Run crawl with this crawl job`
      : $localize`:@@configDetailRunCrawljobAction:Run crawl`;
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
