import {TestBed} from '@angular/core/testing';
import {EMPTY, of} from 'rxjs';

import {ListRequest, UpdateRequest} from '../../../api/config/v1/config_pb';
import {ConfigApiService} from '../../core';
import {ConfigQuery} from '../func';
import {BrowserScriptType, ConfigObject, Kind, Label, Role, robotsPolicies, RobotsPolicy} from '../models';
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
  const startUpdate = vi.fn((request: UpdateRequest) => {
    void request;
    return of('task-123');
  });
  let service: ConfigService;

  beforeEach(() => {
    list.mockClear();
    count.mockClear();
    startUpdate.mockClear();
    TestBed.configureTestingModule({
      providers: [
        ConfigService,
        {provide: ConfigApiService, useValue: {list, count, startUpdate}},
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

  it.each(robotsPolicies)('uses the generic robots-policy field mask for value %s in searches and counts', policy => {
    const query = politenessQuery(policy);
    service.search(query, {offset: 0, pageSize: 10}).subscribe();
    service.count(query).subscribe();

    for (const request of [list.mock.calls[0][0], count.mock.calls[0][0]]) {
      expect(request.queryMask?.paths).toEqual(['politenessConfig.robotsPolicy']);
      expect(request.queryTemplate?.spec.case).toBe('politenessConfig');
      if (request.queryTemplate?.spec.case !== 'politenessConfig') throw new Error('Expected politeness query');
      expect(request.queryTemplate.spec.value.robotsPolicy).toBe(policy);
    }
  });

  it('omits the politeness query field mask after clearing the policy', () => {
    service.search(politenessQuery(null), {offset: 0, pageSize: 10}).subscribe();
    expect(list.mock.calls[0][0].queryTemplate).toBeUndefined();
    expect(list.mock.calls[0][0].queryMask).toBeUndefined();
  });

  it('uses the repeated role field mask for role mapping searches and counts', () => {
    const query = {...browserScriptQuery(null), kind: Kind.ROLEMAPPING, role: Role.CURATOR};

    service.search(query, {offset: 0, pageSize: 10}).subscribe();
    service.count(query).subscribe();

    for (const request of [list.mock.calls[0][0], count.mock.calls[0][0]]) {
      expect(request.queryMask?.paths).toEqual(['roleMapping.role']);
      expect(request.queryTemplate?.spec.case).toBe('roleMapping');
      if (request.queryTemplate?.spec.case !== 'roleMapping') throw new Error('Expected role mapping query');
      expect(request.queryTemplate.spec.value.role).toEqual([Role.CURATOR]);
    }
  });

  it('preserves label selectors whose key contains the label marker text', () => {
    const query = {
      ...browserScriptQuery(null),
      kind: Kind.CRAWLENTITY,
      term: 'label:thisisalabel:hei',
    };

    service.search(query, {offset: 0, pageSize: 10}).subscribe();
    service.count(query).subscribe();

    expect(list.mock.calls[0][0].labelSelector).toEqual(['thisisalabel:hei']);
    expect(count.mock.calls[0][0].labelSelector).toEqual(['thisisalabel:hei']);
  });

  it('orders configuration lists by indexed metadata fields', () => {
    service.search({
      ...browserScriptQuery(null),
      active: 'lastModified',
      direction: 'desc',
    }, {offset: 0, pageSize: 10}).subscribe();

    expect(list.mock.calls[0][0]).toEqual(expect.objectContaining({
      orderByPath: 'meta.lastModified',
      orderDescending: true,
    }));
  });

  it('starts an all-database update with the compact active filter', () => {
    const query: ConfigQuery = {
      ...browserScriptQuery(null),
      kind: Kind.SEED,
      crawlJobIdList: ['job-1'],
      term: 'label:status:📦',
    };
    service.search(query, {offset: 200, pageSize: 100}).subscribe();
    const updateTemplate = new ConfigObject({kind: Kind.SEED});
    updateTemplate.meta.labelList = [new Label({key: 'status', value: '📦'})];

    service.startUpdateWithTemplate(updateTemplate, ['meta.label-']).subscribe();

    expect(startUpdate).toHaveBeenCalledOnce();
    const request = startUpdate.mock.calls[0][0];
    expect(request.listRequest).toEqual(expect.objectContaining({
      kind: Kind.SEED,
      id: [],
      labelSelector: ['status:📦'],
      pageSize: 0,
      offset: 0,
    }));
    expect(request.listRequest?.queryMask?.paths).toEqual(['seed.jobRef']);
    expect(request.listRequest?.queryTemplate?.spec.case).toBe('seed');
    expect(request.updateMask?.paths).toEqual(['meta.label-']);
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
    robotsPolicy: null,
    role: null,
    crawlJobIdList: [],
    scriptIdList: [],
    term: null,
    active: '',
    direction: '',
  };
}

function politenessQuery(robotsPolicy: RobotsPolicy | null): ConfigQuery {
  return {...browserScriptQuery(null), kind: Kind.POLITENESSCONFIG, robotsPolicy};
}
