import {ChangeDetectionStrategy, Component, computed, DestroyRef, effect, ErrorHandler, Signal, inject} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {AsyncPipe} from '@angular/common';
import {MatProgressBar} from '@angular/material/progress-bar';
import {SortDirection} from '@angular/material/sort';
import {combineLatest, Observable} from 'rxjs';
import {distinctUntilChanged, map} from 'rxjs/operators';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';

import {compareListValues, Sort} from '../../../../shared/func';
import {ListDataSource, ListItem, PageLog} from '../../../../shared/models';
import {LogListShortcutsComponent, PageLogListComponent} from '../../components';
import {PageLogQueryComponent} from '../../components/page-log-query/page-log-query.component';
import {equalPageLogQuery, pageLogQueryFromParamMap} from '../../func';
import {PageLogQuery, PageLogService} from '../../services/pagelog.service';

@Component({
  selector: 'app-pagelog',
  templateUrl: './pagelog.component.html',
  styleUrls: ['./pagelog.component.css'],
  imports: [
    AsyncPipe,
    LogListShortcutsComponent,
    MatProgressBar,
    PageLogQueryComponent,
    PageLogListComponent,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class PageLogComponent {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private pageLogService = inject(PageLogService);
  private errorHandler = inject(ErrorHandler);
  readonly sortDirection: Signal<SortDirection>;
  readonly sortActive: Signal<string>;
  readonly query: Signal<PageLogQuery>;
  readonly dataSource: ListDataSource<PageLog, PageLogQuery>;
  readonly loading$: Observable<boolean>;

  constructor() {
    const destroyRef = inject(DestroyRef);

    const queryParamMap = toSignal(this.route.queryParamMap, {requireSync: true});
    this.query = computed(
      () => pageLogQueryFromParamMap(queryParamMap()),
      {equal: equalPageLogQuery}
    );
    this.sortDirection = computed(() => this.query().direction);
    this.sortActive = computed(() => this.query().active);

    const serverQuery = computed(() => ({...this.query(), active: '', direction: '' as SortDirection}), {
      equal: equalPageLogQuery,
    });
    const query$ = toObservable(serverQuery);
    this.dataSource = ListDataSource.fromQuery({
      query$,
      load: (query, range) => this.pageLogService.search(query, range),
      destroyRef,
      capacity: query => query.watch ? 100 : 0,
    });
    effect(() => this.applySort(this.query().active, this.query().direction));
    this.loading$ = combineLatest([this.dataSource.loading$, this.pageLogService.loading$]).pipe(
      map(([listLoading, operationLoading]) => listLoading || operationLoading),
      distinctUntilChanged()
    );
  }

  onRowClick(item: ListItem) {
    if (item !== null) {
      this.router.navigate([item.id], {
        relativeTo: this.route,
      }).catch(error => this.errorHandler.handleError(error));
    }
  }

  onSort(sort: Sort) {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParamsHandling: 'merge',
      queryParams: {p: null, s: null, sort: sort.active && sort.direction ? `${sort.active}:${sort.direction}` : null}
    }).catch(error => this.errorHandler.handleError(error));
  }

  onQueryChange(query: Partial<PageLogQuery>) {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParamsHandling: 'merge',
      queryParams: {
        p: null,
        s: null,
        uri: query.uri || null,
        job_execution_id: query.jobExecutionId || null,
        execution_id: query.executionId || null,
        watch: query.watch || null
      },
    }).catch(error => this.errorHandler.handleError(error));
  }

  private applySort(active: string, direction: SortDirection): void {
    if (!active || !direction) {
      this.dataSource.setComparator(null);
      return;
    }
    const selectors: Record<string, (row: PageLog) => string | number> = {
      uri: row => row.uri,
      nrOfResources: row => row.resource?.length ?? 0,
      nrOfOutlinks: row => row.outlink?.length ?? 0,
    };
    const selector = selectors[active];
    this.dataSource.setComparator(selector
      ? (left, right) => compareListValues(
        selector(left), selector(right), direction, active === 'uri' ? 'string' : 'number')
      : null);
  }
}
