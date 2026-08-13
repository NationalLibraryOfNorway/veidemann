import {ChangeDetectionStrategy, Component, computed, DestroyRef, ErrorHandler, Signal, inject} from '@angular/core';
import {CommonModule} from '@angular/common';
import {ActivatedRoute, Router, RouterModule} from '@angular/router';
import {MatDialog} from '@angular/material/dialog';
import {MatIconModule} from '@angular/material/icon';
import {MatProgressBarModule} from '@angular/material/progress-bar';
import {SortDirection} from '@angular/material/sort';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {MatMenuModule} from '@angular/material/menu';
import {MatTooltipModule} from '@angular/material/tooltip';
import {Observable} from 'rxjs';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';

import {ControllerApiService, SnackBarService} from '../../../../core';
import {ActionDirective} from '../../../../shared/directives';
import {Sort} from '../../../../shared/func';
import {ConfigObject, ListDataSource} from '../../../../shared/models';
import {CrawlExecutionState, CrawlExecutionStatus} from '../../../../shared/models/report';
import {AbortCrawlDialogComponent} from '../../components/abort-crawl-dialog/abort-crawl-dialog.component';
import {CrawlExecutionStatusListComponent, CrawlExecutionStatusQueryComponent} from '../../components';
import {crawlExecutionQueryFromParamMap, equalCrawlExecutionQuery} from '../../func';
import {
  CrawlExecutionService,
  CrawlExecutionStatusQuery,
  ExecutionQueueCounts,
  ExecutionQueueCountService,
} from '../../services';

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
    MatIconModule,
    MatMenuModule,
    MatProgressBarModule,
    MatTooltipModule,
    RouterModule,
  ]
})
export class CrawlExecutionComponent {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private crawlExecutionService = inject(CrawlExecutionService);
  private executionQueueCountService = inject(ExecutionQueueCountService);
  private errorHandler = inject(ErrorHandler);
  private dialog = inject(MatDialog);
  private controllerApiService = inject(ControllerApiService);
  private snackBarService = inject(SnackBarService);
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  readonly CrawlExecutionState = CrawlExecutionState;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];
  readonly sortDirection: Signal<SortDirection>;
  readonly sortActive: Signal<string>;
  readonly query: Signal<CrawlExecutionStatusQuery>;
  readonly dataSource: ListDataSource<CrawlExecutionStatus, CrawlExecutionStatusQuery>;
  readonly loading$: Observable<boolean>;
  readonly queueCounts$: Observable<ExecutionQueueCounts>;
  readonly emptyQueueCounts: ExecutionQueueCounts = new Map();
  readonly crawlJobOptions: ConfigObject[];
  readonly hasActions = (row: CrawlExecutionStatus): boolean =>
    this.can('read', 'pagelog')
    || this.can('read', 'crawllog')
    || (this.canAbort(row.state) && this.can('abort', 'crawlexecution'));

  constructor() {
    const destroyRef = inject(DestroyRef);

    this.crawlJobOptions = this.route.snapshot.data['options'].crawlJobs;
    this.can = this.abilityService.can;

    const queryParamMap = toSignal(this.route.queryParamMap, {requireSync: true});
    this.query = computed(
      () => crawlExecutionQueryFromParamMap(queryParamMap()),
      {equal: equalCrawlExecutionQuery}
    );
    this.sortDirection = computed(() => this.query().direction);
    this.sortActive = computed(() => this.query().active);

    const query$ = toObservable(this.query);
    this.dataSource = ListDataSource.fromQuery({
      query$,
      load: (query, range) => this.crawlExecutionService.search(query, range),
      destroyRef,
    });
    this.loading$ = this.dataSource.initialLoading$;
    this.queueCounts$ = this.executionQueueCountService.forCrawlExecutions(this.dataSource);
  }

  onQueryChange(query: Partial<CrawlExecutionStatusQuery>) {
    const queryParams = {
      p: null,
      s: null,
      state: query.stateList || null,
      seed_id: query.seedId || null,
      job_id: query.jobId || null,
      job_execution_id: query.jobExecutionId || null,
      start_time_to: query.startTimeTo || null,
      start_time_from: query.startTimeFrom || null,
      has_error: query.hasError || null,
    };
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams,
      queryParamsHandling: 'merge',
    })
      .catch(error => this.errorHandler.handleError(error));
  }

  onSort(sort: Sort) {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParamsHandling: 'merge',
      queryParams: {p: null, s: null, sort: sort.active && sort.direction ? `${sort.active}:${sort.direction}` : null}
    }).catch(error => this.errorHandler.handleError(error));
  }

  onRowClick(row: CrawlExecutionStatus): void {
    this.router.navigate([row.id], {relativeTo: this.route})
      .catch(error => this.errorHandler.handleError(error));
  }

  onRefresh(): void {
    this.dataSource.refreshLoaded();
  }

  isDone(item: CrawlExecutionStatus): boolean {
    return CrawlExecutionStatus.DONE_STATES.includes(item.state);
  }

  canAbort(state: CrawlExecutionState) {
    return !CrawlExecutionStatus.DONE_STATES.includes(state);
  }

  onAbortCrawlExecution(crawlExecutionStatus: CrawlExecutionStatus) {
    const dialogRef = this.dialog.open(AbortCrawlDialogComponent, {
      disableClose: false,
      autoFocus: true,
      data: {crawlExecutionStatus}
    });
    dialogRef.afterClosed().subscribe(executionId => {
      if (executionId) {
        this.controllerApiService.abortCrawlExecution(executionId).subscribe(crawlExecStatus => {
          if (crawlExecStatus.state === CrawlExecutionState.ABORTED_MANUAL) {
            this.snackBarService.openSnackBar('Crawl aborted');
            this.dataSource.reload();
          }
        });
      }
    });
  }
}
