import {ChangeDetectionStrategy, Component, EventEmitter, Input, Output} from '@angular/core';
import {MatButtonToggleModule} from '@angular/material/button-toggle';
import {MatIcon} from '@angular/material/icon';
import {MatTooltip} from '@angular/material/tooltip';

@Component({
  selector: 'app-multi-update-operation',
  templateUrl: './multi-update-operation.component.html',
  styleUrls: ['./multi-update-operation.component.scss'],
  imports: [MatButtonToggleModule, MatIcon, MatTooltip],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
})
export class MultiUpdateOperationComponent {
  @Input({required: true}) heading = '';
  @Input({required: true}) addLabel = '';
  @Input({required: true}) removeLabel = '';
  @Input() mode: boolean | null | undefined = null;
  @Input() disabled = false;
  @Output() modeChange = new EventEmitter<boolean>();
}
