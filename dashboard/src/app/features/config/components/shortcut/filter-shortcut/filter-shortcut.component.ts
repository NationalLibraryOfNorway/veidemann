import {Component, EventEmitter, Input, Output} from '@angular/core';
import {ConfigObject, Kind} from '../../../../../shared/models/config';
import {Params, RouterLink} from '@angular/router';
import {Observable} from "rxjs";
import {AbilityService} from "@casl/angular";
import {AsyncPipe} from '@angular/common';
import {MatListModule} from '@angular/material/list';
import {MatIcon} from '@angular/material/icon';
import {MatTooltip} from '@angular/material/tooltip';

@Component({
  selector: 'app-filter-shortcut',
  templateUrl: './filter-shortcut.component.html',
  imports: [
    AsyncPipe,
    MatIcon,
    MatListModule,
    MatTooltip,
    RouterLink,
  ],
  standalone: true
})
export class FilterShortcutComponent {
  readonly Kind = Kind;
  readonly ability$: Observable<any>;

  @Input()
  configObject: ConfigObject;

  @Output()
  clone = new EventEmitter();

  constructor(private abilityService: AbilityService<any>) {
    this.ability$ = this.abilityService.ability$;
  }

  getJobRefListQueryParams(configObject: ConfigObject): Params {
    return {crawl_job_id: configObject.seed.jobRefList.map(jobRef => jobRef.id)};
  }

  onClone() {
    this.clone.emit();
  }
}
