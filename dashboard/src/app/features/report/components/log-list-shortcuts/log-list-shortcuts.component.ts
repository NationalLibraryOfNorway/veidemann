import {AsyncPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, Input, OnChanges, inject} from '@angular/core';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {Observable, ReplaySubject, of} from 'rxjs';
import {catchError, defaultIfEmpty, distinctUntilChanged, map, shareReplay, startWith, switchMap} from 'rxjs/operators';

import {CrawlExecutionStatus, JobExecutionStatus} from '../../../../shared/models';
import {DetailOverflowComponent} from '../../../../shared/components';
import {CrawlExecutionService, JobExecutionService} from '../../services';
import {
  CrawlExecutionShortcutHelpersComponent,
} from '../crawl-execution-shortcuts/crawl-execution-shortcuts.component';
import {
  JobExecutionShortcutHelpersComponent,
} from '../job-execution-shortcuts/job-execution-shortcuts.component';

type LogListKind = 'pagelog' | 'crawllog';

type LogListContext =
  | {kind: 'crawlExecution'; status: CrawlExecutionStatus}
  | {kind: 'jobExecution'; status: JobExecutionStatus};

interface ContextIds {
  executionId: string;
  jobExecutionId: string;
}

@Component({
  selector: 'app-log-list-shortcuts',
  templateUrl: './log-list-shortcuts.component.html',
  styleUrls: ['./log-list-shortcuts.component.scss'],
  imports: [
    AsyncPipe,
    CrawlExecutionShortcutHelpersComponent,
    DetailOverflowComponent,
    JobExecutionShortcutHelpersComponent,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
})
export class LogListShortcutsComponent implements OnChanges {
  private readonly crawlExecutionService = inject(CrawlExecutionService);
  private readonly jobExecutionService = inject(JobExecutionService);
  private readonly abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);
  private readonly contextIds$ = new ReplaySubject<ContextIds>(1);

  @Input() executionId = '';
  @Input() jobExecutionId = '';
  @Input({required: true}) logKind: LogListKind;

  get actionsMenuLabel(): string {
    return this.logKind === 'crawllog'
      ? $localize`:@@crawlLogActionsMenuLabel:Crawl log actions`
      : $localize`:@@pageLogActionsMenuLabel:Page log actions`;
  }

  readonly context$: Observable<LogListContext | null> = this.contextIds$.pipe(
    distinctUntilChanged((previous, current) =>
      previous.executionId === current.executionId
      && previous.jobExecutionId === current.jobExecutionId),
    switchMap(({executionId, jobExecutionId}) => {
      if (executionId) {
        if (!this.abilityService.can('read', 'crawlexecution')) {
          return of(null);
        }
        return this.crawlExecutionService.get({id: executionId, watch: false}).pipe(
          map(status => ({kind: 'crawlExecution', status}) as LogListContext),
          defaultIfEmpty(null),
          catchError(() => of(null)),
          startWith(null),
        );
      }

      if (jobExecutionId) {
        if (!this.abilityService.can('read', 'jobexecution')) {
          return of(null);
        }
        return this.jobExecutionService.get({id: jobExecutionId, watch: false}).pipe(
          map(status => ({kind: 'jobExecution', status}) as LogListContext),
          defaultIfEmpty(null),
          catchError(() => of(null)),
          startWith(null),
        );
      }

      return of(null);
    }),
    shareReplay({bufferSize: 1, refCount: true}),
  );

  ngOnChanges(): void {
    this.contextIds$.next({
      executionId: this.executionId,
      jobExecutionId: this.jobExecutionId,
    });
  }
}
