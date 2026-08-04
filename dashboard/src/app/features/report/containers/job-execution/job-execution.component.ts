import { ChangeDetectionStrategy, Component, computed, DestroyRef, Signal, inject } from '@angular/core';
import {AsyncPipe} from '@angular/common';
import {MatDialog} from '@angular/material/dialog';
import {MatIconModule} from '@angular/material/icon';
import {MatProgressBarModule} from '@angular/material/progress-bar';
import {SortDirection} from '@angular/material/sort';
import {ActivatedRoute, Router, RouterLink} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {MatMenuItem} from '@angular/material/menu';
import {MatTooltip} from '@angular/material/tooltip';
import {combineLatest, Observable} from 'rxjs';
import {distinctUntilChanged, map} from 'rxjs/operators';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';

import {ControllerApiService, ErrorService, SnackBarService} from '../../../../core';
import {ActionDirective, ExtraDirective, FilterDirective, ShortcutDirective} from '../../../../shared/directives';
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
import {equalJobExecutionQuery, jobExecutionQueryFromParamMap} from '../../func';
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
    ExtraDirective,
    JobExecutionStatusQueryComponent,
    JobExecutionStatusListComponent,
    MatIconModule,
    MatMenuItem,
    MatProgressBarModule,
    MatTooltip,
    RouterLink,
    ShortcutDirective,
  ]
})
export class JobExecutionComponent {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private jobExecutionService = inject(JobExecutionService);
  private errorService = inject(ErrorService);
  private dialog = inject(MatDialog);
  private controllerApiService = inject(ControllerApiService);
  private snackBarService = inject(SnackBarService);
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  readonly jobExecutionStates = jobExecutionStates;
  readonly JobExecutionState = JobExecutionState;
  readonly crawlJobOptions: ConfigObject[];
  readonly Kind = Kind;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];
  readonly sortDirection: Signal<SortDirection>;
  readonly sortActive: Signal<string>;
  readonly query: Signal<JobExecutionStatusQuery>;
  readonly dataSource: ListDataSource<JobExecutionStatus, JobExecutionStatusQuery>;
  readonly loading$: Observable<boolean>;

  constructor() {
    const destroyRef = inject(DestroyRef);

    this.crawlJobOptions = this.route.snapshot.data['options'].crawlJobs;
    this.can = this.abilityService.can;

    const queryParamMap = toSignal(this.route.queryParamMap, {requireSync: true});
    this.query = computed(
      () => jobExecutionQueryFromParamMap(queryParamMap()),
      {equal: equalJobExecutionQuery}
    );
    this.sortDirection = computed(() => this.query().direction);
    this.sortActive = computed(() => this.query().active);

    const query$ = toObservable(this.query);
    this.dataSource = ListDataSource.fromQuery({
      query$,
      load: (query, range) => this.jobExecutionService.search(query, range),
      destroyRef,
      capacity: query => query.watch ? 100 : 0,
    });
    this.loading$ = combineLatest([this.dataSource.loading$, this.jobExecutionService.loading$]).pipe(
      map(([listLoading, operationLoading]) => listLoading || operationLoading),
      distinctUntilChanged()
    );
  }

  onQueryChange(query: Partial<JobExecutionStatusQuery>) {
    const queryParams = {
      p: null,
      s: null,
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
      queryParams: {p: null, s: null, sort: sort.active && sort.direction ? `${sort.active}:${sort.direction}` : null}
    }).catch(error => this.errorService.dispatch(error));
  }

  onRowClick(row: JobExecutionStatus): void {
    this.router.navigate([row.id], {relativeTo: this.route})
      .catch(error => this.errorService.dispatch(error));
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
