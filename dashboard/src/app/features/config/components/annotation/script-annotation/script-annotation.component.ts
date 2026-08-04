import { ChangeDetectionStrategy, Component, Input, inject } from '@angular/core';
import {Annotation} from '../../../../../shared/models/config';
import {AbilityServiceSignal} from "@casl/angular";
import {MongoAbility} from '@casl/ability';
import {MatChipsModule} from '@angular/material/chips';

@Component({
  selector: 'app-script-annotation',
  templateUrl: './script-annotation.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatChipsModule,
  ],
  standalone: true
})
export class ScriptAnnotationComponent {
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];
  @Input()
  annotations: Annotation[];

  constructor() {
    const abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

    this.can = abilityService.can;
  }
}
