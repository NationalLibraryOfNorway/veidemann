import {Component, OnInit, ChangeDetectionStrategy, ErrorHandler, inject} from '@angular/core';
import {MatDialog} from '@angular/material/dialog';
import {combineLatest, concat, Observable, of, timer} from 'rxjs';
import {catchError, defaultIfEmpty, distinctUntilChanged, map, shareReplay, switchMap, takeWhile} from 'rxjs/operators';
import {Router} from '@angular/router';
import {MatButtonModule} from '@angular/material/button';
import {MatIconModule} from '@angular/material/icon';
import {MatTooltipModule} from '@angular/material/tooltip';
import {ControllerApiService, SnackBarService} from '../../../../core';
import {Detail} from '../../../../shared/func';
import {JobExecutionState, JobExecutionStatus} from '../../../../shared/models';
import {JobExecutionStatusComponent} from '../../components';
import {AbortCrawlDialogComponent} from '../../components/abort-crawl-dialog/abort-crawl-dialog.component';
import {DetailDirective} from '../../directives';
import {JobExecutionService} from '../../services';
import {
  JobExecutionShortcutHelpersComponent
} from '../../components/job-execution-shortcuts/job-execution-shortcuts.component';
import {ExecutionAbortActionComponent} from '../../components/execution-abort-action/execution-abort-action.component';
import {CommonModule} from '@angular/common';
import {DetailOverflowComponent} from '../../../../shared/components';

@Component({
  selector: 'app-crawl-log-detail',
  templateUrl: './job-execution-detail.component.html',
  styleUrls: ['../detail-layout.scss'],
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    DetailOverflowComponent,
    JobExecutionStatusComponent,
    JobExecutionShortcutHelpersComponent,
    ExecutionAbortActionComponent,
    MatButtonModule,
    MatIconModule,
    MatTooltipModule,
  ]
})
export class JobExecutionDetailComponent extends DetailDirective<JobExecutionStatus> implements OnInit {
  protected override service = inject(JobExecutionService);
  protected controllerApiService = inject(ControllerApiService);
  private errorHandler = inject(ErrorHandler);
  private router = inject(Router);
  protected dialog = inject(MatDialog);
  protected snackBarService = inject(SnackBarService);

  readonly JobExecutionState = JobExecutionState;
  readonly JobExecutionStatus = JobExecutionStatus;
  readonly watchUpdatesLabel = $localize`:@@executionWatchUpdatesAction:Watch updates`;
  readonly stopWatchingLabel = $localize`:@@executionStopWatchingAction:Stop watching`;
  queueSize$: Observable<number | null>;
  watching$: Observable<boolean>;

  declare protected query$: Observable<Detail>;

  override ngOnInit() {
    super.ngOnInit();

    this.watching$ = this.query$.pipe(
      map(query => query.watch),
      distinctUntilChanged(),
    );
    this.item$ = this.query$.pipe(
      switchMap(query => this.service.get({id: query.id, watch: false}).pipe(
        switchMap(item => query.watch && !JobExecutionStatus.DONE_STATES.includes(item.state)
          ? concat(
            of(item),
            this.service.get({id: query.id, watch: true}).pipe(
              takeWhile(update => !JobExecutionStatus.DONE_STATES.includes(update.state), true),
            ),
          )
          : of(item)),
      )),
      shareReplay({bufferSize: 1, refCount: true}),
    );
    this.queueSize$ = combineLatest([this.item$, this.watching$]).pipe(
      switchMap(([item, watching]) => (
        watching && !JobExecutionStatus.DONE_STATES.includes(item.state) ? timer(0, 15_000) : of(0)
      ).pipe(switchMap(() => this.getQueueSize(item)))),
      shareReplay({bufferSize: 1, refCount: true}),
    );
  }

  toggleWatch(watching: boolean): void {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: {watch: watching ? null : true},
      queryParamsHandling: 'merge',
    }).catch(error => this.errorHandler.handleError(error));
  }

  private getQueueSize(item: JobExecutionStatus): Observable<number | null> {
    return this.controllerApiService.queueCountsForJobExecutions([item.id]).pipe(
      map(counts => counts.get(item.id) ?? 0),
      defaultIfEmpty(null),
      catchError(error => {
        this.errorHandler.handleError(error);
        return of(null);
      }),
    );
  }

  onAbortJobExecution(jobExecutionStatus: JobExecutionStatus) {
    const dialogRef = this.dialog.open(AbortCrawlDialogComponent, {
      disableClose: false,
      autoFocus: true,
      data: {jobExecutionStatus}
    });
    dialogRef.afterClosed()
      .subscribe(executionId => {
        if (executionId) {
          this.controllerApiService.abortJobExecution(executionId).subscribe(jobExecStatus => {
            if (jobExecStatus.state === JobExecutionState.ABORTED_MANUAL) {
              this.snackBarService.openSnackBar('Job aborted');
              this.reload.next();
            }
          });
        }
      });
  }
}
