import { Component, OnInit, ChangeDetectionStrategy, inject } from '@angular/core';
import {MatDialog} from '@angular/material/dialog';
import {combineLatest, merge, Observable} from 'rxjs';
import {filter, map, switchMap, takeWhile} from 'rxjs/operators';
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
  protected dialog = inject(MatDialog);
  protected snackBarService = inject(SnackBarService);

  readonly JobExecutionState = JobExecutionState;

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

    this.item$ = merge(item$, watchedItem$);
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
