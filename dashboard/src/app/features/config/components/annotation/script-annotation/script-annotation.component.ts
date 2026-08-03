import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {Annotation} from '../../../../../shared/models/config';
import {AbilityServiceSignal} from "@casl/angular";
import {LayoutDirective} from '@ngbracket/ngx-layout';
import {MatChipsModule} from '@angular/material/chips';

@Component({
  selector: 'app-script-annotation',
  templateUrl: './script-annotation.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatChipsModule,
    LayoutDirective,
  ],
  standalone: true
})
export class ScriptAnnotationComponent {
  protected readonly can: AbilityServiceSignal<any>['can'];
  @Input()
  annotations: Annotation[];

  constructor(abilityService: AbilityServiceSignal<any>) {
    this.can = abilityService.can;
  }
}
