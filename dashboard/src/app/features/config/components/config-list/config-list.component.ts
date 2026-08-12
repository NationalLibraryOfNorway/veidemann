import {ChangeDetectionStrategy, Component, EventEmitter, HostListener, Input, Output} from '@angular/core';
import {SortDirection} from '@angular/material/sort';
import {ConfigObject, Kind, Label, Role} from '../../../../shared/models/config';
import {CONFIG_LIST_IMPORTS, ConfigListBaseComponent} from './config-list-base';
import {isEmojiLabel, LabelDisplayComponent} from '../../../../shared/components';
import {configKindIcon} from '../../func/config-kind-icon';

@Component({
    selector: 'app-config-list',
    templateUrl: './config-list.component.html',
    styleUrls: [
        './config-list-base.scss',
        './config-list.component.scss',
    ],
    changeDetection: ChangeDetectionStrategy.OnPush,
    imports: [
      ...CONFIG_LIST_IMPORTS,
      LabelDisplayComponent,
    ],
    standalone: true,
})

export class ConfigListComponent extends ConfigListBaseComponent<ConfigObject> {
  @Input() configKind: Kind | null = null;
  @Input() titleOnly = false;
  @Input() showKindIcon = true;
  @Input() flush = false;
  @Input() primaryLinkDestination: 'detail' | 'title' = 'detail';
  @Input() showOrderControl = false;
  @Input() showStateFilter = false;
  @Input() disabledFilter: boolean | null = null;
  @Output() readonly labelClick = new EventEmitter<Label>();
  @Output() readonly disabledFilterChange = new EventEmitter<boolean | null>();
  readonly deactivatedAriaDescription = $localize`:@@configurationListDeactivatedStatus:Deactivated`;
  protected readonly isEmojiLabel = isEmojiLabel;
  protected override readonly autoSelectAppendedRows = true;
  readonly orderOptions = [
    {value: '', label: $localize`:@@configurationListDefaultOrder:Default order`},
    {value: 'name:asc', label: $localize`:@@configurationListNameAscending:Name: A–Z`},
    {value: 'name:desc', label: $localize`:@@configurationListNameDescending:Name: Z–A`},
    {
      value: 'lastModified:desc',
      label: $localize`:@@configurationListModifiedNewest:Last modified: newest first`,
    },
    {
      value: 'lastModified:asc',
      label: $localize`:@@configurationListModifiedOldest:Last modified: oldest first`,
    },
  ];

  isSelectionMode(): boolean {
    return this.multiSelect && this.selectedRows().length > 0;
  }

  get orderValue(): string {
    return this.sortActive && this.sortDirection ? `${this.sortActive}:${this.sortDirection}` : '';
  }

  get orderLabel(): string {
    return this.orderOptions.find(option => option.value === this.orderValue)?.label
      ?? this.orderOptions[0].label;
  }

  onDisabledFilterChange(value: boolean | null | undefined): void {
    this.disabledFilterChange.emit(value ?? null);
  }

  onOrderChange(value: string): void {
    const [active = '', direction = ''] = value.split(':');
    this.sort.emit({active, direction: direction as SortDirection});
  }

  isDeactivated(config: ConfigObject): boolean {
    switch (config.kind) {
      case this.Kind.SEED:
        return !!config.seed?.disabled;
      case this.Kind.CRAWLJOB:
        return !!config.crawlJob?.disabled;
      default:
        return false;
    }
  }

  configKindIcon(config: ConfigObject): string {
    return configKindIcon(config.kind);
  }

  selectionAriaLabel(config: ConfigObject): string {
    if (this.isChecked(config)) {
      return $localize`:@@configurationListDeselectConfigurationAriaLabel:Deselect ${this.configTitle(config)}:CONFIGURATION_NAME:`;
    }
    return $localize`:@@configurationListSelectConfigurationAriaLabel:Select ${this.configTitle(config)}:CONFIGURATION_NAME:`;
  }

  configTitle(config: ConfigObject): string {
    if (config.kind === this.Kind.ROLEMAPPING) {
      return config.roleMapping?.email || config.roleMapping?.group || config.meta.name;
    }
    return config.meta.name;
  }

  override detailHref(config: ConfigObject): string {
    return this.primaryLinkDestination === 'title'
      ? this.configTitle(config)
      : super.detailHref(config);
  }

  configSubtitle(config: ConfigObject): string {
    if (config.kind === this.Kind.ROLEMAPPING) {
      return config.roleMapping?.roleList.map(role => Role[role]).join(', ') || '';
    }
    return config.meta.description;
  }

  onSelectionStart(config: ConfigObject, event: Event): void {
    event.preventDefault();
    event.stopPropagation();
    this.onCheckboxToggle(config);
  }

  override onRowClick(config: ConfigObject, event?: Event): void {
    if (this.isSelectionMode()) {
      this.onCheckboxToggle(config);
      return;
    }
    super.onRowClick(config, event);
  }

  override onRowKeydown(config: ConfigObject, event: KeyboardEvent): void {
    const nestedAction = event.target instanceof Element
      && event.target !== event.currentTarget
      && !!event.target.closest('a, button, input, [role="button"]');
    if (nestedAction) {
      return;
    }
    if (this.isSelectionMode() && (event.key === 'Enter' || event.key === ' ')) {
      event.preventDefault();
      this.onCheckboxToggle(config);
      return;
    }
    super.onRowKeydown(config, event);
  }

  @HostListener('keydown.escape', ['$event'])
  onSelectionEscape(event: Event): void {
    if (!this.isSelectionMode()) {
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    this.onDeselectAll();
  }

  labelSearchAriaLabel(label: Label): string {
    const labelText = `${label.key}:${label.value}`;
    return $localize`:@@configurationListLabelSearchAriaLabel:Search for exact label ${labelText}:LABEL:`;
  }

  onLabelClick(label: Label, event: Event): void {
    event.preventDefault();
    event.stopPropagation();
    this.labelClick.emit(label);
  }

  configTypePluralLabel(): string {
    switch (this.configKind ?? this.selectedRows()[0]?.kind) {
      case this.Kind.CRAWLENTITY:
        return $localize`:@@configurationListTypeEntities:entities`;
      case this.Kind.SEED:
        return $localize`:@@configurationListTypeSeeds:seeds`;
      case this.Kind.CRAWLJOB:
        return $localize`:@@configurationListTypeCrawlJobs:crawl jobs`;
      case this.Kind.CRAWLSCHEDULECONFIG:
        return $localize`:@@configurationListTypeSchedules:schedules`;
      case this.Kind.CRAWLCONFIG:
        return $localize`:@@configurationListTypeCrawlConfigurations:crawl configurations`;
      case this.Kind.COLLECTION:
        return $localize`:@@configurationListTypeCollections:collections`;
      case this.Kind.BROWSERCONFIG:
        return $localize`:@@configurationListTypeBrowserConfigurations:browser configurations`;
      case this.Kind.BROWSERSCRIPT:
        return $localize`:@@configurationListTypeBrowserScripts:browser scripts`;
      case this.Kind.POLITENESSCONFIG:
        return $localize`:@@configurationListTypePolitenessConfigurations:politeness configurations`;
      case this.Kind.CRAWLHOSTGROUPCONFIG:
        return $localize`:@@configurationListTypeCrawlHostGroups:crawl host groups`;
      case this.Kind.ROLEMAPPING:
        return $localize`:@@configurationListTypeRoleMappings:role mappings`;
      default:
        return $localize`:@@configurationListTypeConfigurations:configurations`;
    }
  }
}
