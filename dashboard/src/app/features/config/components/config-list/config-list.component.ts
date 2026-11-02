import {
  ChangeDetectionStrategy,
  Component,
  effect,
  ElementRef,
  EventEmitter,
  HostListener,
  Input,
  OnChanges,
  Output,
  signal,
  SimpleChanges,
  ViewChild,
} from '@angular/core';
import {MatPaginatorModule, PageEvent} from '@angular/material/paginator';
import {SortDirection} from '@angular/material/sort';
import {ConfigObject, Kind, Label, Role} from '../../../../shared/models/config';
import {CONFIG_LIST_IMPORTS, ConfigListBaseComponent} from './config-list-base';
import {isEmojiLabel, LabelDisplayComponent} from '../../../../shared/components';
import {configKindIcon} from '../../func/config-kind-icon';

type RowDecoration = 'labels' | 'crawlJobs';
type PagingMode = 'scroll' | 'pages';

const PAGING_MODE_KEY = 'configList.pagingMode';
const PAGE_SIZE_KEY = 'configList.pageSize';
const PAGE_SIZE_OPTIONS = [25, 50, 100];

function readStored(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function writeStored(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
  } catch {
    // The choice then only lasts for this session.
  }
}

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
      MatPaginatorModule,
    ],
    standalone: true,
})

export class ConfigListComponent extends ConfigListBaseComponent<ConfigObject> implements OnChanges {
  @Input() configKind: Kind | null = null;
  @Input() titleOnly = false;
  @Input() flush = false;
  @Input() primaryLinkDestination: 'detail' | 'title' = 'detail';
  @Input() showOrderControl = false;
  @Input() showStateFilter = false;
  @Input() disabledFilter: boolean | null = null;
  @Input() showDecorationToggle = false;
  /** Offers fixed pages as an alternative to endless scrolling. */
  @Input() allowPaging = false;
  @Input() crawlJobs: ConfigObject[] = [];
  @Output() readonly labelClick = new EventEmitter<Label>();
  @Output() readonly crawlJobClick = new EventEmitter<string>();
  @Output() readonly disabledFilterChange = new EventEmitter<boolean | null>();
  readonly deactivatedAriaDescription = $localize`:@@configurationListDeactivatedStatus:Deactivated`;
  readonly sortByNameActionDescription = $localize`:@@configurationListSortByNameAriaLabel:Sort by name`;
  protected readonly isEmojiLabel = isEmojiLabel;
  readonly crawlJobIcon = configKindIcon(Kind.CRAWLJOB);
  readonly rowDecorations = signal<RowDecoration[]>(['labels']);
  readonly pageSizeOptions = PAGE_SIZE_OPTIONS;
  readonly pagingMode = signal<PagingMode>(readStored(PAGING_MODE_KEY) === 'pages' ? 'pages' : 'scroll');
  /** The last count received, so the paginator keeps its range while the total is recounted. */
  readonly knownTotal = signal(0);
  private pageSize = PAGE_SIZE_OPTIONS.find(size => String(size) === readStored(PAGE_SIZE_KEY)) ?? PAGE_SIZE_OPTIONS[1];

  @ViewChild('scrollContainer') private scrollContainer?: ElementRef<HTMLElement>;
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

  constructor() {
    super();
    effect(() => {
      const total = this.totalLength();
      if (total !== null) {
        this.knownTotal.set(total);
      }
    });
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['dataSource'] || changes['allowPaging']) {
      this.applyPaging();
    }
  }

  setPagingMode(mode: PagingMode): void {
    this.pagingMode.set(mode);
    writeStored(PAGING_MODE_KEY, mode);
    this.applyPaging();
  }

  onPage(event: PageEvent): void {
    this.expandedId.set(null);
    if (event.pageSize !== this.pageSize) {
      this.pageSize = event.pageSize;
      writeStored(PAGE_SIZE_KEY, String(event.pageSize));
      this.dataSource.setPaging(event.pageSize);
    } else {
      this.dataSource.goToPage(event.pageIndex);
    }
    if (this.scrollContainer) {
      this.scrollContainer.nativeElement.scrollTop = 0;
    }
  }

  private applyPaging(): void {
    const pageSize = this.allowPaging && this.pagingMode() === 'pages' ? this.pageSize : null;
    if (this.dataSource && (this.dataSource.currentPaging?.pageSize ?? null) !== pageSize) {
      this.dataSource.setPaging(pageSize);
    }
  }

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

  toggleDisabledFilter(value: boolean): void {
    if (this.isSelectionMode()) {
      return;
    }
    this.disabledFilterChange.emit(this.disabledFilter === value ? null : value);
  }

  onOrderChange(value: string): void {
    if (this.isSelectionMode()) {
      return;
    }
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

  previewAriaLabel(config: ConfigObject): string {
    return $localize`:@@configurationListPreviewAriaLabel:Preview of ${this.configTitle(config)}:CONFIGURATION_NAME:`;
  }

  expandAriaLabel(config: ConfigObject): string {
    if (this.expandedId() === config.id) {
      return $localize`:@@configurationListHidePreviewAriaLabel:Hide preview of ${this.configTitle(config)}:CONFIGURATION_NAME:`;
    }
    return $localize`:@@configurationListShowPreviewAriaLabel:Show preview of ${this.configTitle(config)}:CONFIGURATION_NAME:`;
  }

  configSubtitle(config: ConfigObject): string {
    if (config.kind === this.Kind.ROLEMAPPING) {
      return config.roleMapping?.roleList.map(role => Role[role]).join(', ') || '';
    }
    return config.meta.description;
  }

  // Rows are only selected through their checkbox. Without a preview a row click
  // would navigate away, which is not wanted in the middle of a selection.
  override onRowClick(config: ConfigObject, event?: Event): void {
    if (this.isSelectionMode() && !this.expandable) {
      return;
    }
    super.onRowClick(config, event);
  }

  // Links that open in a new tab keep working in selection mode. Links that navigate in place
  // would drop the selection, so a plain click then acts like a click on the row instead.
  override onPrimaryLink(config: ConfigObject, event: Event): void {
    if (this.isSelectionMode() && this.detailLinkTarget !== '_blank') {
      event.preventDefault();
      this.onRowClick(config);
      return;
    }
    super.onPrimaryLink(config, event);
  }

  override onRowKeydown(config: ConfigObject, event: KeyboardEvent): void {
    if (this.isSelectionMode() && !this.expandable) {
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

  toggleDecoration(decoration: RowDecoration): void {
    this.rowDecorations.update(decorations => decorations.includes(decoration)
      ? decorations.filter(value => value !== decoration)
      : [...decorations, decoration]);
  }

  showLabels(): boolean {
    return !this.showDecorationToggle || this.rowDecorations().includes('labels');
  }

  showCrawlJobs(): boolean {
    return this.showDecorationToggle && this.rowDecorations().includes('crawlJobs');
  }

  hasDecoration(config: ConfigObject): boolean {
    return this.showLabels() && config.meta.labelList.length > 0
      || this.showCrawlJobs() && !!config.seed?.jobRefList.length;
  }

  decorationAriaLabel(): string {
    if (this.showLabels() && this.showCrawlJobs()) {
      return $localize`:@@configurationListLabelsAndCrawlJobsAriaLabel:Configuration labels and crawl jobs`;
    }
    if (this.showCrawlJobs()) {
      return $localize`:@@configurationListCrawlJobsAriaLabel:Crawl jobs`;
    }
    return $localize`:@@configurationListLabelsAriaLabel:Configuration labels`;
  }

  crawlJobName(jobId: string): string {
    return this.crawlJobs?.find(job => job.id === jobId)?.meta.name?.trim() || jobId;
  }

  crawlJobFilterAriaLabel(jobId: string): string {
    return $localize`:@@configurationListCrawlJobFilterAriaLabel:Filter by crawl job ${this.crawlJobName(jobId)}:CRAWL_JOB_NAME:`;
  }

  onCrawlJobClick(jobId: string, event: Event): void {
    event.preventDefault();
    event.stopPropagation();
    this.crawlJobClick.emit(jobId);
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
