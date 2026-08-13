import {ChangeDetectionStrategy, Component, computed, DestroyRef, effect, ErrorHandler, Signal, inject, signal} from '@angular/core';
import {AsyncPipe} from '@angular/common';
import {MatProgressBarModule} from '@angular/material/progress-bar';
import {SortDirection} from '@angular/material/sort';
import {ActivatedRoute, Router} from '@angular/router';
import {combineLatest, Observable} from 'rxjs';
import {distinctUntilChanged, map} from 'rxjs/operators';
import {takeUntilDestroyed, toObservable, toSignal} from '@angular/core/rxjs-interop';
import {MatChipsModule} from '@angular/material/chips';

import {compareListValues, Sort} from '../../../../shared/func';
import {CrawlLog, ListDataSource} from '../../../../shared/models';
import {
  CrawlLogListComponent,
  HttpStatusFamily,
  HttpStatusFilterComponent,
  httpStatusFamily,
  LogListShortcutsComponent,
  uniqueHttpStatusCodes,
} from '../../components';
import {CrawlLogQueryComponent} from '../../components/crawl-log-query/crawl-log-query.component';
import {crawlLogQueryFromParamMap, equalCrawlLogQuery} from '../../func';
import {CrawlLogQuery, CrawlLogService} from '../../services';

@Component({
  selector: 'app-crawl-log',
  templateUrl: './crawl-log.component.html',
  styleUrls: ['./crawl-log.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    AsyncPipe,
    CrawlLogListComponent,
    CrawlLogQueryComponent,
    HttpStatusFilterComponent,
    LogListShortcutsComponent,
    MatChipsModule,
    MatProgressBarModule,
  ]
})
export class CrawlLogComponent {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private crawlLogService = inject(CrawlLogService);
  private errorHandler = inject(ErrorHandler);
  readonly sortDirection: Signal<SortDirection>;
  readonly sortActive: Signal<string>;
  readonly query: Signal<CrawlLogQuery>;
  readonly dataSource: ListDataSource<CrawlLog, CrawlLogQuery>;
  readonly loading$: Observable<boolean>;
  readonly loadedContentTypes: Signal<readonly string[]>;
  readonly loadedMethods: Signal<readonly string[]>;
  readonly loadedStatusFamilies: Signal<readonly HttpStatusFamily[]>;
  readonly loadedStatusCodes: Signal<readonly number[]>;
  readonly selectedContentTypes = signal<string[]>([]);
  readonly selectedMethods = signal<string[]>([]);
  readonly selectedStatusFamilies = signal<HttpStatusFamily[]>([]);
  readonly selectedStatusCodes = signal<number[]>([]);
  readonly hasClientFilters = computed(() =>
    this.selectedContentTypes().length > 0 ||
    this.selectedMethods().length > 0 ||
    this.selectedStatusFamilies().length > 0 ||
    this.selectedStatusCodes().length > 0
  );

  constructor() {
    const destroyRef = inject(DestroyRef);

    const queryParamMap = toSignal(this.route.queryParamMap, {requireSync: true});
    this.query = computed(
      () => crawlLogQueryFromParamMap(queryParamMap()),
      {equal: equalCrawlLogQuery}
    );
    this.sortDirection = computed(() => this.query().direction);
    this.sortActive = computed(() => this.query().active);

    const serverQuery = computed(() => ({...this.query(), active: '', direction: '' as SortDirection}), {
      equal: equalCrawlLogQuery,
    });
    const query$ = toObservable(serverQuery);
    this.dataSource = ListDataSource.fromQuery({
      query$,
      load: (query, range) => this.crawlLogService.search(query, range),
      destroyRef,
    });
    this.loadedContentTypes = toSignal(this.dataSource.rows$.pipe(
      map(() => [...new Set(this.dataSource.loadedSnapshot
        .map(row => normalizeContentType(row.contentType))
        .filter(contentType => contentType.length > 0))]
        .sort((left, right) => left.localeCompare(right))),
      distinctUntilChanged((left, right) => arraysEqual(left, right))
    ), {initialValue: []});
    this.loadedMethods = toSignal(this.dataSource.rows$.pipe(
      map(() => uniqueLoadedValues(this.dataSource.loadedSnapshot.map(row => normalizeMethod(row.method)))),
      distinctUntilChanged((left, right) => arraysEqual(left, right))
    ), {initialValue: []});
    this.loadedStatusFamilies = toSignal(this.dataSource.rows$.pipe(
      map(() => [...new Set(this.dataSource.loadedSnapshot
        .map(row => httpStatusFamily(row.statusCode))
        .filter((family): family is HttpStatusFamily => family !== null))]
        .sort((left, right) => left - right)),
      distinctUntilChanged((left, right) => arraysEqual(left, right))
    ), {initialValue: []});
    this.loadedStatusCodes = toSignal(this.dataSource.rows$.pipe(
      map(() => uniqueHttpStatusCodes(this.dataSource.loadedSnapshot.map(row => row.statusCode))),
      distinctUntilChanged((left, right) => arraysEqual(left, right))
    ), {initialValue: []});
    this.dataSource.reset$.pipe(takeUntilDestroyed(destroyRef)).subscribe(() => {
      this.selectedContentTypes.set([]);
      this.selectedMethods.set([]);
      this.selectedStatusFamilies.set([]);
      this.selectedStatusCodes.set([]);
    });
    effect(() => this.applySort(this.query().active, this.query().direction));
    effect(() => this.applyClientFilters(
      this.selectedContentTypes(), this.selectedMethods(), this.selectedStatusFamilies(), this.selectedStatusCodes()
    ));
    this.loading$ = combineLatest([this.dataSource.loading$, this.crawlLogService.loading$]).pipe(
      map(([listLoading, operationLoading]) => listLoading || operationLoading),
      distinctUntilChanged()
    );
  }

  onSort(sort: Sort) {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParamsHandling: 'merge',
      queryParams: {p: null, s: null, sort: sort.active && sort.direction ? `${sort.active}:${sort.direction}` : null}
    }).catch(error => this.errorHandler.handleError(error));
  }

  onQueryChange(query: Partial<CrawlLogQuery>) {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParamsHandling: 'merge',
      queryParams: {
        p: null,
        s: null,
        job_execution_id: null,
        execution_id: query.executionId || null,
      },
    }).catch(error => this.errorHandler.handleError(error));
  }

  onExecutionFilterRemove(): void {
    this.onQueryChange({...this.query(), executionId: ''});
  }

  onRowClick(row: CrawlLog): void {
    this.router.navigate([row.id], {relativeTo: this.route})
      .catch(error => this.errorHandler.handleError(error));
  }

  onContentTypeFilterChange(contentTypes: string[] | null): void {
    this.selectedContentTypes.set(contentTypes ?? []);
  }

  onStatusFamilyFilterChange(statusFamilies: HttpStatusFamily[]): void {
    this.selectedStatusFamilies.set(statusFamilies);
  }

  onExactStatusFilterChange(statusCodes: number[]): void {
    this.selectedStatusCodes.set(statusCodes);
  }

  onMethodFilterChange(methods: string[] | null): void {
    this.selectedMethods.set(methods ?? []);
  }

  private applyClientFilters(
    contentTypes: readonly string[], methods: readonly string[], statusFamilies: readonly HttpStatusFamily[],
    statusCodes: readonly number[]
  ): void {
    if (contentTypes.length === 0 && methods.length === 0 &&
      statusFamilies.length === 0 && statusCodes.length === 0) {
      this.dataSource.setPredicate(null);
      return;
    }
    const selectedContentTypes = new Set(contentTypes);
    const selectedMethods = new Set(methods);
    const selectedStatusFamilies = new Set(statusFamilies);
    const selectedStatusCodes = new Set(statusCodes);
    this.dataSource.setPredicate(row =>
      (selectedContentTypes.size === 0 || selectedContentTypes.has(normalizeContentType(row.contentType))) &&
      (selectedMethods.size === 0 || selectedMethods.has(normalizeMethod(row.method))) &&
      (selectedStatusFamilies.size === 0 && selectedStatusCodes.size === 0 ||
        selectedStatusFamilies.has(httpStatusFamily(row.statusCode) as HttpStatusFamily) ||
        selectedStatusCodes.has(row.statusCode))
    );
  }

  private applySort(active: string, direction: SortDirection): void {
    if (!active || !direction) {
      this.dataSource.setComparator(null);
      return;
    }
    const selectors: Record<string, {value: (row: CrawlLog) => string | number; type: 'string' | 'number' | 'date'}> = {
      method: {value: row => row.method, type: 'string'},
      requestedUri: {value: row => row.requestedUri, type: 'string'},
      timestamp: {value: row => row.timeStamp, type: 'date'},
      statusCode: {value: row => row.statusCode, type: 'number'},
      discoveryPath: {value: row => row.discoveryPath, type: 'string'},
      contentType: {value: row => row.contentType, type: 'string'},
    };
    const selector = selectors[active];
    this.dataSource.setComparator(selector
      ? (left, right) => compareListValues(selector.value(left), selector.value(right), direction, selector.type)
      : null);
  }
}

function normalizeContentType(contentType: string): string {
  return contentType.split(';', 1)[0].trim().toLowerCase();
}

function normalizeMethod(method: string): string {
  return method.trim().toLocaleUpperCase();
}

function uniqueLoadedValues(values: readonly string[]): string[] {
  return [...new Set(values.filter(value => value.length > 0))]
    .sort((left, right) => left.localeCompare(right));
}

function arraysEqual<T>(left: readonly T[], right: readonly T[]): boolean {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}
