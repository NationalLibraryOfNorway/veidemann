import {AsyncPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, EventEmitter, Input, OnChanges, Output, inject} from '@angular/core';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {MatChipsModule} from '@angular/material/chips';
import {MatIcon} from '@angular/material/icon';
import {MatTooltipModule} from '@angular/material/tooltip';
import {Observable, ReplaySubject, of} from 'rxjs';
import {catchError, defaultIfEmpty, distinctUntilChanged, map, shareReplay, startWith, switchMap} from 'rxjs/operators';

import {CrawlExecutionStatus} from '../../../../shared/models';
import {DetailOverflowComponent} from '../../../../shared/components';
import {CrawlExecutionService} from '../../services';
import {
  CrawlExecutionShortcutHelpersComponent,
} from '../crawl-execution-shortcuts/crawl-execution-shortcuts.component';

type LogListKind = 'pagelog' | 'crawllog';

interface LogListContext {
  executionId: string;
  status: CrawlExecutionStatus | null;
  seedLabel: string;
}

@Component({
  selector: 'app-log-list-shortcuts',
  templateUrl: './log-list-shortcuts.component.html',
  styleUrls: ['./log-list-shortcuts.component.scss'],
  imports: [
    AsyncPipe,
    CrawlExecutionShortcutHelpersComponent,
    DetailOverflowComponent,
    MatChipsModule,
    MatIcon,
    MatTooltipModule,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
})
export class LogListShortcutsComponent implements OnChanges {
  private readonly crawlExecutionService = inject(CrawlExecutionService);
  private readonly abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);
  private readonly executionId$ = new ReplaySubject<string>(1);

  @Input() executionId = '';
  @Input({required: true}) logKind: LogListKind;
  @Output() readonly removeExecutionFilter = new EventEmitter<void>();

  get actionsMenuLabel(): string {
    return this.logKind === 'crawllog'
      ? $localize`:@@crawlLogActionsMenuLabel:Crawl log actions`
      : $localize`:@@pageLogActionsMenuLabel:Page log actions`;
  }

  readonly context$: Observable<LogListContext | null> = this.executionId$.pipe(
    distinctUntilChanged(),
    switchMap(executionId => {
      if (!executionId) {
        return of(null);
      }
      const fallback: LogListContext = {executionId, status: null, seedLabel: ''};
      if (!this.abilityService.can('read', 'crawlexecution')) {
        return of(fallback);
      }
      return this.crawlExecutionService.get({id: executionId, watch: false}).pipe(
        switchMap(status => {
          const statusFallback: LogListContext = {
            executionId,
            status,
            seedLabel: '',
          };
          if (!status.seedId) {
            return of(statusFallback);
          }
          return this.crawlExecutionService.getSeed(status.seedId).pipe(
            map(seed => ({
              executionId,
              status,
              seedLabel: seed?.meta?.name || '',
            })),
            defaultIfEmpty(statusFallback),
            catchError(() => of(statusFallback)),
            startWith(statusFallback),
          );
        }),
        defaultIfEmpty(fallback),
        catchError(() => of(fallback)),
        startWith(fallback),
      );
    }),
    shareReplay({bufferSize: 1, refCount: true}),
  );

  ngOnChanges(): void {
    this.executionId$.next(this.executionId);
  }

  executionFilterTooltip(id: string): string {
    return $localize`:@@logListExecutionFilterTooltip:Crawl execution ID: ${id}:EXECUTION_ID:`;
  }

  removeExecutionFilterLabel(id: string): string {
    return $localize`:@@logListRemoveExecutionFilterLabel:Remove crawl execution ${id}:EXECUTION_ID: filter`;
  }
}
