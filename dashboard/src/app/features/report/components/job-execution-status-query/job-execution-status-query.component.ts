import { ChangeDetectionStrategy,Component,Input } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatTimepickerModule } from '@angular/material/timepicker';
import { QueryComponent } from '../../../../shared/components';
import { JobExecutionState,jobExecutionStates } from '../../../../shared/models';
import { ConfigObject } from '../../../../shared/models/config';
import { JobExecutionStatusQuery } from '../../services';

@Component({
  selector: 'app-job-execution-status-query',
  templateUrl: './job-execution-status-query.component.html',
  styleUrls: ['./job-execution-status-query.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
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
