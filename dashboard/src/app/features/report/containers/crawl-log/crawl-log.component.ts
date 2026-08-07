import {ChangeDetectionStrategy, Component, computed, DestroyRef, effect, ErrorHandler, Signal, inject} from '@angular/core';
import {AsyncPipe} from '@angular/common';
import {MatProgressBarModule} from '@angular/material/progress-bar';
import {SortDirection} from '@angular/material/sort';
import {ActivatedRoute, Router} from '@angular/router';
import {combineLatest, Observable} from 'rxjs';
import {distinctUntilChanged, map} from 'rxjs/operators';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';

import {compareListValues, Sort} from '../../../../shared/func';
import {CrawlLog, ListDataSource} from '../../../../shared/models';
import {CrawlLogListComponent, LogListShortcutsComponent} from '../../components';
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
    LogListShortcutsComponent,
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
      capacity: query => query.watch ? 100 : 0,
    });
    effect(() => this.applySort(this.query().active, this.query().direction));
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
        job_execution_id: query.jobExecutionId || null,
        execution_id: query.executionId || null,
        watch: query.watch || null
      },
    }).catch(error => this.errorHandler.handleError(error));
  }

  onRowClick(row: CrawlLog): void {
    this.router.navigate([row.id], {relativeTo: this.route})
      .catch(error => this.errorHandler.handleError(error));
  }

  private applySort(active: string, direction: SortDirection): void {
    if (!active || !direction) {
      this.dataSource.setComparator(null);
      return;
    }
    const selectors: Record<string, {value: (row: CrawlLog) => string | number; type: 'string' | 'number' | 'date'}> = {
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
