import {ChangeDetectionStrategy, Component, computed, DestroyRef, Signal} from '@angular/core';
import {AsyncPipe} from '@angular/common';
import {MatDialog} from '@angular/material/dialog';
import {MatIconModule} from '@angular/material/icon';
import {PageEvent} from '@angular/material/paginator';
import {MatProgressBarModule} from '@angular/material/progress-bar';
import {SortDirection} from '@angular/material/sort';
import {ActivatedRoute, Router, RouterLink} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {FlexDirective, LayoutDirective} from '@ngbracket/ngx-layout';
import {MatMenuItem} from '@angular/material/menu';
import {combineLatest, merge, Observable} from 'rxjs';
import {distinctUntilChanged, map, startWith} from 'rxjs/operators';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';

import {ControllerApiService, ErrorService, SnackBarService} from '../../../../core';
import {ActionDirective, FilterDirective, ShortcutDirective} from '../../../../shared/directives';
import {Sort} from '../../../../shared/func';
import {
  ConfigObject,
  JobExecutionState,
  jobExecutionStates,
  JobExecutionStatus,
  Kind,
  ListDataSource
} from '../../../../shared/models';
import {AbortCrawlDialogComponent} from '../../components/abort-crawl-dialog/abort-crawl-dialog.component';
import {JobExecutionStatusListComponent, JobExecutionStatusQueryComponent} from '../../components';
import {equalJobExecutionQuery, jobExecutionQueryFromParamMap, unknownPageLength} from '../../func';
import {JobExecutionService, JobExecutionStatusQuery} from '../../services';

@Component({
  selector: 'app-job-execution',
  templateUrl: './job-execution.component.html',
  styleUrls: ['./job-execution.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    ActionDirective,
    AsyncPipe,
    FilterDirective,
    FlexDirective,
    JobExecutionStatusQueryComponent,
    JobExecutionStatusListComponent,
    LayoutDirective,
    MatIconModule,
    MatMenuItem,
    MatProgressBarModule,
    RouterLink,
    ShortcutDirective,
  ]
})
export class JobExecutionComponent {
  readonly jobExecutionStates = jobExecutionStates;
  readonly JobExecutionState = JobExecutionState;
  readonly crawlJobOptions: ConfigObject[];
  readonly Kind = Kind;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];
  readonly pageSize: Signal<number>;
  readonly pageIndex: Signal<number>;
  readonly sortDirection: Signal<SortDirection>;
  readonly sortActive: Signal<string>;
  readonly query: Signal<JobExecutionStatusQuery>;
  readonly dataSource: ListDataSource<JobExecutionStatus, JobExecutionStatusQuery>;
  readonly pageLength$: Observable<number>;
  readonly loading$: Observable<boolean>;

  constructor(private route: ActivatedRoute,
              private router: Router,
              private jobExecutionService: JobExecutionService,
              private errorService: ErrorService,
              private dialog: MatDialog,
              private controllerApiService: ControllerApiService,
              private snackBarService: SnackBarService,
              private abilityService: AbilityServiceSignal<MongoAbility>,
              destroyRef: DestroyRef) {
    this.crawlJobOptions = this.route.snapshot.data['options'].crawlJobs;
    this.can = this.abilityService.can;

    const queryParamMap = toSignal(this.route.queryParamMap, {requireSync: true});
    this.query = computed(
      () => jobExecutionQueryFromParamMap(queryParamMap()),
      {equal: equalJobExecutionQuery}
    );
    this.pageSize = computed(() => this.query().pageSize);
    this.pageIndex = computed(() => this.query().pageIndex);
    this.sortDirection = computed(() => this.query().direction);
    this.sortActive = computed(() => this.query().active);

    const query$ = toObservable(this.query);
    this.dataSource = ListDataSource.fromQuery({
      query$,
      load: query => this.jobExecutionService.search(query),
      destroyRef,
      capacity: query => query.watch ? query.pageSize : 0,
    });
    this.pageLength$ = merge(
      this.dataSource.reset$.pipe(map(() => 0)),
      this.dataSource.completed$.pipe(map(({query, rows}) => unknownPageLength(query, rows)))
    ).pipe(
      startWith(0)
    );
    this.loading$ = combineLatest([this.dataSource.loading$, this.jobExecutionService.loading$]).pipe(
      map(([listLoading, operationLoading]) => listLoading || operationLoading),
      distinctUntilChanged()
    );
  }

  onQueryChange(query: Partial<JobExecutionStatusQuery>) {
    const queryParams = {
      state: query.stateList && query.stateList.length ? query.stateList : null,
      job_id: query.jobId || null,
      start_time_to: query.startTimeTo || null,
      start_time_from: query.startTimeFrom || null,
      watch: query.watch || null,
    };
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams,
      queryParamsHandling: 'merge',
    }).catch(error => this.errorService.dispatch(error));
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

  onAbortJobExecution(jobExecutionStatus: JobExecutionStatus) {
    const dialogRef = this.dialog.open(AbortCrawlDialogComponent, {
      disableClose: true,
      autoFocus: true,
      data: {jobExecutionStatus}
    });
    dialogRef.afterClosed().subscribe(executionId => {
      if (executionId) {
        this.controllerApiService.abortJobExecution(executionId).subscribe(jobExecStatus => {
          if (jobExecStatus.state === JobExecutionState.ABORTED_MANUAL) {
            this.snackBarService.openSnackBar('Job aborted');
            this.dataSource.reload({retainRows: this.query().watch});
          }
        });
      }
    });
  }
}
