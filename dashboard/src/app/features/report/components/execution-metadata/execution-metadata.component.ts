import {DatePipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {ExecutionStatePresentation} from '../../func';

@Component({
  selector: 'app-execution-metadata',
  templateUrl: './execution-metadata.component.html',
  styleUrls: ['./execution-metadata.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [DatePipe],
  standalone: true,
})
export class ExecutionMetadataComponent {
  @Input({required: true}) state: ExecutionStatePresentation;
  @Input() presentation: 'compact' | 'metrics' = 'compact';
  @Input() desiredState: ExecutionStatePresentation | null = null;
  @Input() startTime = '';
  @Input() endTime = '';
  @Input() contextLabel = '';
  @Input() contextValue = '';
  @Input() additionalContextLabel = '';
  @Input() additionalContextValue = '';

  get showDesiredStateInLifecycle(): boolean {
    return this.hasDistinctDesiredState
      && this.state.lifecycle === 'active'
      && !this.endTime;
  }

  private get hasDistinctDesiredState(): boolean {
    return !!this.desiredState
      && this.desiredState.lifecycle !== 'undefined'
      && this.desiredState.label !== this.state.label;
  }
}
