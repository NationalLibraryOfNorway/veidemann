import {ChangeDetectionStrategy, Component, EventEmitter, Input, Output} from '@angular/core';
import {RunStatus} from '../../../shared/models/controller';
import {CrawlerStatus} from '../../../shared/models/controller/controller.model';
import {Observable} from "rxjs";
import {AbilityService} from "@casl/angular";
import {AsyncPipe} from '@angular/common';
import {MatCardModule} from '@angular/material/card';
import {MatIcon} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {FlexLayoutModule} from '@ngbracket/ngx-layout';
import {MatTooltip} from '@angular/material/tooltip';


@Component({
  selector: 'app-crawlerstatus',
  templateUrl: './crawlerstatus.component.html',
  styleUrls: ['./crawlerstatus.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    AsyncPipe,
    FlexLayoutModule,
    MatButtonModule,
    MatCardModule,
    MatIcon,
    MatTooltip,
  ],
  standalone: true
})
export class CrawlerStatusComponent {
  readonly RunStatus = RunStatus;
  readonly ability$: Observable<any>;

  constructor(private abilityService: AbilityService<any>) {
    this.ability$ = this.abilityService.ability$;
  }

  @Input()
  crawlerStatus: CrawlerStatus;

  @Output()
  changeRunStatus: EventEmitter<boolean> = new EventEmitter<boolean>();

  onPauseCrawler(shouldPause: boolean) {
    this.changeRunStatus.emit(shouldPause);
  }
}
