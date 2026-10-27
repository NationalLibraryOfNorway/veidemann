import {JobExecutionState} from '../../../shared/models';
import {maxDurationOverrun} from './job-max-duration';

describe('maxDurationOverrun', () => {
  const start = '2026-08-11T10:00:00.000Z';
  const ended = (endTime: string, state = JobExecutionState.FINISHED) => ({state, startTime: start, endTime});
  const running = {state: JobExecutionState.RUNNING, startTime: start, endTime: ''};

  it('reports an ended execution that ran more than an hour over the limit', () => {
    expect(maxDurationOverrun(ended('2026-08-11T13:12:40.000Z'), 7200)).toEqual({
      running: false,
      overrun: '+1 h 12min',
      label: 'Ran 1 h 12min over the maximum duration of 2 h set on the job',
    });
  });

  it('ignores overruns within the one hour tolerance', () => {
    expect(maxDurationOverrun(ended('2026-08-11T10:30:00.000Z'), 1800)).toBeNull();
    expect(maxDurationOverrun(ended('2026-08-11T11:30:00.000Z'), 1800)).toBeNull();
    expect(maxDurationOverrun(ended('2026-08-11T11:31:00.000Z'), 1800)).not.toBeNull();
  });

  it('measures created and running executions against now', () => {
    const now = new Date('2026-08-11T14:05:00.000Z');
    expect(maxDurationOverrun(running, 7200, now)).toEqual({
      running: true,
      overrun: '+2 h 5min',
      label: 'Running 2 h 5min over the maximum duration of 2 h set on the job',
    });
    expect(maxDurationOverrun({...running, state: JobExecutionState.CREATED}, 7200, now)?.running).toBe(true);
    expect(maxDurationOverrun(running, 7200, new Date('2026-08-11T11:00:00.000Z'))).toBeNull();
  });

  it('cannot measure ended executions without an end time, nor jobs without a limit', () => {
    expect(maxDurationOverrun({...running, state: JobExecutionState.FAILED}, 60)).toBeNull();
    expect(maxDurationOverrun(ended('2026-08-20T00:00:00.000Z'), 0)).toBeNull();
    expect(maxDurationOverrun(ended('2026-08-20T00:00:00.000Z'), null)).toBeNull();
  });

  it('ignores executions without a valid start or end', () => {
    expect(maxDurationOverrun({state: JobExecutionState.RUNNING, startTime: '', endTime: ''}, 60)).toBeNull();
    expect(maxDurationOverrun({state: JobExecutionState.RUNNING, startTime: 'nope', endTime: ''}, 60)).toBeNull();
    expect(maxDurationOverrun(ended('nope'), 60)).toBeNull();
  });
});
