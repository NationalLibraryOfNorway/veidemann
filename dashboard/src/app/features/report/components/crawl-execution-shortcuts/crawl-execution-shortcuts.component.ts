import {Component, EventEmitter, Input, Output, ChangeDetectionStrategy} from '@angular/core';
import {CrawlExecutionState, CrawlExecutionStatus, Kind} from '../../../../shared/models';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {RouterLink} from '@angular/router';
import {MatIcon} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {MatMenuModule} from '@angular/material/menu';

@Component({
  selector: 'app-crawl-execution-shortcuts',
  templateUrl: './crawl-execution-shortcuts.component.html',
  styleUrls: ['../shortcut-actions.scss'],
  imports: [
    MatIcon,
    MatButtonModule,
    MatMenuModule,
    RouterLink,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class CrawlExecutionShortcutsComponent {
  readonly Kind = Kind;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  @Input()
  crawlExecutionStatus: CrawlExecutionStatus;

  @Output()
  abortCrawlExecution = new EventEmitter<CrawlExecutionStatus>();

  constructor(private abilityService: AbilityServiceSignal<MongoAbility>) {
    this.can = this.abilityService.can;
  }

  onAbortCrawlExecution(crawlExecutionStatus: CrawlExecutionStatus) {
    this.abortCrawlExecution.emit(crawlExecutionStatus);
  }

  canAbort(state: CrawlExecutionState) {
    return !CrawlExecutionStatus.DONE_STATES.includes(state);
  }
}
