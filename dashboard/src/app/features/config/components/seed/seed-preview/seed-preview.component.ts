import {AsyncPipe, DatePipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, Input, inject} from '@angular/core';
import {MatButtonModule} from '@angular/material/button';
import {MatChipsModule} from '@angular/material/chips';
import {MatIcon} from '@angular/material/icon';
import {MatProgressBar} from '@angular/material/progress-bar';
import {MatTableModule} from '@angular/material/table';
import {MatTooltip} from '@angular/material/tooltip';
import {RouterLink} from '@angular/router';
import {Observable, of} from 'rxjs';
import {catchError, filter, map, toArray} from 'rxjs/operators';

import {ReportApiService} from '../../../../../core/api/report-api.service';
import {AuthService} from '../../../../../core/auth';
import {isEmojiLabel, LabelDisplayComponent} from '../../../../../shared/components';
import {CopyIdDirective} from '../../../../../shared/directives';
import {ConfigObject} from '../../../../../shared/models';
import {CrawlExecutionStatus} from '../../../../../shared/models/report';
import {ConfigService} from '../../../../../shared/services';
import {crawlExecutionStatePresentation} from '../../../../report/func';

const RECENT_CRAWL_EXECUTIONS = 3;

interface RecentCrawlExecutions {
  executions: CrawlExecutionStatus[];
  failed: boolean;
}

@Component({
  selector: 'app-seed-preview',
  templateUrl: './seed-preview.component.html',
  styleUrls: ['./seed-preview.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    AsyncPipe,
    CopyIdDirective,
    DatePipe,
    LabelDisplayComponent,
    MatButtonModule,
    MatChipsModule,
    MatIcon,
    MatProgressBar,
    MatTableModule,
    MatTooltip,
    RouterLink,
  ],
  standalone: true,
})
export class SeedPreviewComponent {
  private readonly configService = inject(ConfigService);
  private readonly reportService = inject(ReportApiService);
  protected readonly authService = inject(AuthService);
  protected readonly isEmojiLabel = isEmojiLabel;
  protected readonly statePresentation = crawlExecutionStatePresentation;
  readonly crawlExecutionColumns = ['crawlJob', 'startTime', 'endTime', 'state'];

  @Input() crawlJobs: ConfigObject[] = [];

  recentCrawlExecutions$: Observable<RecentCrawlExecutions> = of({executions: [], failed: false});
  entity$: Observable<ConfigObject | null> = of(null);
  private _seed: ConfigObject;

  @Input({required: true})
  set seed(seed: ConfigObject) {
    this._seed = seed;
    this.recentCrawlExecutions$ = this.reportService.getLastSeedStatus(seed.id, RECENT_CRAWL_EXECUTIONS).pipe(
      // The API streams one execution at a time; collect them into a single emission.
      filter(Boolean),
      toArray(),
      map(executions => ({executions, failed: false})),
      catchError(() => of({executions: [], failed: true})));
    const entityRef = seed.seed.entityRef;
    this.entity$ = entityRef?.id
      ? this.configService.get(entityRef, {suppressNotFound: true}).pipe(catchError(() => of(null)))
      : of(null);
  }

  get seed(): ConfigObject {
    return this._seed;
  }

  get jobIds(): string[] {
    return this.seed.seed.jobRefList.map(ref => ref.id);
  }

  crawlJobName(jobId: string): string {
    return this.crawlJobs?.find(job => job.id === jobId)?.meta.name?.trim() || jobId;
  }
}
