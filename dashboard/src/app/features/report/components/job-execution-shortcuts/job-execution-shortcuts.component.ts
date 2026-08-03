import {Component, EventEmitter, Input, Output, ChangeDetectionStrategy} from '@angular/core';
import {JobExecutionState, JobExecutionStatus, Kind} from '../../../../shared/models';
import {AbilityServiceSignal} from '@casl/angular';
import {RouterLink} from '@angular/router';
import {MatIcon} from '@angular/material/icon';
import {MatListModule} from '@angular/material/list';

@Component({
  selector: 'app-job-execution-shortcuts',
  templateUrl: './job-execution-shortcuts.component.html',
  styleUrls: ['./job-execution-shortcuts.component.css'],
  imports: [
    MatListModule,
    MatIcon,
    RouterLink
  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: true
})
export class JobExecutionShortcutsComponent {
  readonly Kind = Kind;
  readonly JobExecutionState = JobExecutionState;
  protected readonly can: AbilityServiceSignal<any>['can'];

  @Input() jobExecutionStatus: JobExecutionStatus;

  @Output()
  abortJobExecution = new EventEmitter<JobExecutionStatus>();

  constructor(private abilityService: AbilityServiceSignal<any>) {
    this.can = this.abilityService.can;
  }

  onAbortJobExecution(jobExecutionStatus: JobExecutionStatus) {
    this.abortJobExecution.emit(jobExecutionStatus);
  }
}
