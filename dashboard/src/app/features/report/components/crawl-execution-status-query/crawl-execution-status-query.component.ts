import {AsyncPipe} from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  EventEmitter,
  inject,
  Input,
  Output,
  SimpleChanges,
} from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';
import {MatButtonModule} from '@angular/material/button';
import {MatChipsModule} from '@angular/material/chips';
import {MatDatepickerModule} from '@angular/material/datepicker';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatInputModule} from '@angular/material/input';
import {MatIcon} from '@angular/material/icon';
import {MatSelectModule} from '@angular/material/select';
import {MatTooltipModule} from '@angular/material/tooltip';
import {Observable, of, ReplaySubject} from 'rxjs';
import {catchError, defaultIfEmpty, distinctUntilChanged, map, shareReplay, switchMap} from 'rxjs/operators';

import {ConfigObject, CrawlExecutionState, crawlExecutionStates} from '../../../../shared/models';
import {CrawlExecutionService, CrawlExecutionStatusQuery, JobExecutionService} from '../../services';
import {StartTimeDateRangeQueryComponent} from '../start-time-date-range-query.component';
import {PollingRefreshButtonComponent} from '../../../../shared/components';

@Component({
  selector: 'app-crawl-execution-status-query',
  templateUrl: './crawl-execution-status-query.component.html',
  styleUrls: ['./crawl-execution-status-query.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    AsyncPipe,
    MatButtonModule,
    MatChipsModule,
    MatDatepickerModule,
    MatFormFieldModule,
    MatInputModule,
    MatIcon,
    MatSelectModule,
    MatTooltipModule,
    PollingRefreshButtonComponent,
    ReactiveFormsModule
  ]
})
export class CrawlExecutionStatusQueryComponent extends StartTimeDateRangeQueryComponent<CrawlExecutionStatusQuery> {

  private readonly crawlExecutionService = inject(CrawlExecutionService);
  private readonly jobExecutionService = inject(JobExecutionService);
  private readonly jobExecutionId$ = new ReplaySubject<string>(1);
  private readonly seedId$ = new ReplaySubject<string>(1);

  readonly CrawlExecutionState = CrawlExecutionState;
  readonly crawlExecutionStates = crawlExecutionStates.filter(
    state => state !== CrawlExecutionState.UNDEFINED
  );

  @Input()
  crawlJobOptions: ConfigObject[];

  @Output() readonly refresh = new EventEmitter<void>();

  readonly jobExecutionLabel$: Observable<string> = this.jobExecutionId$.pipe(
    distinctUntilChanged(),
    switchMap(id => id
      ? this.jobExecutionService.get({id, watch: false}).pipe(
        switchMap(status => status.jobId ? this.jobExecutionService.getJob(status.jobId) : of(null)),
        map(job => job?.meta?.name || ''),
        defaultIfEmpty(''),
        catchError(() => of('')),
      )
      : of('')),
    shareReplay({bufferSize: 1, refCount: true}),
  );

  readonly seedLabel$: Observable<string> = this.seedId$.pipe(
    distinctUntilChanged(),
    switchMap(id => id
      ? this.crawlExecutionService.getSeed(id).pipe(
        map(seed => seed?.meta?.name || ''),
        defaultIfEmpty(''),
        catchError(() => of('')),
      )
      : of('')),
    shareReplay({bufferSize: 1, refCount: true}),
  );

  override ngOnChanges(changes: SimpleChanges): void {
    super.ngOnChanges(changes);
    if (changes['query']) {
      this.jobExecutionId$.next(this.query?.jobExecutionId || '');
      this.seedId$.next(this.query?.seedId || '');
    }
  }

  removeDirectFilter(controlName: 'jobExecutionId' | 'seedId'): void {
    this.form.controls[controlName].setValue('');
  }

  toggleHasError(): void {
    const control = this.form.controls['hasError'];
    control.setValue(!control.value);
  }

  jobExecutionFilterTooltip(id: string): string {
    return $localize`:@@crawlExecutionJobExecutionFilterTooltip:Job execution ID: ${id}:JOB_EXECUTION_ID:`;
  }

  seedFilterTooltip(id: string): string {
    return $localize`:@@crawlExecutionSeedFilterTooltip:Seed ID: ${id}:SEED_ID:`;
  }

  removeJobExecutionFilterLabel(id: string): string {
    return $localize`:@@crawlExecutionRemoveJobExecutionFilterLabel:Remove job execution ${id}:JOB_EXECUTION_ID: filter`;
  }

  removeSeedFilterLabel(id: string): string {
    return $localize`:@@crawlExecutionRemoveSeedFilterLabel:Remove seed ${id}:SEED_ID: filter`;
  }

  protected override createForm(): void {
    this.form = this.fb.group({
      stateList: null,
      seedId: '',
      jobId: '',
      jobExecutionId: '',
      startTimeFrom: '',
      startTimeTo: '',
      hasError: null,
    });
  }
}
