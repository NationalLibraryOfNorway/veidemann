import {AsyncPipe} from '@angular/common';
import {Component, Input, ChangeDetectionStrategy, inject} from '@angular/core';
import {JobExecutionStatus, Kind} from '../../../../shared/models';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {RouterLink} from '@angular/router';
import {MatIcon} from '@angular/material/icon';
import {MatChipsModule} from '@angular/material/chips';
import {JobNamePipe} from '../../pipe/job-name.pipe';
import {CopyIdDirective} from '../../../../shared/directives';
import {MatTooltip} from '@angular/material/tooltip';

@Component({
  selector: 'app-job-execution-shortcut-helpers',
  templateUrl: './job-execution-shortcuts.component.html',
  styleUrls: ['../execution-shortcut-helpers.scss'],
  imports: [
    AsyncPipe,
    CopyIdDirective,
    JobNamePipe,
    MatChipsModule,
    MatIcon,
    RouterLink,
    MatTooltip,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class JobExecutionShortcutHelpersComponent {
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  readonly Kind = Kind;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  @Input() jobExecutionStatus: JobExecutionStatus;

  @Input() showJobExecution = false;
  @Input() showCrawlJob = true;
  @Input() showCrawlExecutions = true;

  constructor() {
    this.can = this.abilityService.can;
  }
}
