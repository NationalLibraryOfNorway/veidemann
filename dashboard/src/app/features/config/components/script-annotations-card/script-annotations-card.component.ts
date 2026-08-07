import {ChangeDetectionStrategy, Component, Input, inject} from '@angular/core';
import {MatCardModule} from '@angular/material/card';
import {MatIcon} from '@angular/material/icon';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';

import {Annotation, ConfigRef} from '../../../../shared/models';

export interface ScriptAnnotationContext {
  jobRef: ConfigRef;
  jobName: string;
  annotations: Annotation[];
  unavailable: boolean;
}

@Component({
  selector: 'app-script-annotations-card',
  templateUrl: './script-annotations-card.component.html',
  styleUrls: ['./script-annotations-card.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [MatCardModule, MatIcon],
  standalone: true,
})
export class ScriptAnnotationsCardComponent {
  private readonly abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);
  protected readonly can = this.abilityService.can;

  @Input({required: true}) contexts: ScriptAnnotationContext[] = [];
}
