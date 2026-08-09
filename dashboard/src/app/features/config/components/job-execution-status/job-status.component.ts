import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {DatePipe} from '@angular/common';
import {MatButtonModule} from '@angular/material/button';
import {MatExpansionModule} from '@angular/material/expansion';
import {RouterLink} from '@angular/router';

import {ConfigObject, JobExecutionStatus} from '../../../../shared/models';
import {
  JobExecutionStatisticsComponent
} from '../../../report/components/job-execution-statistics/job-execution-statistics.component';
import {jobExecutionStatePresentation} from '../../../report/func';

@Component({
  selector: 'app-config-job-execution-status',
  templateUrl: './job-status.component.html',
  styleUrls: ['./job-status.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    DatePipe,
    JobExecutionStatisticsComponent,
    MatButtonModule,
    MatExpansionModule,
    RouterLink,
  ],
  standalone: true
})
export class JobStatusComponent {
  readonly statePresentation = jobExecutionStatePresentation;

  @Input({required: true})
  jobExecutionStatus: JobExecutionStatus;

  @Input({required: true})
  crawlJob: ConfigObject;

  @Input()
  canReadJobExecution = true;
}
