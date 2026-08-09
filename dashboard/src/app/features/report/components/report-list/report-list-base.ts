import {AsyncPipe, NgTemplateOutlet} from '@angular/common';
import {
  ContentChildren,
  Directive,
  ElementRef,
  EventEmitter,
  Input,
  OnDestroy,
  Output,
  QueryList,
  TemplateRef,
  ViewChild,
} from '@angular/core';
import {MatButtonModule} from '@angular/material/button';
import {MatIconModule} from '@angular/material/icon';
import {MatMenuModule} from '@angular/material/menu';
import {MatSortModule, Sort, SortDirection} from '@angular/material/sort';
import {MatTableModule} from '@angular/material/table';
import {MatTooltipModule} from '@angular/material/tooltip';
import {Subscription} from 'rxjs';

import {ListDataSource, ListItem} from '../../../../shared/models';
import {ActionDirective, ExtraDirective, FilterDirective, ShortcutDirective} from '../../../../shared/directives';

export const REPORT_LIST_IMPORTS = [
  AsyncPipe,
  MatButtonModule,
  MatIconModule,
  MatMenuModule,
  MatSortModule,
  MatTableModule,
  MatTooltipModule,
  NgTemplateOutlet,
];

@Directive()
export abstract class ReportListBaseComponent<T extends ListItem> implements OnDestroy {
  @Input() sortDirection: SortDirection = '';
  @Input() sortActive = '';
  @Input() displayedColumns: string[] = [];

  private _loadMoreDisabled = false;

  @Input()
  set loadMoreDisabled(value: boolean) {
    if (this._loadMoreDisabled !== value) {
      this._loadMoreDisabled = value;
      this.refreshObserver();
    }
  }

  get loadMoreDisabled(): boolean {
    return this._loadMoreDisabled;
  }

  private _dataSource: ListDataSource<T, unknown>;
  private dataSourceSubscription = Subscription.EMPTY;
  private observer: IntersectionObserver | null = null;
  private sentinel: ElementRef<HTMLElement> | null = null;

  @Input({required: true})
  set dataSource(dataSource: ListDataSource<T, unknown>) {
    this.dataSourceSubscription.unsubscribe();
    this._dataSource = dataSource;
    this.dataSourceSubscription = dataSource.completed$.subscribe(() => this.refreshObserver());
  }

  get dataSource(): ListDataSource<T, unknown> {
    return this._dataSource;
  }

  @Output() readonly rowClick = new EventEmitter<T>();
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

  ngOnDestroy(): void {
    this.observer?.disconnect();
    this.dataSourceSubscription.unsubscribe();
  }

  onSortChange(sort: Sort): void {
    this.sort.emit(sort);
  }

  onRowClick(item: T, event?: Event): void {
    if (event?.target instanceof Element && event.target.closest('a, button, input, [role="button"]')) {
      return;
    }
    this.rowClick.emit(item);
  }

  onRowKeydown(item: T, event: KeyboardEvent): void {
    if (event.key === 'Enter'
      && !(event.target instanceof Element && event.target.closest('a, button, input, [role="button"]'))) {
      event.preventDefault();
      this.rowClick.emit(item);
    }
  }

  onPrimaryLink(item: T, event: Event): void {
    event.preventDefault();
    event.stopPropagation();
    this.rowClick.emit(item);
  }

  retry(): void {
    this.dataSource?.retry();
  }

  private createObserver(): void {
    this.observer?.disconnect();
    if (!this.sentinel || typeof IntersectionObserver === 'undefined') {
      return;
    }
    this.observer = new IntersectionObserver(entries => {
      if (!this.loadMoreDisabled && entries.some(entry => entry.isIntersecting)) {
        this.dataSource?.loadMore();
      }
    }, {rootMargin: '400px 0px'});
    this.observer.observe(this.sentinel.nativeElement);
  }

  private refreshObserver(): void {
    queueMicrotask(() => this.createObserver());
  }
}
