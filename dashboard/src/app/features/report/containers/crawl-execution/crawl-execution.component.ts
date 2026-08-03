import {ChangeDetectionStrategy, Component, computed, DestroyRef, Signal} from '@angular/core';
import {CommonModule} from '@angular/common';
import {ActivatedRoute, Router, RouterModule} from '@angular/router';
import {MatDialog} from '@angular/material/dialog';
import {MatIconModule} from '@angular/material/icon';
import {PageEvent} from '@angular/material/paginator';
import {MatProgressBarModule} from '@angular/material/progress-bar';
import {SortDirection} from '@angular/material/sort';
import {AbilityService} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {MatMenuModule} from '@angular/material/menu';
import {FlexDirective, LayoutDirective} from '@ngbracket/ngx-layout';
import {combineLatest, merge, Observable} from 'rxjs';
import {distinctUntilChanged, map, startWith} from 'rxjs/operators';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';

import {ControllerApiService, ErrorService, SnackBarService} from '../../../../core';
import {ActionDirective, FilterDirective, ShortcutDirective} from '../../../../shared/directives';
import {Sort} from '../../../../shared/func';
import {ConfigObject, Kind, ListDataSource} from '../../../../shared/models';
import {CrawlExecutionState, CrawlExecutionStatus} from '../../../../shared/models/report';
import {AbortCrawlDialogComponent} from '../../components/abort-crawl-dialog/abort-crawl-dialog.component';
import {CrawlExecutionStatusListComponent, CrawlExecutionStatusQueryComponent} from '../../components';
import {crawlExecutionQueryFromParamMap, equalCrawlExecutionQuery, unknownPageLength} from '../../func';
import {CrawlExecutionService, CrawlExecutionStatusQuery} from '../../services';

@Component({
  selector: 'app-crawl-execution',
  templateUrl: './crawl-execution.component.html',
  styleUrls: ['./crawl-execution.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    ActionDirective,
    CommonModule,
    CrawlExecutionStatusListComponent,
    CrawlExecutionStatusQueryComponent,
    FilterDirective,
    FlexDirective,
    LayoutDirective,
    MatIconModule,
    MatMenuModule,
    MatProgressBarModule,
    RouterModule,
    ShortcutDirective,
  ]
})
export class CrawlExecutionComponent {
  readonly CrawlExecutionState = CrawlExecutionState;
  readonly Kind = Kind;
  readonly ability$: Observable<MongoAbility>;
  readonly pageSize: Signal<number>;
  readonly pageIndex: Signal<number>;
  readonly sortDirection: Signal<SortDirection>;
  readonly sortActive: Signal<string>;
  readonly query: Signal<CrawlExecutionStatusQuery>;
  readonly dataSource: ListDataSource<CrawlExecutionStatus, CrawlExecutionStatusQuery>;
  readonly pageLength$: Observable<number>;
  readonly loading$: Observable<boolean>;
  readonly crawlJobOptions: ConfigObject[];

  constructor(private route: ActivatedRoute,
              private router: Router,
              private crawlExecutionService: CrawlExecutionService,
              private errorService: ErrorService,
              private dialog: MatDialog,
              private controllerApiService: ControllerApiService,
              private snackBarService: SnackBarService,
              private abilityService: AbilityService<MongoAbility>,
              destroyRef: DestroyRef) {
    this.crawlJobOptions = this.route.snapshot.data['options'].crawlJobs;
    this.ability$ = this.abilityService.ability$;

    const queryParamMap = toSignal(this.route.queryParamMap, {requireSync: true});
    this.query = computed(
      () => crawlExecutionQueryFromParamMap(queryParamMap()),
      {equal: equalCrawlExecutionQuery}
    );
    this.pageSize = computed(() => this.query().pageSize);
    this.pageIndex = computed(() => this.query().pageIndex);
    this.sortDirection = computed(() => this.query().direction);
    this.sortActive = computed(() => this.query().active);

    const query$ = toObservable(this.query);
    this.dataSource = ListDataSource.fromQuery({
      query$,
      load: query => this.crawlExecutionService.search(query),
      destroyRef,
      capacity: query => query.watch ? query.pageSize : 0,
    });
    this.pageLength$ = merge(
      this.dataSource.reset$.pipe(map(() => 0)),
      this.dataSource.completed$.pipe(map(({query, rows}) => unknownPageLength(query, rows)))
    ).pipe(
      startWith(0)
    );
    this.loading$ = combineLatest([this.dataSource.loading$, this.crawlExecutionService.loading$]).pipe(
      map(([listLoading, operationLoading]) => listLoading || operationLoading),
      distinctUntilChanged()
    );
  }

  onQueryChange(query: Partial<CrawlExecutionStatusQuery>) {
    const queryParams = {
      state: query.stateList || null,
      seed_id: query.seedId || null,
      job_id: query.jobId || null,
      job_execution_id: query.jobExecutionId || null,
      start_time_to: query.startTimeTo || null,
      start_time_from: query.startTimeFrom || null,
      has_error: query.hasError || null,
      watch: query.watch || null
    };
    this.router.navigate([], {relativeTo: this.route, queryParams})
      .catch(error => this.errorService.dispatch(error));
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

  isDone(item: CrawlExecutionStatus): boolean {
    return CrawlExecutionStatus.DONE_STATES.includes(item.state);
  }

  canAbort(state: CrawlExecutionState) {
    return !CrawlExecutionStatus.DONE_STATES.includes(state);
  }

  onAbortCrawlExecution(crawlExecutionStatus: CrawlExecutionStatus) {
    const dialogRef = this.dialog.open(AbortCrawlDialogComponent, {
      disableClose: true,
      autoFocus: true,
      data: {crawlExecutionStatus}
    });
    dialogRef.afterClosed().subscribe(executionId => {
      if (executionId) {
        this.controllerApiService.abortCrawlExecution(executionId).subscribe(crawlExecStatus => {
          if (crawlExecStatus.state === CrawlExecutionState.ABORTED_MANUAL) {
            this.snackBarService.openSnackBar('Crawl aborted');
            this.dataSource.reload({retainRows: this.query().watch});
          }
        });
      }
    });
  }
}
