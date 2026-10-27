import {formatDuration, isValidDate} from '../../../shared/func';
import {JobExecutionState} from '../../../shared/models';

// Executions a few minutes over their limit are normal (e.g. while winding down), so
// only an overrun longer than this is reported.
export const MAX_DURATION_TOLERANCE_MS = 60 * 60 * 1000;

export interface MaxDurationOverrun {
  // Still created or running: the overrun is ongoing, otherwise the execution has ended.
  running: boolean;
  // The time beyond the maximum duration, e.g. "+1 h 12min".
  overrun: string;
  label: string;
}

interface ExecutionTiming {
  state: JobExecutionState;
  startTime: string;
  endTime: string;
}

function readable(ms: number): string {
  return formatDuration(ms).replaceAll(':', ' ')
    .replace('days', ' d').replace('hours', ' h');
}

/**
 * Describes how far an execution has run past the job's maximum duration, or null if it has
 * not run past it by more than the tolerance. Still running executions are measured against
 * `now`; executions that have ended without an end time cannot be measured.
 */
export function maxDurationOverrun(
  execution: ExecutionTiming,
  maxDurationS: number | null | undefined,
  now = new Date(),
): MaxDurationOverrun | null {
  if (!maxDurationS || !execution.startTime) {
    return null;
  }
  const running = execution.state === JobExecutionState.CREATED || execution.state === JobExecutionState.RUNNING;
  if (!running && !execution.endTime) {
    return null;
  }
  const start = new Date(execution.startTime);
  const end = execution.endTime ? new Date(execution.endTime) : now;
  if (!isValidDate(start) || !isValidDate(end)) {
    return null;
  }
  const overrunMs = end.getTime() - start.getTime() - maxDurationS * 1000;
  if (overrunMs <= MAX_DURATION_TOLERANCE_MS) {
    return null;
  }
  const overrun = readable(Math.floor(overrunMs / 60_000) * 60_000);
  const maxDuration = readable(maxDurationS * 1000);
  return {
    running,
    overrun: `+${overrun}`,
    label: running
      ? $localize`:@@jobExecutionRunningOverMaxDuration:Running ${overrun}:OVERRUN: over the maximum duration of ${maxDuration}:MAX_DURATION: set on the job`
      : $localize`:@@jobExecutionRanOverMaxDuration:Ran ${overrun}:OVERRUN: over the maximum duration of ${maxDuration}:MAX_DURATION: set on the job`,
  };
}
