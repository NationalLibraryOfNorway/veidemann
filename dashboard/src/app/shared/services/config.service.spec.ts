import {TestBed} from '@angular/core/testing';
import {EMPTY, of} from 'rxjs';

import {ListRequest} from '../../../api/config/v1/config_pb';
import {ConfigApiService} from '../../core';
import {ConfigQuery} from '../func';
import {BrowserScriptType, Kind} from '../models';
import {ConfigService} from './config.service';

describe('ConfigService BrowserScript filtering', () => {
  const list = vi.fn((request: ListRequest) => {
    void request;
    return EMPTY;
  });
  const count = vi.fn((request: ListRequest) => {
    void request;
    return of(0);
  });
  let service: ConfigService;

  beforeEach(() => {
    list.mockClear();
    count.mockClear();
    TestBed.configureTestingModule({
      providers: [
        ConfigService,
        {provide: ConfigApiService, useValue: {list, count}},
      ],
    });
    service = TestBed.inject(ConfigService);
  });

  it('uses the BrowserScript type field mask for searches and counts', () => {
    const query = browserScriptQuery(BrowserScriptType.ON_LOAD);

    service.search(query, {offset: 20, pageSize: 10}).subscribe();
    service.count(query).subscribe();

    for (const request of [list.mock.calls[0][0], count.mock.calls[0][0]]) {
      expect(request.queryMask?.paths).toEqual(['browserScript.browserScriptType']);
      expect(request.queryTemplate?.spec.case).toBe('browserScript');
      if (request.queryTemplate?.spec.case !== 'browserScript') {
        throw new Error('Expected a BrowserScript query template');
      }
      expect(request.queryTemplate.spec.value.browserScriptType).toBe(BrowserScriptType.ON_LOAD);
    }
  });

  it('omits the query template and field mask when no type is selected', () => {
    service.search(browserScriptQuery(null), {offset: 0, pageSize: 10}).subscribe();

    const request = list.mock.calls[0][0];
    expect(request.queryTemplate).toBeUndefined();
    expect(request.queryMask).toBeUndefined();
  });
});

function browserScriptQuery(browserScriptType: BrowserScriptType | null): ConfigQuery {
  return {
    kind: Kind.BROWSERSCRIPT,
    entityId: null,
    scheduleId: null,
    crawlConfigId: null,
    collectionId: null,
    browserConfigId: null,
    politenessId: null,
    disabled: null,
    browserScriptType,
    crawlJobIdList: [],
    scriptIdList: [],
    term: null,
    active: '',
    direction: '',
  };
}
