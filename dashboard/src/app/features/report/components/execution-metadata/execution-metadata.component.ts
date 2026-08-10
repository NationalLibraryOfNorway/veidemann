import {DatePipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {MatIcon} from '@angular/material/icon';

import {ExecutionStatePresentation} from '../../func';

export type ExecutionTerminalEvent = 'finished' | 'failed' | 'died' | 'aborted' | null;

@Component({
  selector: 'app-execution-metadata',
  templateUrl: './execution-metadata.component.html',
  styleUrls: ['./execution-metadata.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [DatePipe, MatIcon],
  standalone: true,
})
export class ExecutionMetadataComponent {
  @Input({required: true}) state: ExecutionStatePresentation;
  @Input() desiredState: ExecutionStatePresentation | null = null;
  @Input() startTime = '';
  @Input() endTime = '';
  @Input() terminalEvent: ExecutionTerminalEvent = null;

  sameCalendarDay(): boolean {
    const start = new Date(this.startTime);
    const end = new Date(this.endTime);
    return !Number.isNaN(start.getTime())
      && !Number.isNaN(end.getTime())
      && start.getFullYear() === end.getFullYear()
      && start.getMonth() === end.getMonth()
      && start.getDate() === end.getDate();
  }
}
