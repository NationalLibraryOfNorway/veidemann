import {ChangeDetectionStrategy, Component, EventEmitter, Input, Output} from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';
import {MatButtonModule} from '@angular/material/button';
import {MatChipsModule} from '@angular/material/chips';
import {MatDatepickerModule} from '@angular/material/datepicker';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatInputModule} from '@angular/material/input';
import {MatSelectModule} from '@angular/material/select';
import {JobExecutionState, jobExecutionStates} from '../../../../shared/models';
import {ConfigObject} from '../../../../shared/models/config';
import {JobExecutionStatusQuery} from '../../services';
import {StartTimeDateRangeQueryComponent} from '../start-time-date-range-query.component';
import {PollingRefreshButtonComponent} from '../../../../shared/components';

@Component({
  selector: 'app-job-execution-status-query',
  templateUrl: './job-execution-status-query.component.html',
  styleUrls: ['./job-execution-status-query.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatButtonModule,
    MatChipsModule,
    MatDatepickerModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    PollingRefreshButtonComponent,
    ReactiveFormsModule
  ],
  standalone: true
})
export class JobExecutionStatusQueryComponent extends StartTimeDateRangeQueryComponent<JobExecutionStatusQuery> {

  readonly JobExecutionState = JobExecutionState;
  readonly jobExecutionStates = jobExecutionStates.filter(
    state => state !== JobExecutionState.UNDEFINED
  );

  @Input()
  crawlJobOptions: ConfigObject[];

  @Output() readonly refresh = new EventEmitter<void>();

  protected override createForm(): void {
    this.form = this.fb.group({
      stateList: null,
      jobId: '',
      startTimeFrom: '',
      startTimeTo: '',
    });
  }

}
