import {JobExecutionState} from '../../../shared/models';
import {maxDurationOverrun} from './job-max-duration';

describe('maxDurationOverrun', () => {
  const start = '2026-08-11T10:00:00.000Z';
  const running = {state: JobExecutionState.RUNNING, startTime: start};

  it('measures running executions against now', () => {
    const now = new Date('2026-08-11T14:05:00.000Z');
    expect(maxDurationOverrun(running, 7200, now)).toEqual({
      badge: '+2 h 5min',
      label: 'Running 2 h 5min over the maximum duration of 2 h set on the job',
    });
  });

  it('also reports created executions', () => {
    const now = new Date('2026-08-11T14:05:00.000Z');
    expect(maxDurationOverrun({...running, state: JobExecutionState.CREATED}, 7200, now)).not.toBeNull();
  });

  it('ignores overruns within the one hour tolerance', () => {
    expect(maxDurationOverrun(running, 1800, new Date('2026-08-11T10:30:00.000Z'))).toBeNull();
    expect(maxDurationOverrun(running, 1800, new Date('2026-08-11T11:30:00.000Z'))).toBeNull();
    expect(maxDurationOverrun(running, 1800, new Date('2026-08-11T11:31:00.000Z'))).not.toBeNull();
  });

  it('never reports ended executions, as the limit they started with is not known', () => {
    const now = new Date('2026-08-20T00:00:00.000Z');
    [JobExecutionState.FINISHED, JobExecutionState.FAILED, JobExecutionState.ABORTED_MANUAL, JobExecutionState.DIED]
      .forEach(state => expect(maxDurationOverrun({state, startTime: start}, 60, now)).toBeNull());
  });

  it('ignores jobs without a limit and executions without a valid start', () => {
    const now = new Date('2026-08-20T00:00:00.000Z');
    expect(maxDurationOverrun(running, 0, now)).toBeNull();
    expect(maxDurationOverrun(running, null, now)).toBeNull();
    expect(maxDurationOverrun({state: JobExecutionState.RUNNING, startTime: ''}, 60, now)).toBeNull();
    expect(maxDurationOverrun({state: JobExecutionState.RUNNING, startTime: 'nope'}, 60, now)).toBeNull();
  });
});
