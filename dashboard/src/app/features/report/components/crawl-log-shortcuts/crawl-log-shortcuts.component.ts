import {Component, Input, ChangeDetectionStrategy} from '@angular/core';
import {CrawlLog} from '../../../../shared/models';
import {AbilityServiceSignal} from '@casl/angular';
import {RouterLink} from '@angular/router';
import {MatListModule} from '@angular/material/list';
import {MatIcon} from '@angular/material/icon';

@Component({
  selector: 'app-crawl-log-shortcuts',
  templateUrl: './crawl-log-shortcuts.component.html',
  styleUrls: ['./crawl-log-shortcuts.component.css'],
  imports: [
    RouterLink,
    MatIcon,
    MatListModule,

  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: true
})
export class CrawlLogShortcutsComponent {
  protected readonly can: AbilityServiceSignal<any>['can'];

  @Input() crawlLog: CrawlLog;

  constructor(private abilityService: AbilityServiceSignal<any>) {
    this.can = this.abilityService.can;
  }
}
