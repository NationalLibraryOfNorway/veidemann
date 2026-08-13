import {ChangeDetectionStrategy, Component, computed, DestroyRef, ErrorHandler, inject, Signal} from '@angular/core';
import {AsyncPipe} from '@angular/common';
import {MatIconModule} from '@angular/material/icon';
import {MatProgressBarModule} from '@angular/material/progress-bar';
import {SortDirection} from '@angular/material/sort';
import {ActivatedRoute, Router, RouterLink} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {MatMenuItem} from '@angular/material/menu';
import {Observable} from 'rxjs';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';

import {FilterDirective} from '../../../../shared/directives';
import {Sort} from '../../../../shared/func';
import {
  ConfigObject,
  jobExecutionStates,
  JobExecutionStatus,
  ListDataSource
} from '../../../../shared/models';
import {JobExecutionStatusListComponent, JobExecutionStatusQueryComponent} from '../../components';
import {equalJobExecutionQuery, jobExecutionQueryFromParamMap} from '../../func';
import {
  ExecutionQueueCounts,
  ExecutionQueueCountService,
  JobExecutionService,
  JobExecutionStatusQuery,
} from '../../services';

@Component({
  selector: 'app-job-execution',
  templateUrl: './job-execution.component.html',
  styleUrls: ['./job-execution.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    AsyncPipe,
    FilterDirective,
    JobExecutionStatusQueryComponent,
    JobExecutionStatusListComponent,
    MatIconModule,
    MatMenuItem,
    MatProgressBarModule,
    RouterLink,
  ]
})
export class JobExecutionComponent {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private jobExecutionService = inject(JobExecutionService);
  private executionQueueCountService = inject(ExecutionQueueCountService);
  private errorHandler = inject(ErrorHandler);
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  readonly jobExecutionStates = jobExecutionStates;
  readonly crawlJobOptions: ConfigObject[];
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];
  readonly sortDirection: Signal<SortDirection>;
  readonly sortActive: Signal<string>;
  readonly query: Signal<JobExecutionStatusQuery>;
  readonly dataSource: ListDataSource<JobExecutionStatus, JobExecutionStatusQuery>;
  readonly loading$: Observable<boolean>;
  readonly queueCounts$: Observable<ExecutionQueueCounts>;
  readonly emptyQueueCounts: ExecutionQueueCounts = new Map();

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
    });
    this.loading$ = this.dataSource.initialLoading$;
    this.queueCounts$ = this.executionQueueCountService.forJobExecutions(this.dataSource);
  }

  onQueryChange(query: Partial<JobExecutionStatusQuery>) {
    const queryParams = {
      p: null,
      s: null,
      state: query.stateList && query.stateList.length ? query.stateList : null,
      job_id: query.jobId || null,
      start_time_to: query.startTimeTo || null,
      start_time_from: query.startTimeFrom || null,
    };
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams,
      queryParamsHandling: 'merge',
    }).catch(error => this.errorHandler.handleError(error));
  }

  onSort(sort: Sort) {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParamsHandling: 'merge',
      queryParams: {p: null, s: null, sort: sort.active && sort.direction ? `${sort.active}:${sort.direction}` : null}
    }).catch(error => this.errorHandler.handleError(error));
  }

  onRowClick(row: JobExecutionStatus): void {
    this.router.navigate([row.id], {relativeTo: this.route})
      .catch(error => this.errorHandler.handleError(error));
  }

  onRefresh(): void {
    this.dataSource.refreshLoaded();
  }
}
