import {formatDuration, isValidDate} from '../../../shared/func';
import {DurationNotice} from '../../../shared/components/execution-metrics/execution-metrics.component';
import {JobExecutionState} from '../../../shared/models';

// A few minutes over the limit is normal (e.g. while winding down), so only an overrun
// longer than this is reported.
export const MAX_DURATION_TOLERANCE_MS = 60 * 60 * 1000;

interface ExecutionTiming {
  state: JobExecutionState;
  startTime: string;
}

function readable(ms: number): string {
  return formatDuration(ms).replaceAll(':', ' ')
    .replace('days', ' d').replace('hours', ' h');
}

/**
 * Describes how far a created or running execution has run past the job's maximum duration
 * (the badge is the overrun, e.g. "+1 h 12min"), or null if it has not run past it by more than the tolerance. Ended executions are never
 * reported: the limit is not stored per execution, so the job's current limit says nothing
 * about the limit an earlier execution started with.
 */
export function maxDurationOverrun(
  execution: ExecutionTiming,
  maxDurationS: number | null | undefined,
  now = new Date(),
): DurationNotice | null {
  if (!maxDurationS || !execution.startTime) {
    return null;
  }
  if (execution.state !== JobExecutionState.CREATED && execution.state !== JobExecutionState.RUNNING) {
    return null;
  }
  const start = new Date(execution.startTime);
  if (!isValidDate(start)) {
    return null;
  }
  const overrunMs = now.getTime() - start.getTime() - maxDurationS * 1000;
  if (overrunMs <= MAX_DURATION_TOLERANCE_MS) {
    return null;
  }
  const overrun = readable(Math.floor(overrunMs / 60_000) * 60_000);
  const maxDuration = readable(maxDurationS * 1000);
  return {
    badge: `+${overrun}`,
    label: $localize`:@@jobExecutionRunningOverMaxDuration:Running ${overrun}:OVERRUN: over the maximum duration of ${maxDuration}:MAX_DURATION: set on the job`,
  };
}
