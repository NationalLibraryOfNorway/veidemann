import {Component, EventEmitter, Input, Output, ChangeDetectionStrategy} from '@angular/core';
import {CrawlExecutionState, CrawlExecutionStatus, Kind} from '../../../../shared/models';
import {Observable, Subject} from 'rxjs';
import {AbilityServiceSignal} from '@casl/angular';
import {MatListModule} from '@angular/material/list';
import {RouterLink} from '@angular/router';
import {MatIcon} from '@angular/material/icon';

@Component({
  selector: 'app-crawl-execution-shortcuts',
  templateUrl: './crawl-execution-shortcuts.component.html',
  styleUrls: ['./crawl-execution-shortcuts.component.css'],
  imports: [
    MatIcon,
    MatListModule,
    RouterLink,
  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: true
})
export class CrawlExecutionShortcutsComponent {
  readonly Kind = Kind;
  protected readonly can: AbilityServiceSignal<any>['can'];

  private reload$: Observable<void>;
  private reload: Subject<void>;

  @Input()
  crawlExecutionStatus: CrawlExecutionStatus;

  @Output()
  abortCrawlExecution = new EventEmitter<CrawlExecutionStatus>();

  constructor(private abilityService: AbilityServiceSignal<any>) {
    this.reload = new Subject<void>();
    this.reload$ = this.reload.asObservable();
    this.can = this.abilityService.can;
  }

  onAbortCrawlExecution(crawlExecutionStatus: CrawlExecutionStatus) {
    this.abortCrawlExecution.emit(crawlExecutionStatus);
  }

  canAbort(state: CrawlExecutionState) {
    return !CrawlExecutionStatus.DONE_STATES.includes(state);
  }
}
