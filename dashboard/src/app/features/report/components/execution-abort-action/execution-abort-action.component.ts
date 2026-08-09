import {ChangeDetectionStrategy, Component, EventEmitter, Input, Output, inject} from '@angular/core';
import {AbilityServiceSignal} from '@casl/angular';
import {MongoAbility} from '@casl/ability';
import {MatButtonModule} from '@angular/material/button';
import {MatIcon} from '@angular/material/icon';

@Component({
  selector: 'app-execution-abort-action',
  templateUrl: './execution-abort-action.component.html',
  styleUrls: ['./execution-abort-action.component.scss'],
  imports: [MatButtonModule, MatIcon],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
})
export class ExecutionAbortActionComponent {
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  @Input({required: true}) subject: 'jobexecution' | 'crawlexecution';
  @Input() abortable = false;
  @Input() inline = false;
  // The shared action API intentionally uses the domain verb requested by its containers.
  // eslint-disable-next-line @angular-eslint/no-output-native
  @Output() abort = new EventEmitter<void>();

  constructor() {
    this.can = this.abilityService.can;
  }
}
