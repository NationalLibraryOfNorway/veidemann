import {CrawlExecutionState, JobExecutionState} from '../../../shared/models';

export type ExecutionStateTone = 'active' | 'waiting' | 'finished' | 'error' | 'neutral';

export interface ExecutionStatePresentation {
  icon: string;
  label: string;
  tone: ExecutionStateTone;
}

export function jobExecutionStatePresentation(state: JobExecutionState): ExecutionStatePresentation {
  switch (state) {
    case JobExecutionState.CREATED:
      return waiting($localize`:@@executionStateCreated:Created`);
    case JobExecutionState.RUNNING:
      return active($localize`:@@executionStateRunning:Running`);
    case JobExecutionState.FINISHED:
      return finished($localize`:@@executionStateFinished:Finished`);
    case JobExecutionState.ABORTED_MANUAL:
      return error($localize`:@@executionStateAborted:Aborted`);
    case JobExecutionState.FAILED:
      return error($localize`:@@executionStateFailed:Failed`);
    case JobExecutionState.DIED:
      return error($localize`:@@executionStateDied:Died`);
    default:
      return neutral($localize`:@@executionStateUndefined:Undefined`);
  }
}

export function crawlExecutionStatePresentation(state: CrawlExecutionState): ExecutionStatePresentation {
  switch (state) {
    case CrawlExecutionState.CREATED:
      return waiting($localize`:@@executionStateCreated:Created`);
    case CrawlExecutionState.FETCHING:
      return active($localize`:@@crawlExecutionStateFetching:Fetching`);
    case CrawlExecutionState.SLEEPING:
      return waiting($localize`:@@crawlExecutionStateSleeping:Sleeping`);
    case CrawlExecutionState.FINISHED:
      return finished($localize`:@@executionStateFinished:Finished`);
    case CrawlExecutionState.ABORTED_TIMEOUT:
      return error($localize`:@@crawlExecutionStateAbortedTimeout:Aborted (timeout)`);
    case CrawlExecutionState.ABORTED_SIZE:
      return error($localize`:@@crawlExecutionStateAbortedSize:Aborted (size)`);
    case CrawlExecutionState.ABORTED_MANUAL:
      return error($localize`:@@executionStateAborted:Aborted`);
    case CrawlExecutionState.FAILED:
      return error($localize`:@@executionStateFailed:Failed`);
    case CrawlExecutionState.DIED:
      return error($localize`:@@executionStateDied:Died`);
    default:
      return neutral($localize`:@@executionStateUndefined:Undefined`);
  }
}

function active(label: string): ExecutionStatePresentation {
  return {label, icon: 'progress_activity', tone: 'active'};
}

function waiting(label: string): ExecutionStatePresentation {
  return {label, icon: 'schedule', tone: 'waiting'};
}

function finished(label: string): ExecutionStatePresentation {
  return {label, icon: 'check_circle', tone: 'finished'};
}

function error(label: string): ExecutionStatePresentation {
  return {label, icon: 'error', tone: 'error'};
}

function neutral(label: string): ExecutionStatePresentation {
  return {label, icon: 'help', tone: 'neutral'};
}
