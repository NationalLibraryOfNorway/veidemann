import {convertToParamMap} from '@angular/router';
import {BrowserScriptType, Kind} from '../../../shared/models';
import {configQueryFromParamMap, equalConfigCountQuery, equalConfigQuery} from './query';

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
      browserScriptType: null,
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

  it('parses operational BrowserScript types and rejects undefined or invalid values', () => {
    expect(configQueryFromParamMap(Kind.BROWSERSCRIPT, convertToParamMap({
      script_type: BrowserScriptType.ON_LOAD.toString(),
    })).browserScriptType).toBe(BrowserScriptType.ON_LOAD);

    for (const scriptType of ['', 'invalid', '2', '99', BrowserScriptType.UNDEFINED.toString()]) {
      expect(configQueryFromParamMap(Kind.BROWSERSCRIPT, convertToParamMap({
        script_type: scriptType,
      })).browserScriptType).toBeNull();
    }
  });

  it('treats BrowserScript type changes as new result and count queries', () => {
    const unfiltered = configQueryFromParamMap(Kind.BROWSERSCRIPT, convertToParamMap({}));
    const filtered = configQueryFromParamMap(Kind.BROWSERSCRIPT, convertToParamMap({
      script_type: BrowserScriptType.SCOPE_CHECK.toString(),
    }));

    expect(equalConfigQuery(unfiltered, filtered)).toBe(false);
    expect(equalConfigCountQuery(unfiltered, filtered)).toBe(false);
  });
});
