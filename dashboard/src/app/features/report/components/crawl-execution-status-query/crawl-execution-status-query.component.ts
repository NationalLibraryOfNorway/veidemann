import {ChangeDetectionStrategy, Component, EventEmitter, Input, Output} from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';
import {MatButtonModule} from '@angular/material/button';
import {MatChipsModule} from '@angular/material/chips';
import {MatDatepickerModule} from '@angular/material/datepicker';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatInputModule} from '@angular/material/input';
import {MatSelectModule} from '@angular/material/select';

import {ConfigObject, CrawlExecutionState, crawlExecutionStates} from '../../../../shared/models';
import {CrawlExecutionStatusQuery} from '../../services';
import {StartTimeDateRangeQueryComponent} from '../start-time-date-range-query.component';
import {PollingRefreshButtonComponent} from '../../../../shared/components';

@Component({
  selector: 'app-crawl-execution-status-query',
  templateUrl: './crawl-execution-status-query.component.html',
  styleUrls: ['./crawl-execution-status-query.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    MatButtonModule,
    MatChipsModule,
    MatDatepickerModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    PollingRefreshButtonComponent,
    ReactiveFormsModule
  ]
})
export class CrawlExecutionStatusQueryComponent extends StartTimeDateRangeQueryComponent<CrawlExecutionStatusQuery> {

  readonly CrawlExecutionState = CrawlExecutionState;
  readonly crawlExecutionStates = crawlExecutionStates.filter(
    state => state !== CrawlExecutionState.UNDEFINED
  );

  @Input()
  crawlJobOptions: ConfigObject[];

  @Output() readonly refresh = new EventEmitter<void>();

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
