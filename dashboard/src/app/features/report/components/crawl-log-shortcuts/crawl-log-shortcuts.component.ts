import {Component, Input} from '@angular/core';
import {CrawlLog} from '../../../../shared/models';
import {Observable} from 'rxjs';
import {AbilityService} from '@casl/angular';
import {AsyncPipe} from '@angular/common';
import {RouterLink} from '@angular/router';
import {MatListModule} from '@angular/material/list';
import {MatIcon} from '@angular/material/icon';

@Component({
  selector: 'app-crawl-log-shortcuts',
  templateUrl: './crawl-log-shortcuts.component.html',
  styleUrls: ['./crawl-log-shortcuts.component.css'],
  imports: [
    AsyncPipe,
    RouterLink,
    MatIcon,
    MatListModule,

  ],
  standalone: true
})
export class CrawlLogShortcutsComponent {
  readonly ability$: Observable<any>

  @Input() crawlLog: CrawlLog;

  constructor(private abilityService: AbilityService<any>) {
    this.ability$ = this.abilityService.ability$;
  }
}
