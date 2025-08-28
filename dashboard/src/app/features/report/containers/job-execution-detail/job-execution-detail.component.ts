import { Component, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { BrowserModule } from '@angular/platform-browser';
import { ActivatedRoute } from '@angular/router';
import { combineLatest, merge, Observable } from 'rxjs';
import { filter, map, switchMap, takeWhile } from 'rxjs/operators';
import { ControllerApiService, SnackBarService } from '../../../../core';
import { Detail } from '../../../../shared/func';
import { JobExecutionState, JobExecutionStatus } from '../../../../shared/models';
import { JobExecutionStatusComponent } from '../../components';
import { AbortCrawlDialogComponent } from '../../components/abort-crawl-dialog/abort-crawl-dialog.component';
import { DetailDirective } from '../../directives';
import { JobExecutionService } from '../../services';
import { JobExecutionShortcutsComponent } from '../../components/job-execution-shortcuts/job-execution-shortcuts.component';
import { CommonModule } from '@angular/common';

@Component({
    selector: 'app-crawl-log-detail',
    templateUrl: './job-execution-detail.component.html',
    styleUrls: ['./job-execution-detail.component.css'],
    standalone: true,
    imports: [
      CommonModule,
      JobExecutionStatusComponent,
      JobExecutionShortcutsComponent,
    ]
})
export class JobExecutionDetailComponent extends DetailDirective<JobExecutionStatus> implements OnInit {
  readonly JobExecutionState = JobExecutionState;

  declare protected query$: Observable<Detail>;

  constructor(protected override route: ActivatedRoute,
              protected override service: JobExecutionService,
              protected controllerApiService: ControllerApiService,
              protected dialog: MatDialog,
              protected snackBarService: SnackBarService) {
    super(route, service);
  }

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
      disableClose: true,
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
