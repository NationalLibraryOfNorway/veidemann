import {create} from '@bufbuild/protobuf';
import {ConfigObject, Kind} from '../../../shared/models';
import {FieldMask} from '../../../../api/commons/v1/resources_pb';
import {ListRequestSchema} from '../../../../api/config/v1/config_pb';
import {ParamMap} from '@angular/router';
import {ConfigQuery} from '../../../shared/func';
import {SortDirection} from '@angular/material/sort';

export function createListRequest(kind: Kind, queryTemplate?: Partial<ConfigObject>, queryMask?: FieldMask) {
  const listRequest = create(ListRequestSchema, {kind: kind.valueOf()});

  if (queryTemplate) {
    const configObject = new ConfigObject();
    Object.assign(configObject, queryTemplate);

    listRequest.queryTemplate = ConfigObject.toProto(configObject);
  }
  if (queryMask) {
    listRequest.queryMask = queryMask;
  }
  return listRequest;
}

export function configQueryFromParamMap(kind: Kind, params: ParamMap): ConfigQuery {
  const [active = '', parsedDirection = ''] = params.get('sort')?.split(':') ?? [];
  const direction = parsedDirection ? parsedDirection as SortDirection : '';

  return {
    kind,
    entityId: params.get('entity_id'),
    scheduleId: params.get('schedule_id'),
    crawlConfigId: params.get('crawl_config_id'),
    collectionId: params.get('collection_id'),
    browserConfigId: params.get('browser_config_id'),
    politenessId: params.get('politeness_id'),
    disabled: params.has('disabled') ? params.get('disabled') === 'true' : null,
    crawlJobIdList: params.getAll('crawl_job_id'),
    scriptIdList: params.getAll('script_id'),
    term: params.get('q'),
    active: direction ? active : '',
    direction,
  };
}

export function equalConfigQuery(previous: ConfigQuery, current: ConfigQuery): boolean {
  return previous.kind === current.kind
    && previous.entityId === current.entityId
    && previous.scheduleId === current.scheduleId
    && previous.crawlConfigId === current.crawlConfigId
    && previous.collectionId === current.collectionId
    && previous.browserConfigId === current.browserConfigId
    && previous.politenessId === current.politenessId
    && previous.disabled === current.disabled
    && equalArrayValues(previous.crawlJobIdList, current.crawlJobIdList)
    && equalArrayValues(previous.scriptIdList, current.scriptIdList)
    && previous.term === current.term
    && previous.active === current.active
    && previous.direction === current.direction;
}

export function equalConfigCountQuery(previous: ConfigQuery, current: ConfigQuery): boolean {
  return previous.kind === current.kind
    && previous.entityId === current.entityId
    && previous.scheduleId === current.scheduleId
    && previous.crawlConfigId === current.crawlConfigId
    && previous.collectionId === current.collectionId
    && previous.browserConfigId === current.browserConfigId
    && previous.politenessId === current.politenessId
    && previous.disabled === current.disabled
    && equalArrayValues(previous.crawlJobIdList, current.crawlJobIdList)
    && equalArrayValues(previous.scriptIdList, current.scriptIdList)
    && previous.term === current.term;
}

function equalArrayValues(previous: readonly string[], current: readonly string[]): boolean {
  return previous.length === current.length
    && previous.every(value => current.some(currentValue => value === currentValue));
}
