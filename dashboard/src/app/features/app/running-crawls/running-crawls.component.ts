import {AsyncPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, EventEmitter, Input, Output} from '@angular/core';
import {MatChipSelectionChange, MatChipsModule} from '@angular/material/chips';
import {Sort, SortDirection} from '@angular/material/sort';

import {
  jobExecutionStates,
  JobExecutionState,
  JobExecutionStatus,
  ListDataSource,
} from '../../../shared/models';
import {JobExecutionStatusListComponent} from '../../report/components';
import {JobExecutionStatusQuery} from '../../report/services';
import {ExecutionQueueCounts} from '../../report/services';
import {PollingRefreshButtonComponent} from '../../../shared/components';

interface StateChip {
  label: string;
  states: readonly JobExecutionState[];
}

@Component({
  selector: 'app-running-crawls',
  templateUrl: './running-crawls.component.html',
  styleUrls: ['./running-crawls.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    AsyncPipe,
    JobExecutionStatusListComponent,
    MatChipsModule,
    PollingRefreshButtonComponent,
  ],
})
export class RunningCrawlsComponent {
  private readonly abortedStates = jobExecutionStates.filter(
    state => JobExecutionState[state].startsWith('ABORTED_')
  );

  readonly stateChips: readonly StateChip[] = [
    {label: 'RUNNING', states: [JobExecutionState.RUNNING]},
    {label: 'FINISHED', states: [JobExecutionState.FINISHED]},
    {label: 'ABORTED', states: this.abortedStates},
    {label: 'FAILED', states: [JobExecutionState.FAILED]},
  ];

  @Input({required: true})
  dataSource: ListDataSource<JobExecutionStatus, JobExecutionStatusQuery>;

  @Input({required: true})
  selectedStates: readonly JobExecutionState[] = [JobExecutionState.RUNNING];

  @Input()
  queueCounts: ExecutionQueueCounts = new Map();

  @Input()
  sortActive = 'startTime';

  @Input()
  sortDirection: SortDirection = 'desc';

  @Output() readonly selectedStatesChange = new EventEmitter<readonly JobExecutionState[]>();
  @Output() readonly rowClick = new EventEmitter<JobExecutionStatus>();
  @Output() readonly refresh = new EventEmitter<void>();
  @Output() readonly sort = new EventEmitter<Sort>();

  isChipSelected(chip: StateChip): boolean {
    return chip.states.every(state => this.selectedStates.includes(state));
  }

  onChipSelectionChange(chip: StateChip, event: MatChipSelectionChange): void {
    if (!event.isUserInput) {
      return;
    }
    const nextStates = new Set(this.selectedStates);
    for (const state of chip.states) {
      if (event.selected) {
        nextStates.add(state);
      } else {
        nextStates.delete(state);
      }
    }
    this.selectedStatesChange.emit([...nextStates]);
  }
}
