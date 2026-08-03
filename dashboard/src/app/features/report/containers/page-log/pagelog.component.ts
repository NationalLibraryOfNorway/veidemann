import {ChangeDetectionStrategy, Component, computed, DestroyRef, Signal} from '@angular/core';
import {ActivatedRoute, Router, RouterLink} from '@angular/router';
import {AsyncPipe} from '@angular/common';
import {MatIcon} from '@angular/material/icon';
import {PageEvent} from '@angular/material/paginator';
import {MatProgressBar} from '@angular/material/progress-bar';
import {SortDirection} from '@angular/material/sort';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {FlexDirective, LayoutDirective} from '@ngbracket/ngx-layout';
import {MatMenuItem} from '@angular/material/menu';
import {combineLatest, merge, Observable} from 'rxjs';
import {distinctUntilChanged, map, startWith} from 'rxjs/operators';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';

import {ErrorService} from '../../../../core';
import {ActionDirective} from '../../../../shared/directives';
import {Sort} from '../../../../shared/func';
import {ListDataSource, ListItem, PageLog} from '../../../../shared/models';
import {PageLogListComponent} from '../../components';
import {PageLogQueryComponent} from '../../components/page-log-query/page-log-query.component';
import {equalPageLogQuery, pageLogQueryFromParamMap, unknownPageLength} from '../../func';
import {PageLogQuery, PageLogService} from '../../services/pagelog.service';

@Component({
  selector: 'app-pagelog',
  templateUrl: './pagelog.component.html',
  styleUrls: ['./pagelog.component.css'],
  imports: [
    ActionDirective,
    AsyncPipe,
    FlexDirective,
    LayoutDirective,
    MatIcon,
    MatMenuItem,
    MatProgressBar,
    PageLogQueryComponent,
    PageLogListComponent,
    RouterLink,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class PageLogComponent {
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];
  readonly pageSize: Signal<number>;
  readonly pageIndex: Signal<number>;
  readonly sortDirection: Signal<SortDirection>;
  readonly sortActive: Signal<string>;
  readonly query: Signal<PageLogQuery>;
  readonly dataSource: ListDataSource<PageLog, PageLogQuery>;
  readonly pageLength$: Observable<number>;
  readonly loading$: Observable<boolean>;

  constructor(private route: ActivatedRoute,
              private router: Router,
              private pageLogService: PageLogService,
              private errorService: ErrorService,
              private abilityService: AbilityServiceSignal<MongoAbility>,
              destroyRef: DestroyRef) {
    this.can = this.abilityService.can;

    const queryParamMap = toSignal(this.route.queryParamMap, {requireSync: true});
    this.query = computed(
      () => pageLogQueryFromParamMap(queryParamMap()),
      {equal: equalPageLogQuery}
    );
    this.pageSize = computed(() => this.query().pageSize);
    this.pageIndex = computed(() => this.query().pageIndex);
    this.sortDirection = computed(() => this.query().direction);
    this.sortActive = computed(() => this.query().active);

    const query$ = toObservable(this.query);
    this.dataSource = ListDataSource.fromQuery({
      query$,
      load: query => this.pageLogService.search(query),
      destroyRef,
      capacity: query => query.watch ? query.pageSize : 0,
    });
    this.pageLength$ = merge(
      this.dataSource.reset$.pipe(map(() => 0)),
      this.dataSource.completed$.pipe(map(({query, rows}) => unknownPageLength(query, rows)))
    ).pipe(
      startWith(0)
    );
    this.loading$ = combineLatest([this.dataSource.loading$, this.pageLogService.loading$]).pipe(
      map(([listLoading, operationLoading]) => listLoading || operationLoading),
      distinctUntilChanged()
    );
  }

  onRowClick(item: ListItem) {
    if (item !== null) {
      this.router.navigate([item.id], {
        relativeTo: this.route,
      }).catch(error => this.errorService.dispatch(error));
    }
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

  onQueryChange(query: Partial<PageLogQuery>) {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParamsHandling: 'merge',
      queryParams: {
        p: query.pageIndex || null,
        s: query.pageSize || null,
        uri: query.uri || null,
        job_execution_id: query.jobExecutionId || null,
        execution_id: query.executionId || null,
        watch: query.watch || null
      },
    }).catch(error => this.errorService.dispatch(error));
  }
}
