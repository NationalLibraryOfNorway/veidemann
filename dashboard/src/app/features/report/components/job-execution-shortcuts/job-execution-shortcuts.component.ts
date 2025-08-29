import {Component, EventEmitter, Input, Output} from '@angular/core';
import {JobExecutionState, JobExecutionStatus, Kind} from '../../../../shared/models';
import {Observable} from 'rxjs';
import {AbilityService} from '@casl/angular';
import {AsyncPipe} from '@angular/common';
import {RouterLink} from '@angular/router';
import {MatIcon} from '@angular/material/icon';
import {MatListModule} from '@angular/material/list';

@Component({
  selector: 'app-job-execution-shortcuts',
  templateUrl: './job-execution-shortcuts.component.html',
  styleUrls: ['./job-execution-shortcuts.component.css'],
  imports: [
    AsyncPipe,
    MatListModule,
    MatIcon,
    RouterLink
  ],
  standalone: true
})
export class JobExecutionShortcutsComponent {
  readonly Kind = Kind;
  readonly JobExecutionState = JobExecutionState;
  readonly ability$: Observable<any>;

  @Input() jobExecutionStatus: JobExecutionStatus;

  @Output()
  abortJobExecution = new EventEmitter<JobExecutionStatus>();

  constructor(private abilityService: AbilityService<any>) {
    this.ability$ = this.abilityService.ability$;
  }

  onAbortJobExecution(jobExecutionStatus: JobExecutionStatus) {
    this.abortJobExecution.emit(jobExecutionStatus);
  }
}
