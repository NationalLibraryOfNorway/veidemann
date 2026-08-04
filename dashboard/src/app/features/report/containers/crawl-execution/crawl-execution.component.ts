import { ChangeDetectionStrategy, Component, computed, DestroyRef, Signal, inject } from '@angular/core';
import {CommonModule} from '@angular/common';
import {ActivatedRoute, Router, RouterModule} from '@angular/router';
import {MatDialog} from '@angular/material/dialog';
import {MatIconModule} from '@angular/material/icon';
import {MatProgressBarModule} from '@angular/material/progress-bar';
import {SortDirection} from '@angular/material/sort';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {MatMenuModule} from '@angular/material/menu';
import {MatTooltip} from '@angular/material/tooltip';
import {combineLatest, Observable} from 'rxjs';
import {distinctUntilChanged, map} from 'rxjs/operators';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';

import {ControllerApiService, ErrorService, SnackBarService} from '../../../../core';
import {ActionDirective, ExtraDirective, FilterDirective, ShortcutDirective} from '../../../../shared/directives';
import {Sort} from '../../../../shared/func';
import {ConfigObject, Kind, ListDataSource} from '../../../../shared/models';
import {CrawlExecutionState, CrawlExecutionStatus} from '../../../../shared/models/report';
import {AbortCrawlDialogComponent} from '../../components/abort-crawl-dialog/abort-crawl-dialog.component';
import {CrawlExecutionStatusListComponent, CrawlExecutionStatusQueryComponent} from '../../components';
import {crawlExecutionQueryFromParamMap, equalCrawlExecutionQuery} from '../../func';
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
    ExtraDirective,
    MatIconModule,
    MatMenuModule,
    MatProgressBarModule,
    MatTooltip,
    RouterModule,
    ShortcutDirective,
  ]
})
export class CrawlExecutionComponent {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private crawlExecutionService = inject(CrawlExecutionService);
  private errorService = inject(ErrorService);
  private dialog = inject(MatDialog);
  private controllerApiService = inject(ControllerApiService);
  private snackBarService = inject(SnackBarService);
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  readonly CrawlExecutionState = CrawlExecutionState;
  readonly Kind = Kind;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];
  readonly sortDirection: Signal<SortDirection>;
  readonly sortActive: Signal<string>;
  readonly query: Signal<CrawlExecutionStatusQuery>;
  readonly dataSource: ListDataSource<CrawlExecutionStatus, CrawlExecutionStatusQuery>;
  readonly loading$: Observable<boolean>;
  readonly crawlJobOptions: ConfigObject[];

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
      capacity: query => query.watch ? 100 : 0,
    });
    this.loading$ = combineLatest([this.dataSource.loading$, this.crawlExecutionService.loading$]).pipe(
      map(([listLoading, operationLoading]) => listLoading || operationLoading),
      distinctUntilChanged()
    );
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
      watch: query.watch || null
    };
    this.router.navigate([], {relativeTo: this.route, queryParams})
      .catch(error => this.errorService.dispatch(error));
  }

  onSort(sort: Sort) {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParamsHandling: 'merge',
      queryParams: {p: null, s: null, sort: sort.active && sort.direction ? `${sort.active}:${sort.direction}` : null}
    }).catch(error => this.errorService.dispatch(error));
  }

  onRowClick(row: CrawlExecutionStatus): void {
    this.router.navigate([row.id], {relativeTo: this.route})
      .catch(error => this.errorService.dispatch(error));
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
