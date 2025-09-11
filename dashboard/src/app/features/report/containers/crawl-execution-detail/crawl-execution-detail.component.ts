import { ChangeDetectionStrategy, Component, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { ActivatedRoute } from '@angular/router';
import { combineLatest, merge, Observable } from 'rxjs';
import { filter, map, mergeMap, switchMap, takeWhile } from 'rxjs/operators';
import { ControllerApiService, SnackBarService } from '../../../../core';
import { CrawlExecutionState, CrawlExecutionStatus } from '../../../../shared/models';
import { AbortCrawlDialogComponent } from '../../components/abort-crawl-dialog/abort-crawl-dialog.component';
import { DetailDirective } from '../../directives';
import { CrawlExecutionService } from '../../services';
import { CrawlExecutionShortcutsComponent } from '../../components/crawl-execution-shortcuts/crawl-execution-shortcuts.component';
import { CrawlExecutionStatusComponent } from '../../components';
import { CommonModule } from '@angular/common';
import {FlexLayoutModule} from '@ngbracket/ngx-layout';

@Component({
    selector: 'app-crawl-execution',
    templateUrl: './crawl-execution.component.html',
    styleUrls: ['./crawl-execution.component.css'],
    changeDetection: ChangeDetectionStrategy.OnPush,
    standalone: true,
    imports: [
      FlexLayoutModule,
      CommonModule,
      CrawlExecutionStatusComponent,
      CrawlExecutionShortcutsComponent,
    ],
})
export class CrawlExecutionDetailComponent extends DetailDirective<CrawlExecutionStatus> implements OnInit {

  constructor(protected override route: ActivatedRoute,
              protected crawlExecutionService: CrawlExecutionService,
              protected controllerApiService: ControllerApiService,
              protected dialog: MatDialog,
              protected snackBarService: SnackBarService) {
    super(route, crawlExecutionService);
  }

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
      filter(([_, item]) => !CrawlExecutionStatus.DONE_STATES.includes(item.state)),
      switchMap(([query]) => this.service.get(query).pipe(
        takeWhile(item => !CrawlExecutionStatus.DONE_STATES.includes((item.state)), true),
      )),
    );

    this.item$ = merge(item$, watchedItem$);
  }

  onAbortCrawlExecution(crawlExecutionStatus: CrawlExecutionStatus) {
    const dialogRef = this.dialog.open(AbortCrawlDialogComponent, {
      disableClose: true,
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
