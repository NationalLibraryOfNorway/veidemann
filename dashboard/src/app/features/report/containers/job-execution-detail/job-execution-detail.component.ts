import {Component, OnInit, ChangeDetectionStrategy, ErrorHandler, inject} from '@angular/core';
import {create} from '@bufbuild/protobuf';
import {MatDialog} from '@angular/material/dialog';
import {combineLatest, merge, Observable, of} from 'rxjs';
import {catchError, defaultIfEmpty, filter, map, shareReplay, switchMap, takeWhile, toArray} from 'rxjs/operators';
import {ControllerApiService, ReportApiService, SnackBarService} from '../../../../core';
import {FieldMaskSchema} from '../../../../../api/commons/v1/resources_pb';
import {CrawlExecutionStatus_State} from '../../../../../api/frontier/v1/resources_pb';
import {CrawlExecutionsListRequestSchema} from '../../../../../api/report/v1/report_pb';
import {Detail} from '../../../../shared/func';
import {
  CrawlExecutionStatus,
  JobExecutionState,
  JobExecutionStatus,
} from '../../../../shared/models';
import {JobExecutionStatusComponent} from '../../components';
import {AbortCrawlDialogComponent} from '../../components/abort-crawl-dialog/abort-crawl-dialog.component';
import {DetailDirective} from '../../directives';
import {JobExecutionService} from '../../services';
import {
  JobExecutionShortcutHelpersComponent
} from '../../components/job-execution-shortcuts/job-execution-shortcuts.component';
import {ExecutionAbortActionComponent} from '../../components/execution-abort-action/execution-abort-action.component';
import {CommonModule} from '@angular/common';

@Component({
  selector: 'app-crawl-log-detail',
  templateUrl: './job-execution-detail.component.html',
  styleUrls: ['../detail-layout.scss'],
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    JobExecutionStatusComponent,
    JobExecutionShortcutHelpersComponent,
    ExecutionAbortActionComponent,
  ]
})
export class JobExecutionDetailComponent extends DetailDirective<JobExecutionStatus> implements OnInit {
  protected override service = inject(JobExecutionService);
  protected controllerApiService = inject(ControllerApiService);
  private reportApiService = inject(ReportApiService);
  private errorHandler = inject(ErrorHandler);
  protected dialog = inject(MatDialog);
  protected snackBarService = inject(SnackBarService);

  readonly JobExecutionState = JobExecutionState;
  queueSize$: Observable<number | null>;

  declare protected query$: Observable<Detail>;

  override ngOnInit() {
    super.ngOnInit();

    const item$: Observable<JobExecutionStatus> = this.query$.pipe(
      map(({id}) => ({id, watch: false})),
      switchMap(query => this.service.get(query)),
    );

    const watchedItem$: Observable<JobExecutionStatus> = combineLatest([this.query$, item$]).pipe(
      filter(([query, item]) => query.watch && !JobExecutionStatus.DONE_STATES.includes(item.state)),
      switchMap(([query]) => this.service.get(query).pipe(
        takeWhile(item => query.watch || !JobExecutionStatus.DONE_STATES.includes((item.state)), true),
      )),
    );

    this.item$ = merge(item$, watchedItem$).pipe(
      shareReplay({bufferSize: 1, refCount: true}),
    );
    this.queueSize$ = this.item$.pipe(
      switchMap(item => this.getQueueSize(item)),
      shareReplay({bufferSize: 1, refCount: true}),
    );
  }

  private getQueueSize(item: JobExecutionStatus): Observable<number | null> {
    const activeStates = [
      CrawlExecutionStatus_State.CREATED,
      CrawlExecutionStatus_State.FETCHING,
      CrawlExecutionStatus_State.SLEEPING,
    ];
    const queryTemplate = new CrawlExecutionStatus({
      jobExecutionId: item.id,
      jobId: item.jobId,
    });
    const request = create(CrawlExecutionsListRequestSchema, {
      queryTemplate: CrawlExecutionStatus.toProto(queryTemplate),
      queryMask: create(FieldMaskSchema, {paths: ['jobExecutionId', 'jobId']}),
      returnedFieldsMask: create(FieldMaskSchema, {paths: ['id']}),
      state: activeStates,
    });

    return this.reportApiService.listCrawlExecutionsUnchecked(request).pipe(
      map(execution => execution.id),
      toArray(),
      map(ids => [...new Set(ids.filter(Boolean))]),
      switchMap(ids => ids.length === 0
        ? of(0)
        : this.controllerApiService.queueCountForCrawlExecutions(ids).pipe(
          map(response => response.count),
          defaultIfEmpty(null),
        )),
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
