import {convertToParamMap} from '@angular/router';

import {JobExecutionState} from '../../../shared/models';
import {
  equalJobExecutionQuery,
  jobExecutionQueryFromParamMap,
  pageLogQueryFromParamMap,
  unknownPageLength
} from './query';

describe('report query route parsing', () => {
  it('parses paging, sorting, watch, dates, and numeric states', () => {
    const query = jobExecutionQueryFromParamMap(convertToParamMap({
      job_id: 'job',
      state: [`${JobExecutionState.RUNNING}`],
      start_time_from: '2024-01-01T00:00:00.000Z',
      sort: 'startTime:desc',
      watch: 'true',
      s: '50',
      p: '2',
    }));

    expect(query).toEqual(expect.objectContaining({
      jobId: 'job',
      stateList: [JobExecutionState.RUNNING],
      active: 'startTime',
      direction: 'desc',
      watch: true,
      pageSize: 50,
      pageIndex: 2,
    }));
  });

  it('ignores unrelated parameters and state order for equality', () => {
    const previous = jobExecutionQueryFromParamMap(convertToParamMap({state: ['1', '2'], unrelated: 'one'}));
    const current = jobExecutionQueryFromParamMap(convertToParamMap({state: ['2', '1'], unrelated: 'two'}));

    expect(equalJobExecutionQuery(previous, current)).toBe(true);
  });

  it('calculates an unknown total from the active page', () => {
    const query = pageLogQueryFromParamMap(convertToParamMap({s: '25', p: '2'}));

    expect(unknownPageLength(query, new Array(25))).toBe(76);
    expect(unknownPageLength(query, new Array(10))).toBe(60);
  });
});
