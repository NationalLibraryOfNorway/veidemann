import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {JobExecutionState, JobExecutionStatus} from '../../../../shared/models/report';
import {DatePipe} from '@angular/common';
import {MatExpansionModule} from '@angular/material/expansion';
import {MatTableModule} from '@angular/material/table';
import {LayoutDirective} from '@ngbracket/ngx-layout';

@Component({
  selector: 'app-config-job-execution-status',
  templateUrl: './job-status.component.html',
  styleUrls: ['./job-status.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    DatePipe,
    LayoutDirective,
    MatExpansionModule,
    MatTableModule
  ],
  standalone: true
})
export class JobStatusComponent {
  readonly JobExecutionState = JobExecutionState;

  @Input()
  jobExecutionStatus: JobExecutionStatus;

  displayedColumns: string[] = ['state', 'count'];

  getExecMap(executionStateMap: Map<string, number>) {
    const datasource = [];
    for (const [key, value] of executionStateMap) {
      datasource.push({key, value});
    }
    return datasource;
  }
}
