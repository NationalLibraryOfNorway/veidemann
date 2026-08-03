import {Component, EventEmitter, Input, Output, ChangeDetectionStrategy} from '@angular/core';
import {ConfigObject, Kind} from '../../../../../shared/models/config';
import {Params, RouterLink} from '@angular/router';
import {AbilityServiceSignal} from "@casl/angular";
import {MatListModule} from '@angular/material/list';
import {MatIcon} from '@angular/material/icon';
import {MatTooltip} from '@angular/material/tooltip';

@Component({
  selector: 'app-filter-shortcut',
  templateUrl: './filter-shortcut.component.html',
  imports: [
    MatIcon,
    MatListModule,
    MatTooltip,
    RouterLink,
  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: true
})
export class FilterShortcutComponent {
  readonly Kind = Kind;
  protected readonly can: AbilityServiceSignal<any>['can'];

  @Input()
  configObject: ConfigObject;

  @Output()
  clone = new EventEmitter();

  constructor(private abilityService: AbilityServiceSignal<any>) {
    this.can = this.abilityService.can;
  }

  getJobRefListQueryParams(configObject: ConfigObject): Params {
    return {crawl_job_id: configObject.seed.jobRefList.map(jobRef => jobRef.id)};
  }

  onClone() {
    this.clone.emit();
  }
}
