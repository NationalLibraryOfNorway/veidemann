import {AsyncPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, Input, OnChanges, inject} from '@angular/core';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {Observable, ReplaySubject, of} from 'rxjs';
import {catchError, defaultIfEmpty, distinctUntilChanged, shareReplay, startWith, switchMap} from 'rxjs/operators';

import {CrawlExecutionStatus} from '../../../../shared/models';
import {DetailOverflowComponent} from '../../../../shared/components';
import {CrawlExecutionService} from '../../services';
import {crawlExecutionStatePresentation} from '../../func';
import {JobNamePipe, SeedNamePipe} from '../../pipe';
import {
  CrawlExecutionShortcutHelpersComponent,
} from '../crawl-execution-shortcuts/crawl-execution-shortcuts.component';
import {ExecutionMetadataComponent} from '../execution-metadata/execution-metadata.component';

type LogListKind = 'pagelog' | 'crawllog';

@Component({
  selector: 'app-log-list-shortcuts',
  templateUrl: './log-list-shortcuts.component.html',
  styleUrls: ['./log-list-shortcuts.component.scss'],
  imports: [
    AsyncPipe,
    CrawlExecutionShortcutHelpersComponent,
    DetailOverflowComponent,
    ExecutionMetadataComponent,
    JobNamePipe,
    SeedNamePipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
})
export class LogListShortcutsComponent implements OnChanges {
  private readonly crawlExecutionService = inject(CrawlExecutionService);
  private readonly abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);
  private readonly executionId$ = new ReplaySubject<string>(1);

  readonly statePresentation = crawlExecutionStatePresentation;

  @Input() executionId = '';
  @Input({required: true}) logKind: LogListKind;

  get actionsMenuLabel(): string {
    return this.logKind === 'crawllog'
      ? $localize`:@@crawlLogActionsMenuLabel:Crawl log actions`
      : $localize`:@@pageLogActionsMenuLabel:Page log actions`;
  }

  readonly status$: Observable<CrawlExecutionStatus | null> = this.executionId$.pipe(
    distinctUntilChanged(),
    switchMap(executionId => {
      if (!executionId || !this.abilityService.can('read', 'crawlexecution')) {
        return of(null);
      }
      return this.crawlExecutionService.get({id: executionId, watch: false}).pipe(
        defaultIfEmpty(null),
        catchError(() => of(null)),
        startWith(null),
      );
    }),
    shareReplay({bufferSize: 1, refCount: true}),
  );

  ngOnChanges(): void {
    this.executionId$.next(this.executionId);
  }

}
