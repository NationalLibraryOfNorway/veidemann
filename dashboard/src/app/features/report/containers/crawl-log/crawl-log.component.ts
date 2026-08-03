import {ChangeDetectionStrategy, Component, computed, DestroyRef, Signal} from '@angular/core';
import {AsyncPipe} from '@angular/common';
import {MatIconModule} from '@angular/material/icon';
import {PageEvent} from '@angular/material/paginator';
import {MatProgressBarModule} from '@angular/material/progress-bar';
import {SortDirection} from '@angular/material/sort';
import {ActivatedRoute, Router, RouterModule} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {FlexDirective, LayoutDirective} from '@ngbracket/ngx-layout';
import {MatMenuItem} from '@angular/material/menu';
import {combineLatest, merge, Observable} from 'rxjs';
import {distinctUntilChanged, map, startWith} from 'rxjs/operators';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';

import {ErrorService} from '../../../../core';
import {ActionDirective, ShortcutDirective} from '../../../../shared/directives';
import {Sort} from '../../../../shared/func';
import {CrawlLog, ListDataSource} from '../../../../shared/models';
import {CrawlLogListComponent} from '../../components';
import {CrawlLogQueryComponent} from '../../components/crawl-log-query/crawl-log-query.component';
import {crawlLogQueryFromParamMap, equalCrawlLogQuery, unknownPageLength} from '../../func';
import {CrawlLogQuery, CrawlLogService} from '../../services';

@Component({
  selector: 'app-crawl-log',
  templateUrl: './crawl-log.component.html',
  styleUrls: ['./crawl-log.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    ActionDirective,
    AsyncPipe,
    CrawlLogListComponent,
    CrawlLogQueryComponent,
    FlexDirective,
    LayoutDirective,
    MatIconModule,
    MatMenuItem,
    MatProgressBarModule,
    RouterModule,
    ShortcutDirective,
  ]
})
export class CrawlLogComponent {
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];
  readonly pageLength$: Observable<number>;
  readonly pageSize: Signal<number>;
  readonly pageIndex: Signal<number>;
  readonly sortDirection: Signal<SortDirection>;
  readonly sortActive: Signal<string>;
  readonly query: Signal<CrawlLogQuery>;
  readonly dataSource: ListDataSource<CrawlLog, CrawlLogQuery>;
  readonly loading$: Observable<boolean>;

  constructor(private route: ActivatedRoute,
              private router: Router,
              private crawlLogService: CrawlLogService,
              private errorService: ErrorService,
              private abilityService: AbilityServiceSignal<MongoAbility>,
              destroyRef: DestroyRef) {
    this.can = this.abilityService.can;

    const queryParamMap = toSignal(this.route.queryParamMap, {requireSync: true});
    this.query = computed(
      () => crawlLogQueryFromParamMap(queryParamMap()),
      {equal: equalCrawlLogQuery}
    );
    this.pageSize = computed(() => this.query().pageSize);
    this.pageIndex = computed(() => this.query().pageIndex);
    this.sortDirection = computed(() => this.query().direction);
    this.sortActive = computed(() => this.query().active);

    const query$ = toObservable(this.query);
    this.dataSource = ListDataSource.fromQuery({
      query$,
      load: query => this.crawlLogService.search(query),
      destroyRef,
      capacity: query => query.watch ? query.pageSize : 0,
    });
    this.pageLength$ = merge(
      this.dataSource.reset$.pipe(map(() => 0)),
      this.dataSource.completed$.pipe(map(({query, rows}) => unknownPageLength(query, rows)))
    ).pipe(
      startWith(0)
    );
    this.loading$ = combineLatest([this.dataSource.loading$, this.crawlLogService.loading$]).pipe(
      map(([listLoading, operationLoading]) => listLoading || operationLoading),
      distinctUntilChanged()
    );
  }

  onSort(sort: Sort) {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParamsHandling: 'merge',
      queryParams: {sort: sort.active && sort.direction ? `${sort.active}:${sort.direction}` : null}
    }).catch(error => this.errorService.dispatch(error));
  }

  onPage(page: PageEvent) {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParamsHandling: 'merge',
      queryParams: {p: page.pageIndex, s: page.pageSize}
    }).catch(error => this.errorService.dispatch(error));
  }

  onQueryChange(query: Partial<CrawlLogQuery>) {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParamsHandling: 'merge',
      queryParams: {
        p: query.pageIndex || null,
        s: query.pageSize || null,
        job_execution_id: query.jobExecutionId || null,
        execution_id: query.executionId || null,
        watch: query.watch || null
      },
    }).catch(error => this.errorService.dispatch(error));
  }
}
