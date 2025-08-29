import {Component, EventEmitter, Input, Output} from '@angular/core';
import {CrawlExecutionState, CrawlExecutionStatus, Kind} from '../../../../shared/models';
import {Observable, Subject} from 'rxjs';
import {AbilityService} from '@casl/angular';
import {AsyncPipe} from '@angular/common';
import {MatListModule} from '@angular/material/list';
import {RouterLink} from '@angular/router';
import {MatIcon} from '@angular/material/icon';

@Component({
  selector: 'app-crawl-execution-shortcuts',
  templateUrl: './crawl-execution-shortcuts.component.html',
  styleUrls: ['./crawl-execution-shortcuts.component.css'],
  imports: [
    AsyncPipe,
    MatIcon,
    MatListModule,
    RouterLink,
  ],
  standalone: true
})
export class CrawlExecutionShortcutsComponent {
  readonly Kind = Kind;
  readonly ability$: Observable<any>;

  private reload$: Observable<void>;
  private reload: Subject<void>;

  @Input()
  crawlExecutionStatus: CrawlExecutionStatus;

  @Output()
  abortCrawlExecution = new EventEmitter<CrawlExecutionStatus>();

  constructor(private abilityService: AbilityService<any>) {
    this.reload = new Subject<void>();
    this.reload$ = this.reload.asObservable();
    this.ability$ = this.abilityService.ability$;
  }

  onAbortCrawlExecution(crawlExecutionStatus: CrawlExecutionStatus) {
    this.abortCrawlExecution.emit(crawlExecutionStatus);
  }

  canAbort(state: CrawlExecutionState) {
    return !CrawlExecutionStatus.DONE_STATES.includes(state);
  }
}
