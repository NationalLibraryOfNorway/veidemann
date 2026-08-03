import {ChangeDetectionStrategy, Component, EventEmitter, Input, Output} from '@angular/core';
import {RunStatus} from '../../../shared/models/controller';
import {CrawlerStatus} from '../../../shared/models/controller/controller.model';
import {AbilityServiceSignal} from "@casl/angular";
import {MatCardModule} from '@angular/material/card';
import {MatIcon} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {LayoutDirective} from '@ngbracket/ngx-layout';
import {MatTooltip} from '@angular/material/tooltip';
import {LayoutGapDirective} from '@ngbracket/ngx-layout/flex';


@Component({
  selector: 'app-crawlerstatus',
  templateUrl: './crawlerstatus.component.html',
  styleUrls: ['./crawlerstatus.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    LayoutGapDirective,
    LayoutDirective,
    MatButtonModule,
    MatCardModule,
    MatIcon,
    MatTooltip,
  ],
  standalone: true
})
export class CrawlerStatusComponent {
  readonly RunStatus = RunStatus;
  protected readonly can: AbilityServiceSignal<any>['can'];

  constructor(private abilityService: AbilityServiceSignal<any>) {
    this.can = this.abilityService.can;
  }

  @Input()
  crawlerStatus: CrawlerStatus;

  @Output()
  changeRunStatus: EventEmitter<boolean> = new EventEmitter<boolean>();

  onPauseCrawler(shouldPause: boolean) {
    this.changeRunStatus.emit(shouldPause);
  }
}
