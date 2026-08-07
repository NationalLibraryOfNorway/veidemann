import {ChangeDetectionStrategy, Component, DestroyRef, ErrorHandler, inject, OnInit} from '@angular/core';
import {BehaviorSubject, EMPTY, Observable, Subject, timer} from 'rxjs';
import {catchError, startWith, switchMap, take} from 'rxjs/operators';
import {MatCardModule} from '@angular/material/card';
import {MatDialog} from '@angular/material/dialog';
import {Sort, SortDirection} from '@angular/material/sort';
import {Router} from '@angular/router';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {AbilityServiceSignal} from "@casl/angular";
import {MongoAbility} from '@casl/ability';
import {ControllerApiService} from '../../../core';
import {CrawlerStatus} from '../../../shared/models/controller/controller.model';
import {CrawlerStatusDialogComponent} from '../crawlerstatus-dialog/crawlerstatus-dialog.component';
import {AsyncPipe} from '@angular/common';
import {CrawlerStatusComponent} from '../crawlerstatus/crawlerstatus.component';
import {JobExecutionState, JobExecutionStatus, ListDataSource} from '../../../shared/models';
import {JobExecutionService, JobExecutionStatusQuery} from '../../report/services';
import {RunningCrawlsComponent} from '../running-crawls/running-crawls.component';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.css'],
  imports: [
    AsyncPipe,
    CrawlerStatusComponent,
    MatCardModule,
    RunningCrawlsComponent,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class DashboardComponent implements OnInit {
  private errorHandler = inject(ErrorHandler);
  private controllerApiService = inject(ControllerApiService);
  private dialog = inject(MatDialog);
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);
  private jobExecutionService = inject(JobExecutionService);
  private router = inject(Router);
  private destroyRef = inject(DestroyRef);

  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  updateRunStatus = new Subject<void>();
  crawlerStatus$: Observable<CrawlerStatus>;
  latestJobsDataSource: ListDataSource<JobExecutionStatus, JobExecutionStatusQuery> | null = null;
  readonly latestJobsQuery = new BehaviorSubject<JobExecutionStatusQuery>({
    active: 'startTime',
    direction: 'desc',
    jobId: '',
    startTimeFrom: '',
    startTimeTo: '',
    stateList: [JobExecutionState.RUNNING],
    watch: false,
  });

  constructor() {
    this.can = this.abilityService.can;
  }

  ngOnInit(): void {
    this.crawlerStatus$ = this.updateRunStatus.pipe(
      startWith(undefined),
      switchMap(() => this.controllerApiService.getCrawlerStatus().pipe(
        catchError(error => {
          this.errorHandler.handleError(error);
          return EMPTY;
        }),
      )),
    );
    if (this.can('read', 'jobexecution')) {
      this.latestJobsDataSource = ListDataSource.fromQuery({
        query$: this.latestJobsQuery,
        load: (query, range) => this.jobExecutionService.search(query, range),
        destroyRef: this.destroyRef,
      });
      timer(15_000, 15_000).pipe(takeUntilDestroyed(this.destroyRef)).subscribe(
        () => this.latestJobsDataSource?.refreshLoaded()
      );
    }
  }

  get selectedStates(): readonly JobExecutionState[] {
    return this.latestJobsQuery.value.stateList;
  }

  get sortActive(): string {
    return this.latestJobsQuery.value.active;
  }

  get sortDirection(): SortDirection {
    return this.latestJobsQuery.value.direction;
  }

  onSelectedStatesChange(stateList: readonly JobExecutionState[]): void {
    this.latestJobsQuery.next({...this.latestJobsQuery.value, stateList: [...stateList]});
  }

  onLatestJobsSort(sort: Sort): void {
    this.latestJobsQuery.next({
      ...this.latestJobsQuery.value,
      active: sort.active,
      direction: sort.direction,
    });
  }

  onJobExecutionClick(row: JobExecutionStatus): void {
    this.router.navigate(['/report', 'crawlexecution'], {
      queryParams: {job_execution_id: row.id},
    }).catch(error => this.errorHandler.handleError(error));
  }

  onChangeRunStatus(shouldPause: boolean) {
    this.dialog.open(CrawlerStatusDialogComponent, {
      disableClose: true,
      autoFocus: true,
      data: {shouldPause}
    }).afterClosed().subscribe(changeStatus => {
      if (changeStatus) {
        const changeRunStatus$ = shouldPause
          ? this.controllerApiService.pauseCrawler()
          : this.controllerApiService.unpauseCrawler();
        changeRunStatus$.pipe(take(1)).subscribe(() => this.updateRunStatus.next());
      }
    });
  }


}
