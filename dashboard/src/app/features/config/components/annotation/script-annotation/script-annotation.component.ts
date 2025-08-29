import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {Annotation} from '../../../../../shared/models/config';
import {Observable} from "rxjs";
import {AbilityService} from "@casl/angular";
import {AsyncPipe} from '@angular/common';
import {FlexLayoutModule} from '@angular/flex-layout';
import {MatChipsModule} from '@angular/material/chips';

@Component({
  selector: 'app-script-annotation',
  templateUrl: './script-annotation.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    AsyncPipe,
    FlexLayoutModule,
    MatChipsModule
  ],
  standalone: true
})
export class ScriptAnnotationComponent {
readonly ability$: Observable<any>;
  @Input()
  annotations: Annotation[];

  constructor(abilityService: AbilityService<any>) {
    this.ability$ = abilityService.ability$;
  }
}
