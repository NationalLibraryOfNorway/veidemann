import { ChangeDetectionStrategy, Component, OnInit, inject } from '@angular/core';
import {MatDialog} from '@angular/material/dialog';
import {combineLatest, merge, Observable} from 'rxjs';
import {defaultIfEmpty, filter, map, mergeMap, shareReplay, switchMap, takeWhile} from 'rxjs/operators';
import {ControllerApiService, SnackBarService} from '../../../../core';
import {CrawlExecutionState, CrawlExecutionStatus, ExecutionId} from '../../../../shared/models';
import {AbortCrawlDialogComponent} from '../../components/abort-crawl-dialog/abort-crawl-dialog.component';
import {DetailDirective} from '../../directives';
import {CrawlExecutionService} from '../../services';
import {
  CrawlExecutionShortcutHelpersComponent
} from '../../components/crawl-execution-shortcuts/crawl-execution-shortcuts.component';
import {ExecutionAbortActionComponent} from '../../components/execution-abort-action/execution-abort-action.component';
import {CrawlExecutionStatusComponent} from '../../components';
import {CommonModule} from '@angular/common';
import {DetailOverflowComponent} from '../../../../shared/components';

@Component({
    selector: 'app-crawl-execution',
    templateUrl: './crawl-execution.component.html',
    styleUrls: ['../detail-layout.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush,
    standalone: true,
    imports: [
      CommonModule,
      DetailOverflowComponent,
      CrawlExecutionStatusComponent,
      CrawlExecutionShortcutHelpersComponent,
      ExecutionAbortActionComponent,
    ],
})
export class CrawlExecutionDetailComponent extends DetailDirective<CrawlExecutionStatus> implements OnInit {
  protected override service = inject(CrawlExecutionService);
  protected controllerApiService = inject(ControllerApiService);
  protected dialog = inject(MatDialog);
  protected snackBarService = inject(SnackBarService);
  queueSize$: Observable<number | null>;


  override ngOnInit() {
    super.ngOnInit();

    const item$: Observable<CrawlExecutionStatus> = this.query$.pipe(
      map(({id}) => ({id, watch: false})),
      mergeMap(query => this.service.get(query)),
    );

    const watchedItem$: Observable<CrawlExecutionStatus> = combineLatest([
      this.query$, item$
    ]).pipe(
      // only watch if job execution isn't in one of the done states
      filter(([, item]) => !CrawlExecutionStatus.DONE_STATES.includes(item.state)),
      switchMap(([query]) => this.service.get(query).pipe(
        takeWhile(item => !CrawlExecutionStatus.DONE_STATES.includes((item.state)), true),
      )),
    );

    this.item$ = merge(item$, watchedItem$).pipe(
      shareReplay({bufferSize: 1, refCount: true}),
    );
    this.queueSize$ = this.item$.pipe(
      switchMap(item => this.controllerApiService
        .queueCountForCrawlExecution(new ExecutionId({id: item.id}))
        .pipe(
          map(response => response.count),
          defaultIfEmpty(null),
        )),
      shareReplay({bufferSize: 1, refCount: true}),
    );
  }

  canAbort(crawlExecutionStatus: CrawlExecutionStatus): boolean {
    return !CrawlExecutionStatus.DONE_STATES.includes(crawlExecutionStatus.state);
  }

  onAbortCrawlExecution(crawlExecutionStatus: CrawlExecutionStatus) {
    const dialogRef = this.dialog.open(AbortCrawlDialogComponent, {
      disableClose: false,
      autoFocus: true,
      data: {crawlExecutionStatus}
    });
    dialogRef.afterClosed()
      .subscribe(executionId => {
        if (executionId) {
          this.controllerApiService.abortCrawlExecution(executionId).subscribe(crawlExecStatus => {
            if (crawlExecStatus.state === CrawlExecutionState.ABORTED_MANUAL) {
              this.snackBarService.openSnackBar('Crawl aborted');
              this.reload.next();
            }
          });
        }
      });
  }
}
