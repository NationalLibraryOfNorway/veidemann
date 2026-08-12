import { Component, EventEmitter, Input, Output, ChangeDetectionStrategy, inject } from '@angular/core';
import {ConfigObject, Kind} from '../../../../../shared/models/config';
import {Params, RouterLink} from '@angular/router';
import {AbilityServiceSignal} from "@casl/angular";
import {MongoAbility} from '@casl/ability';
import {MatListModule} from '@angular/material/list';
import {MatIcon} from '@angular/material/icon';
import {MatTooltip} from '@angular/material/tooltip';
import {configKindIcon} from '../../../func/config-kind-icon';

@Component({
  selector: 'app-filter-shortcut',
  templateUrl: './filter-shortcut.component.html',
  imports: [
    MatIcon,
    MatListModule,
    MatTooltip,
    RouterLink,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class FilterShortcutComponent {
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  readonly Kind = Kind;
  readonly configKindIcon = configKindIcon;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  @Input()
  configObject: ConfigObject;

  @Output()
  clone = new EventEmitter();

  constructor() {
    this.can = this.abilityService.can;
  }

  getJobRefListQueryParams(configObject: ConfigObject): Params {
    return {crawl_job_id: configObject.seed.jobRefList.map(jobRef => jobRef.id)};
  }

  onClone() {
    this.clone.emit();
  }
}
