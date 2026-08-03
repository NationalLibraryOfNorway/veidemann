import {SelectionModel} from '@angular/cdk/collections';
import {
  ContentChildren,
  Directive,
  EventEmitter,
  Input,
  OnDestroy,
  Output,
  signal,
  TemplateRef,
  ViewChild
} from '@angular/core';
import {MatPaginator, MatPaginatorModule, PageEvent} from '@angular/material/paginator';
import {MatSort, MatSortModule, Sort, SortDirection} from '@angular/material/sort';
import {combineLatest, Subscription} from 'rxjs';
import {startWith} from 'rxjs/operators';
import {Kind, ListDataSource, ListItem} from '../../../shared/models';
import {ActionDirective, ExtraDirective, FilterDirective, ShortcutDirective} from '../../directives';
import {NgTemplateOutlet} from '@angular/common';
import {FlexDirective, LayoutDirective} from '@ngbracket/ngx-layout';
import {MatTableModule} from '@angular/material/table';
import {MatCheckboxModule} from '@angular/material/checkbox';
import {MatTooltipModule} from '@angular/material/tooltip';
import {UrlFormatPipe} from '../../pipes/url-format.pipe';
import {MatIconModule} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {MatMenuModule} from '@angular/material/menu';
import {PreviewComponent} from '../../../features/config/components/preview/preview.component';

export const BASE_LIST_IMPORTS = [
  LayoutDirective,
  FlexDirective,
  MatButtonModule,
  MatCheckboxModule,
  MatIconModule,
  MatMenuModule,
  MatPaginatorModule,
  MatSortModule,
  MatTableModule,
  MatTooltipModule,
  NgTemplateOutlet,
  PreviewComponent,
  UrlFormatPipe
  ];

@Directive()
export abstract class BaseListComponent<T extends ListItem> implements OnDestroy {
  readonly Kind = Kind;
  readonly totalLength = signal(0);
  readonly allSelected = signal(false);
  readonly isAllInPageSelected = signal(false);
  readonly selectedRows = signal<readonly T[]>([]);

  @Input()
  set length(length: number) {
    this.totalLength.set(length ?? 0);
  }

  @Input()
  pageSize = 25;

  @Input()
  pageIndex = 0;

  @Input()
  pageOptions = [5, 10, 25, 50, 100];

  @Input()
  sortDirection: SortDirection = '';

  @Input()
  sortActive = 'name';

  @Input()
  displayedColumns: string[] = ['select', 'name', 'description', 'extra', 'action'];

  @Input()
  multiSelect = true;

  private _dataSource: ListDataSource<T, unknown>;
  private dataSourceSubscription = Subscription.EMPTY;

  @Input({required: true})
  set dataSource(dataSource: ListDataSource<T, unknown>) {
    this.dataSourceSubscription.unsubscribe();
    this._dataSource = dataSource;
    this.dataSourceSubscription = new Subscription();

    this.dataSourceSubscription.add(dataSource.reset$.subscribe(() => {
      this.reset();
      this.selectedChange.emit([]);
    }));

    this.dataSourceSubscription.add(combineLatest([
      this.selection.changed.pipe(startWith(null)),
      dataSource.rows$,
    ]).subscribe(([, rows]) => {
      this.selectedRows.set([...this.selection.selected]);
      this.isAllInPageSelected.set(
        rows.length === this.selection.selected.length && rows.length > 0
      );
    }));
  }

  get dataSource(): ListDataSource<T, unknown> {
    return this._dataSource;
  }

  @Output()
  rowClick: EventEmitter<T>;

  @Output()
  selectedChange: EventEmitter<T[]>;

  @Output()
  selectAll: EventEmitter<void>;

  @Output()
  sort: EventEmitter<Sort>;

  @Output()
  page: EventEmitter<PageEvent>;

  @ViewChild(MatSort, {static: true}) matSort: MatSort;
  @ViewChild(MatPaginator) paginator: MatPaginator;

  @ContentChildren(ActionDirective, {read: TemplateRef, descendants: true}) actionButtonTemplates;
  @ContentChildren(ExtraDirective, {read: TemplateRef, descendants: true}) extraTemplates;
  @ContentChildren(FilterDirective, {read: TemplateRef, descendants: true}) filterButtonTemplates;
  @ContentChildren(ShortcutDirective, {read: TemplateRef, descendants: true}) shortcutButtonTemplates;

  // selection of checked rows
  selection: SelectionModel<T>;
  selectedRow: T;

  // Keyboard navigation
  selectedRowIndex: number = null;

  constructor() {
    this.sort = new EventEmitter<Sort>();
    this.selectedChange = new EventEmitter<T[]>();
    this.selectAll = new EventEmitter<void>();
    this.page = new EventEmitter<PageEvent>();
    this.rowClick = new EventEmitter<T>();
    this.selection = new SelectionModel<T>(true, []);
  }

  ngOnDestroy(): void {
    this.dataSourceSubscription.unsubscribe();
  }

  reset() {
    this.selection.clear();
    this.selectedRow = null;
    this.allSelected.set(false);
    this.selectedRowIndex = null;
  }

  onSortChange(sort: Sort) {
    this.reset();
    this.sort.emit(sort);
  }

  onRowClick(item: T, event?: MouseEvent) {
    if (event?.target instanceof Element && event.target.closest('a')) {
      return;
    }

    this.allSelected.set(false);
    this.selectedRowIndex = null;
    this.selectedRow = this.selectedRow?.id === item.id ? null : item;
    this.rowClick.emit(this.selectedRow);
  }

  onMasterCheckboxToggle(checked: boolean) {
    this.selectedRow = null;
    this.allSelected.set(false);
    if (checked) {
      this.selection.select(...this.dataSource.snapshot);
      this.selectedChange.emit(this.selection.selected);
    } else {
      this.selection.clear();
      this.selectedChange.emit(this.selection.selected);
    }
  }

  onCheckboxToggle(item: T) {
    this.selectedRow = null;
    this.allSelected.set(false);
    this.selection.toggle(item);
    this.selectedChange.emit(this.selection.selected);
  }

  onSelectAll() {
    this.allSelected.set(true);
    this.selectAll.emit();
  }

  onDeselectAll() {
    this.allSelected.set(false);
    this.onMasterCheckboxToggle(false);
  }

  onPage(pageEvent: PageEvent) {
    this.reset();
    this.page.emit(pageEvent);
  }

  onKeyboardEvent(event: KeyboardEvent) {
    const itemsInPage = this.dataSource.snapshot.length;

    switch (event.key) {

      case 'ArrowDown':
        if (this.selectedRowIndex !== null) {
          if (this.selectedRowIndex + 1 <= itemsInPage - 1) {
            this.selectedRowIndex += 1;
            this.selectedRow = null; // collapse any open preview when navigating
            this.selectRowByIndex(this.selectedRowIndex);
          }
        } else {
          this.selectedRowIndex = 0;
          this.selectRowByIndex(0);
        }
        break;

      case 'ArrowUp':
        if (this.selectedRowIndex !== null) {
          if (this.selectedRowIndex - 1 >= 0) {
            this.selectedRowIndex -= 1;
            this.selectedRow = null; // collapse any open preview when navigating
            this.selectRowByIndex(this.selectedRowIndex);
          }
        }
        break;
      case 'S' :
        if (this.selectedRowIndex !== null) {
          const row = this.dataSource.snapshot[this.selectedRowIndex];
          if (row) {
            this.onCheckboxToggle(row);
          }
        }
        break;

      case 'A':
        this.onMasterCheckboxToggle(true);
        break;

      case 'Enter' :
        this.selectRowByIndex(this.selectedRowIndex, true);
        break;
    }
  }


  selectRowByIndex(index: number, expand?: boolean): void {
    const row = this.dataSource.snapshot[index];
    if (index === null || index === undefined || !row) {
      return;
    }

    this.selectedRowIndex = index;
    this.allSelected.set(false);
    document.getElementById('row' + index.toString())
      ?.scrollIntoView({behavior: 'smooth', block: 'end'});
    if (expand) {
      this.selectedRow = this.selectedRow?.id === row.id ? null : row;
      this.rowClick.emit(this.selectedRow);
      setTimeout(() => {
        document.getElementById('expandedPreviewRow' + index.toString())
          ?.scrollIntoView({behavior: 'smooth', block: 'center'});
      }, 250);
    }
  }

  isChecked(item: T): boolean {
    return this.selection.selected.find(selected => selected.id === item.id) !== undefined;
  }

  isSelected(item: T): boolean {
    return this.selectedRow ? this.selectedRow.id === item.id : false;
  }

  isDisabled(item: T): boolean {
    void item;
    return false;
  }
}
