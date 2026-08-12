import {ChangeDetectionStrategy, Component, Input} from '@angular/core';

import {ConfigObject, JobExecutionStatus} from '../../../../shared/models';
import {
  JobExecutionMetricsSectionComponent
} from '../../../report/components/job-execution-metrics-section/job-execution-metrics-section.component';

@Component({
  selector: 'app-config-job-execution-status',
  templateUrl: './job-status.component.html',
  styleUrls: ['./job-status.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    JobExecutionMetricsSectionComponent,
  ],
  standalone: true
})
export class JobStatusComponent {
  @Input({required: true})
  jobExecutionStatus: JobExecutionStatus;

  @Input({required: true})
  crawlJob: ConfigObject;
}
