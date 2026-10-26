import {SelectionModel} from '@angular/cdk/collections';
import {AsyncPipe, DecimalPipe, NgTemplateOutlet} from '@angular/common';
import {
  ContentChild,
  ContentChildren,
  Directive,
  ElementRef,
  EventEmitter,
  Input,
  OnDestroy,
  Output,
  QueryList,
  signal,
  TemplateRef,
  ViewChild,
} from '@angular/core';
import {MatButtonModule} from '@angular/material/button';
import {MatCheckboxModule} from '@angular/material/checkbox';
import {MatChipsModule} from '@angular/material/chips';
import {MatIconModule} from '@angular/material/icon';
import {MatListModule} from '@angular/material/list';
import {MatMenuModule} from '@angular/material/menu';
import {MatSortModule, Sort, SortDirection} from '@angular/material/sort';
import {MatTooltipModule} from '@angular/material/tooltip';
import {MatToolbarModule} from '@angular/material/toolbar';
import {combineLatest, Subscription} from 'rxjs';
import {startWith} from 'rxjs/operators';

import {Kind, ListDataSource, ListItem} from '../../../../shared/models';
import {
  ActionDirective,
  ExtraDirective,
  FilterDirective,
  PreviewDirective,
  ShortcutDirective,
} from '../../../../shared/directives';

export const CONFIG_LIST_IMPORTS = [
  AsyncPipe,
  DecimalPipe,
  MatButtonModule,
  MatCheckboxModule,
  MatChipsModule,
  MatIconModule,
  MatListModule,
  MatMenuModule,
  MatSortModule,
  MatTooltipModule,
  MatToolbarModule,
  NgTemplateOutlet,
];

@Directive()
export abstract class ConfigListBaseComponent<T extends ListItem> implements OnDestroy {
  readonly Kind = Kind;
  readonly totalLength = signal<number | null>(null);
  readonly loadedLength = signal(0);
  readonly allSelected = signal(false);
  readonly isAllLoadedSelected = signal(false);
  readonly selectedRows = signal<readonly T[]>([]);
  protected readonly autoSelectAppendedRows: boolean = false;

  @Input()
  set length(length: number | null) {
    this.totalLength.set(length);
  }

  @Input() sortDirection: SortDirection = '';
  @Input() sortActive = 'name';
  @Input() displayedColumns: string[] = ['select', 'name', 'description', 'extra', 'action'];
  @Input() multiSelect = true;
  @Input() detailPath = '';
  @Input() detailLinkTarget: '_self' | '_blank' = '_self';
  readonly expandedId = signal<string | null>(null);

  private _dataSource: ListDataSource<T, unknown>;
  private dataSourceSubscription = Subscription.EMPTY;
  private observer: IntersectionObserver | null = null;
  private sentinel: ElementRef<HTMLElement> | null = null;
  private selectLoadedRows = false;

  @Input({required: true})
  set dataSource(dataSource: ListDataSource<T, unknown>) {
    this.dataSourceSubscription.unsubscribe();
    this._dataSource = dataSource;
    this.dataSourceSubscription = new Subscription();

    this.dataSourceSubscription.add(dataSource.reset$.subscribe(() => {
      this.reset();
      this.selectedChange.emit([]);
    }));
    this.dataSourceSubscription.add(dataSource.completed$.subscribe(() => this.refreshObserver()));
    this.dataSourceSubscription.add(combineLatest([
      this.selection.changed.pipe(startWith(null)),
      dataSource.rows$,
    ]).subscribe(([, rows]) => {
      this.loadedLength.set(rows.length);
      const selectedIds = new Set(this.selection.selected.map(row => row.id));
      // With fixed pages, only a database-wide selection follows the page that is shown.
      const followsLoadedRows = this.selectLoadedRows && (!dataSource.currentPaging || this.allSelected());
      const appendedRows = followsLoadedRows && this.autoSelectAppendedRows
        ? rows.filter(row => !selectedIds.has(row.id))
        : [];
      if (appendedRows.length > 0) {
        this.selection.select(...appendedRows);
        if (!this.allSelected()) {
          this.selectedChange.emit(this.selection.selected);
        }
        return;
      }
      // The selection spans every page that has been visited, not just the rows shown now.
      this.selectedRows.set(this.selection.selected);
      this.isAllLoadedSelected.set(rows.length > 0 && rows.every(row => selectedIds.has(row.id)));
    }));
  }

  get dataSource(): ListDataSource<T, unknown> {
    return this._dataSource;
  }

  @Output() readonly rowClick = new EventEmitter<T>();
  @Output() readonly selectedChange = new EventEmitter<T[]>();
  @Output() readonly selectAll = new EventEmitter<void>();
  @Output() readonly sort = new EventEmitter<Sort>();

  @ViewChild('loadMoreSentinel')
  set loadMoreSentinel(value: ElementRef<HTMLElement> | undefined) {
    this.sentinel = value ?? null;
    this.createObserver();
  }

  @ContentChildren(ActionDirective, {read: TemplateRef, descendants: true})
  actionButtonTemplates: QueryList<TemplateRef<unknown>>;
  @ContentChildren(ExtraDirective, {read: TemplateRef, descendants: true})
  extraTemplates: QueryList<TemplateRef<unknown>>;
  @ContentChildren(FilterDirective, {read: TemplateRef, descendants: true})
  filterButtonTemplates: QueryList<TemplateRef<unknown>>;
  @ContentChildren(ShortcutDirective, {read: TemplateRef, descendants: true})
  shortcutButtonTemplates: QueryList<TemplateRef<unknown>>;
  @ContentChild(PreviewDirective, {read: TemplateRef})
  previewTemplate: TemplateRef<unknown> | null = null;

  get expandable(): boolean {
    return !!this.previewTemplate;
  }

  readonly selection = new SelectionModel<T>(true, []);
  selectedRowIndex: number = null;

  ngOnDestroy(): void {
    this.observer?.disconnect();
    this.dataSourceSubscription.unsubscribe();
  }

  reset(): void {
    this.selectLoadedRows = false;
    this.selection.clear();
    this.allSelected.set(false);
    this.selectedRowIndex = null;
    this.expandedId.set(null);
  }

  toggleExpanded(item: T): void {
    this.expandedId.update(id => id === item.id ? null : item.id);
  }

  private activateRow(item: T): void {
    if (this.expandable) {
      this.toggleExpanded(item);
    } else {
      this.rowClick.emit(item);
    }
  }

  onSortChange(sort: Sort): void {
    this.sort.emit(sort);
  }

  onRowClick(item: T, event?: Event): void {
    if (event?.target instanceof Element && event.target.closest('a, button, input, [role="button"]')) {
      return;
    }
    this.activateRow(item);
  }

  onRowKeydown(item: T, event: KeyboardEvent): void {
    if (event.key === 'Enter'
      && !(event.target instanceof Element && event.target.closest('a, button, input, [role="button"]'))) {
      event.preventDefault();
      this.activateRow(item);
    }
  }

  onPrimaryLink(item: T, event: Event): void {
    event.stopPropagation();
    if (this.detailLinkTarget === '_blank') {
      return;
    }
    event.preventDefault();
    this.rowClick.emit(item);
  }

  detailHref(item: T): string {
    return this.detailPath ? `/config/${this.detailPath}/${item.id}` : item.id;
  }

  onMasterCheckboxToggle(checked: boolean): void {
    this.selectLoadedRows = checked;
    this.allSelected.set(false);
    const shownIds = new Set(this.dataSource.snapshot.map(row => row.id));
    if (checked) {
      this.selection.select(...this.dataSource.snapshot);
    } else {
      // Only the shown rows: selections made on other pages are kept.
      this.selection.deselect(...this.selection.selected.filter(row => shownIds.has(row.id)));
    }
    this.selectedChange.emit(this.selection.selected);
  }

  onCheckboxToggle(item: T): void {
    this.selectLoadedRows = false;
    this.allSelected.set(false);
    const selected = this.selection.selected.find(row => row.id === item.id);
    if (selected) {
      this.selection.deselect(selected);
    } else {
      this.selection.select(item);
    }
    this.selectedChange.emit(this.selection.selected);
  }

  onSelectAll(): void {
    this.selectLoadedRows = true;
    this.allSelected.set(true);
    this.selectAll.emit();
  }

  onSelectionChipKeydown(event: KeyboardEvent): void {
    if (event.key !== 'Enter' && event.key !== ' ') {
      return;
    }
    event.preventDefault();
    if (this.allSelected()) {
      this.onDeselectAll();
    } else {
      this.onSelectAll();
    }
  }

  onDeselectAll(): void {
    this.selectLoadedRows = false;
    this.allSelected.set(false);
    this.selection.clear();
    this.selectedChange.emit(this.selection.selected);
  }

  retry(): void {
    this.dataSource?.retry();
  }

  isChecked(item: T): boolean {
    return this.selection.selected.some(selected => selected.id === item.id);
  }

  private createObserver(): void {
    this.observer?.disconnect();
    if (!this.sentinel || typeof IntersectionObserver === 'undefined') {
      return;
    }
    this.observer = new IntersectionObserver(entries => {
      if (entries.some(entry => entry.isIntersecting)) {
        this.dataSource?.loadMore();
      }
    }, {rootMargin: '400px 0px'});
    this.observer.observe(this.sentinel.nativeElement);
  }

  private refreshObserver(): void {
    queueMicrotask(() => this.createObserver());
  }
}
