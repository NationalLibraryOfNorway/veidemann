import {ChangeDetectionStrategy, Component, ErrorHandler, OnInit, inject} from '@angular/core';
import {MatDialog} from '@angular/material/dialog';
import {combineLatest, concat, Observable, of, timer} from 'rxjs';
import {defaultIfEmpty, distinctUntilChanged, map, shareReplay, switchMap, takeWhile} from 'rxjs/operators';
import {Router} from '@angular/router';
import {MatButtonModule} from '@angular/material/button';
import {MatIconModule} from '@angular/material/icon';
import {MatTooltipModule} from '@angular/material/tooltip';
import {ControllerApiService, SnackBarService} from '../../../../core';
import {CrawlExecutionState, CrawlExecutionStatus} from '../../../../shared/models';
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
      MatButtonModule,
      MatIconModule,
      MatTooltipModule,
    ],
})
export class CrawlExecutionDetailComponent extends DetailDirective<CrawlExecutionStatus> implements OnInit {
  protected override service = inject(CrawlExecutionService);
  protected controllerApiService = inject(ControllerApiService);
  protected dialog = inject(MatDialog);
  protected snackBarService = inject(SnackBarService);
  private router = inject(Router);
  private errorHandler = inject(ErrorHandler);
  queueSize$: Observable<number | null>;
  watching$: Observable<boolean>;
  readonly CrawlExecutionStatus = CrawlExecutionStatus;
  readonly watchUpdatesLabel = $localize`:@@executionWatchUpdatesAction:Watch updates`;
  readonly stopWatchingLabel = $localize`:@@executionStopWatchingAction:Stop watching`;


  override ngOnInit() {
    super.ngOnInit();

    this.watching$ = this.query$.pipe(
      map(query => query.watch),
      distinctUntilChanged(),
    );
    this.item$ = this.query$.pipe(
      switchMap(query => this.service.get({id: query.id, watch: false}).pipe(
        switchMap(item => query.watch && !CrawlExecutionStatus.DONE_STATES.includes(item.state)
          ? concat(
            of(item),
            this.service.get({id: query.id, watch: true}).pipe(
              takeWhile(update => !CrawlExecutionStatus.DONE_STATES.includes(update.state), true),
            ),
          )
          : of(item)),
      )),
      shareReplay({bufferSize: 1, refCount: true}),
    );
    this.queueSize$ = combineLatest([this.item$, this.watching$]).pipe(
      switchMap(([item, watching]) => (
        watching && !CrawlExecutionStatus.DONE_STATES.includes(item.state) ? timer(0, 15_000) : of(0)
      ).pipe(
        switchMap(() => this.controllerApiService
          .queueCountsForCrawlExecutions([item.id])
          .pipe(
            map(counts => counts.get(item.id) ?? 0),
            defaultIfEmpty(null),
          )),
      )),
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
