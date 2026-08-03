import { Component, EventEmitter, Input, Output, ChangeDetectionStrategy, inject } from '@angular/core';
import {JobExecutionState, JobExecutionStatus, Kind} from '../../../../shared/models';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {RouterLink} from '@angular/router';
import {MatIcon} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {MatMenuModule} from '@angular/material/menu';

@Component({
  selector: 'app-job-execution-shortcuts',
  templateUrl: './job-execution-shortcuts.component.html',
  styleUrls: ['../shortcut-actions.scss'],
  imports: [
    MatButtonModule,
    MatMenuModule,
    MatIcon,
    RouterLink
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class JobExecutionShortcutsComponent {
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  readonly Kind = Kind;
  readonly JobExecutionState = JobExecutionState;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  @Input() jobExecutionStatus: JobExecutionStatus;

  @Output()
  abortJobExecution = new EventEmitter<JobExecutionStatus>();

  constructor() {
    this.can = this.abilityService.can;
  }

  onAbortJobExecution(jobExecutionStatus: JobExecutionStatus) {
    this.abortJobExecution.emit(jobExecutionStatus);
  }
}
