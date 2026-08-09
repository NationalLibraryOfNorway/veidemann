import {convertToParamMap} from '@angular/router';

import {JobExecutionState} from '../../../shared/models';
import {
  equalJobExecutionQuery,
  crawlLogQueryFromParamMap,
  jobExecutionQueryFromParamMap,
  pageLogQueryFromParamMap
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
    }));
    expect('pageSize' in query).toBe(false);
    expect('pageIndex' in query).toBe(false);
  });

  it('ignores unrelated parameters and state order for equality', () => {
    const previous = jobExecutionQueryFromParamMap(convertToParamMap({state: ['1', '2'], unrelated: 'one'}));
    const current = jobExecutionQueryFromParamMap(convertToParamMap({state: ['2', '1'], unrelated: 'two'}));

    expect(equalJobExecutionQuery(previous, current)).toBe(true);
  });

  it('accepts but ignores legacy paging parameters', () => {
    const query = pageLogQueryFromParamMap(convertToParamMap({s: '25', p: '2'}));
    expect('pageSize' in query).toBe(false);
    expect('pageIndex' in query).toBe(false);
  });

  it('ignores legacy watch parameters for page and crawl logs', () => {
    const params = convertToParamMap({watch: 'true'});

    expect('watch' in pageLogQueryFromParamMap(params)).toBe(false);
    expect('watch' in crawlLogQueryFromParamMap(params)).toBe(false);
  });
});
