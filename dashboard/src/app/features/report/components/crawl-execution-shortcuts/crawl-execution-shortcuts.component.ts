import {AsyncPipe} from '@angular/common';
import {Component, Input, ChangeDetectionStrategy, inject} from '@angular/core';
import {CrawlExecutionState, CrawlExecutionStatus, Kind} from '../../../../shared/models';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {RouterLink} from '@angular/router';
import {MatIcon} from '@angular/material/icon';
import {MatChipsModule} from '@angular/material/chips';
import {catchError, defaultIfEmpty, map, of, ReplaySubject, shareReplay, switchMap} from 'rxjs';
import {AppConfig} from '../../../../app.config';
import {CrawlExecutionService} from '../../services';

const PLAYBACK_STATES = new Set([
  CrawlExecutionState.FINISHED,
  CrawlExecutionState.ABORTED_TIMEOUT,
  CrawlExecutionState.ABORTED_SIZE,
  CrawlExecutionState.ABORTED_MANUAL,
  CrawlExecutionState.FAILED,
]);

export function buildPlaybackUrl(baseUrl: string, startTime: string, seedUrl: string): string {
  const normalizedBaseUrl = normalizePlaybackBaseUrl(baseUrl);
  if (!normalizedBaseUrl || !isHttpUrl(seedUrl)) {
    return '';
  }

  const start = new Date(startTime);
  if (Number.isNaN(start.getTime())) {
    return '';
  }

  const timestamp = start.toISOString().replace(/\D/g, '').slice(0, 14);
  return `${normalizedBaseUrl}/${timestamp}/${seedUrl}`;
}

function normalizePlaybackBaseUrl(baseUrl: string): string {
  const value = baseUrl?.trim().replace(/\/+$/, '');
  if (!value) {
    return '';
  }
  if (value.startsWith('/') && !value.startsWith('//')) {
    return value;
  }
  return isHttpUrl(value) ? value : '';
}

function isHttpUrl(value: string): boolean {
  try {
    const url = new URL(value);
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
}

@Component({
  selector: 'app-crawl-execution-shortcut-helpers',
  templateUrl: './crawl-execution-shortcuts.component.html',
  styleUrls: ['../execution-shortcut-helpers.scss'],
  imports: [
    AsyncPipe,
    MatIcon,
    MatChipsModule,
    RouterLink,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class CrawlExecutionShortcutHelpersComponent {
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);
  private appConfig = inject(AppConfig);
  private crawlExecutionService = inject(CrawlExecutionService);
  private crawlExecutionStatus$ = new ReplaySubject<CrawlExecutionStatus>(1);
  private _crawlExecutionStatus: CrawlExecutionStatus;

  readonly Kind = Kind;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  @Input()
  set crawlExecutionStatus(value: CrawlExecutionStatus) {
    this._crawlExecutionStatus = value;
    this.crawlExecutionStatus$.next(value);
  }

  get crawlExecutionStatus(): CrawlExecutionStatus {
    return this._crawlExecutionStatus;
  }

  @Input() showPageLog = true;
  @Input() showCrawlLog = true;
  @Input() showCrawlExecution = false;
  @Input() showJobExecution = true;
  @Input() showCrawlJob = true;
  @Input() showSeed = true;
  @Input() showPlayback = false;

  readonly playbackUrl$ = this.crawlExecutionStatus$.pipe(
    switchMap(status => {
      if (!PLAYBACK_STATES.has(status.state) || !this.appConfig.playbackBaseUrl) {
        return of('');
      }
      return this.crawlExecutionService.getSeed(status.seedId).pipe(
        map(seed => buildPlaybackUrl(
          this.appConfig.playbackBaseUrl,
          status.startTime,
          seed?.meta?.name ?? '',
        )),
        defaultIfEmpty(''),
        catchError(() => of('')),
      );
    }),
    shareReplay({bufferSize: 1, refCount: true}),
  );

  constructor() {
    this.can = this.abilityService.can;
  }
}
