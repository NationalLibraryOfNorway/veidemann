import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output, inject } from '@angular/core';
import {RunStatus} from '../../../shared/models/controller';
import {CrawlerStatus} from '../../../shared/models/controller/controller.model';
import {AbilityServiceSignal} from "@casl/angular";
import {MongoAbility} from '@casl/ability';
import {MatIcon} from '@angular/material/icon';
import {DecimalPipe} from '@angular/common';
import {MatDividerModule} from '@angular/material/divider';


@Component({
  selector: 'app-crawlerstatus',
  templateUrl: './crawlerstatus.component.html',
  styleUrls: ['./crawlerstatus.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatIcon,
    MatDividerModule,
    DecimalPipe,
  ],
  standalone: true
})
export class CrawlerStatusComponent {
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  readonly RunStatus = RunStatus;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  constructor() {
    this.can = this.abilityService.can;
  }

  @Input()
  crawlerStatus: CrawlerStatus;

  @Input()
  showMetrics = true;

  @Input()
  showControl = true;

  @Output()
  changeRunStatus: EventEmitter<boolean> = new EventEmitter<boolean>();

  onPauseCrawler(shouldPause: boolean) {
    this.changeRunStatus.emit(shouldPause);
  }
}
