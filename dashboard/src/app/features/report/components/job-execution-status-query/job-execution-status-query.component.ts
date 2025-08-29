import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {JobExecutionStatusQuery} from '../../services';
import {ConfigObject} from '../../../../shared/models/config';
import {ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';
import {QueryComponent} from '../../../../shared/components';
import { JobExecutionState, jobExecutionStates } from '../../../../shared/models';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatSelectModule} from '@angular/material/select';
import {MatInputModule} from '@angular/material/input';
import {MatDatepickerModule} from '@angular/material/datepicker';
import {MatTimepickerModule} from '@angular/material/timepicker';
import {FlexLayoutModule} from '@angular/flex-layout';
import {MatCheckboxModule} from '@angular/material/checkbox';

@Component({
  selector: 'app-job-execution-status-query',
  templateUrl: './job-execution-status-query.component.html',
  styleUrls: ['./job-execution-status-query.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    FlexLayoutModule,
    MatCheckboxModule,
    MatDatepickerModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatTimepickerModule,
    ReactiveFormsModule
  ],
  standalone: true
})
export class JobExecutionStatusQueryComponent extends QueryComponent<JobExecutionStatusQuery> {
  readonly JobExecutionState = JobExecutionState;
  readonly jobExecutionStates = jobExecutionStates;

  @Input()
  crawlJobOptions: ConfigObject[];

  constructor(protected override fb: UntypedFormBuilder) {
    super(fb);
  }

  protected override createForm(): void {
    this.form = this.fb.group({
      stateList: null,
      jobId: '',
      startTimeFrom: '',
      startTimeTo: '',
      watch: {value: null, disabled: true},
    });
  }
}
