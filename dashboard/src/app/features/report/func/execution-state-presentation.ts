import {CrawlExecutionState, JobExecutionState} from '../../../shared/models';

export type ExecutionStateTone = 'active' | 'waiting' | 'finished' | 'error' | 'neutral';

export interface ExecutionStatePresentation {
  icon: string;
  label: string;
  tone: ExecutionStateTone;
  lifecycle: 'active' | 'terminal' | 'undefined';
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
      return error($localize`:@@executionStateAbortedManually:Aborted manually`);
    case JobExecutionState.FAILED:
      return error($localize`:@@executionStateFailed:Failed`);
    case JobExecutionState.DIED:
      return error($localize`:@@executionStateDied:Died`);
    default:
      return neutral($localize`:@@executionStateEnded:Ended`);
  }
}

export function crawlExecutionStatePresentation(state: CrawlExecutionState): ExecutionStatePresentation {
  switch (state) {
    case CrawlExecutionState.CREATED:
      return waiting($localize`:@@executionStateCreated:Created`, 'schedule');
    case CrawlExecutionState.FETCHING:
      return active($localize`:@@crawlExecutionStateFetching:Fetching`, 'cloud_download');
    case CrawlExecutionState.SLEEPING:
      return waiting($localize`:@@crawlExecutionStateSleeping:Sleeping`, 'snooze');
    case CrawlExecutionState.FINISHED:
      return finished($localize`:@@executionStateFinished:Finished`, 'check_circle');
    case CrawlExecutionState.ABORTED_TIMEOUT:
      return error($localize`:@@crawlExecutionStateAbortedTimeout:Aborted after timeout`, 'timer_off');
    case CrawlExecutionState.ABORTED_SIZE:
      return error($localize`:@@crawlExecutionStateAbortedSize:Aborted at size limit`, 'data_usage');
    case CrawlExecutionState.ABORTED_MANUAL:
      return error($localize`:@@executionStateAbortedManually:Aborted manually`, 'cancel');
    case CrawlExecutionState.FAILED:
      return error($localize`:@@executionStateFailed:Failed`, 'error');
    case CrawlExecutionState.DIED:
      return error($localize`:@@executionStateDied:Died`, 'dangerous');
    default:
      return neutral($localize`:@@executionStateEnded:Ended`, 'help_outline');
  }
}

function active(label: string, icon = 'progress_activity'): ExecutionStatePresentation {
  return {label, icon, tone: 'active', lifecycle: 'active'};
}

function waiting(label: string, icon = 'schedule'): ExecutionStatePresentation {
  return {label, icon, tone: 'waiting', lifecycle: 'active'};
}

function finished(label: string, icon = 'check_circle'): ExecutionStatePresentation {
  return {label, icon, tone: 'finished', lifecycle: 'terminal'};
}

function error(label: string, icon = 'error'): ExecutionStatePresentation {
  return {label, icon, tone: 'error', lifecycle: 'terminal'};
}

function neutral(label: string, icon = 'help_outline'): ExecutionStatePresentation {
  return {label, icon, tone: 'neutral', lifecycle: 'undefined'};
}
