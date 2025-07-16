import {Component, Input} from '@angular/core';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {AbilityService} from "@casl/angular";
import { PureAbility } from '@casl/ability';
import { Observable } from 'rxjs';

@Component({
    selector: 'app-shortcut-list',
    templateUrl: './shortcut-list.component.html',
    standalone: false
})
export class ShortcutListComponent {
  readonly Kind = Kind;
  readonly ability$: Observable<PureAbility>;

  @Input()
  configObject: ConfigObject;

  constructor(private abilityService: AbilityService<any>) {
    this.ability$ = this.abilityService.ability$;
  }
}
