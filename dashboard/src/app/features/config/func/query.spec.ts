import {convertToParamMap} from '@angular/router';
import {Kind} from '../../../shared/models';
import {configQueryFromParamMap, equalConfigQuery} from './query';

describe('configuration query route parsing', () => {
  it('parses defaults and repeated parameters atomically', () => {
    const query = configQueryFromParamMap(Kind.SEED, convertToParamMap({
      entity_id: 'entity',
      crawl_job_id: ['job-1', 'job-2'],
      disabled: 'false',
      sort: 'name:desc',
      s: '50',
      p: '2',
    }));

    expect(query).toEqual(expect.objectContaining({
      kind: Kind.SEED,
      entityId: 'entity',
      crawlJobIdList: ['job-1', 'job-2'],
      disabled: false,
      active: 'name',
      direction: 'desc',
    }));
    expect('pageSize' in query).toBe(false);
    expect('pageIndex' in query).toBe(false);
  });

  it('ignores legacy paging parameters and invalid sorting', () => {
    const query = configQueryFromParamMap(Kind.CRAWLJOB, convertToParamMap({
      sort: 'name',
      s: 'invalid',
      p: 'invalid',
    }));

    expect(query.active).toBe('');
    expect(query.direction).toBe('');
    expect('pageSize' in query).toBe(false);
    expect('pageIndex' in query).toBe(false);
    expect(query.disabled).toBeNull();
  });

  it('ignores unrelated parameters and repeated-value order for equality', () => {
    const previous = configQueryFromParamMap(Kind.SEED, convertToParamMap({
      crawl_job_id: ['job-1', 'job-2'],
      unrelated: 'first',
    }));
    const current = configQueryFromParamMap(Kind.SEED, convertToParamMap({
      crawl_job_id: ['job-2', 'job-1'],
      unrelated: 'second',
    }));

    expect(equalConfigQuery(previous, current)).toBe(true);
  });
});
