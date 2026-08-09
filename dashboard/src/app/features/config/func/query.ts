import {create} from '@bufbuild/protobuf';
import {BrowserScriptType, ConfigObject, Kind, Role, RobotsPolicy} from '../../../shared/models';
import {FieldMask} from '../../../../api/commons/v1/resources_pb';
import {ListRequestSchema} from '../../../../api/config/v1/config_pb';
import {ParamMap} from '@angular/router';
import {ConfigQuery} from '../../../shared/func';
import {SortDirection} from '@angular/material/sort';

export interface ConfigLabelSelector {
  selector: string;
  key: string;
  value: string;
  structured: boolean;
}

export interface ParsedConfigSearchTerm {
  name: string;
  label: ConfigLabelSelector | null;
}

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
    browserScriptType: browserScriptTypeFromParamMap(params),
    robotsPolicy: robotsPolicyFromParamMap(params),
    role: roleFromParamMap(params),
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
    && previous.browserScriptType === current.browserScriptType
    && previous.robotsPolicy === current.robotsPolicy
    && previous.role === current.role
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
    && previous.browserScriptType === current.browserScriptType
    && previous.robotsPolicy === current.robotsPolicy
    && previous.role === current.role
    && equalArrayValues(previous.crawlJobIdList, current.crawlJobIdList)
    && equalArrayValues(previous.scriptIdList, current.scriptIdList)
    && previous.term === current.term;
}

export function parseConfigSearchTerm(term: string): ParsedConfigSearchTerm {
  const marker = 'label:';
  const markerIndex = term.indexOf(marker);
  if (markerIndex < 0) {
    return {name: term, label: null};
  }
  const selector = term.slice(markerIndex + marker.length).trim();
  if (!selector) {
    return {name: term, label: null};
  }
  const delimiterIndex = selector.indexOf(':');
  return {
    name: term.slice(0, markerIndex).trim(),
    label: {
      selector,
      key: delimiterIndex < 0 ? selector : selector.slice(0, delimiterIndex),
      value: delimiterIndex < 0 ? '' : selector.slice(delimiterIndex + 1),
      structured: delimiterIndex >= 0,
    },
  };
}

export function serializeConfigSearchTerm(name: string, label: ConfigLabelSelector | null): string {
  if (!label) {
    return name;
  }
  const trimmedName = name?.trim() ?? '';
  return `${trimmedName}${trimmedName ? ' ' : ''}label:${label.selector}`;
}

function browserScriptTypeFromParamMap(params: ParamMap): BrowserScriptType | null {
  const rawValue = params.get('script_type');
  if (rawValue === null || rawValue.trim() === '') {
    return null;
  }

  const value = Number(rawValue);
  return Number.isInteger(value)
    && value !== BrowserScriptType.UNDEFINED
    && BrowserScriptType[value] !== undefined
    ? value as BrowserScriptType
    : null;
}

function robotsPolicyFromParamMap(params: ParamMap): RobotsPolicy | null {
  const rawValue = params.get('robots_policy');
  if (rawValue === null || rawValue.trim() === '') return null;
  const value = Number(rawValue);
  return Number.isInteger(value) && RobotsPolicy[value] !== undefined ? value as RobotsPolicy : null;
}

function roleFromParamMap(params: ParamMap): Role | null {
  const rawValue = params.get('role');
  if (rawValue === null || rawValue.trim() === '') return null;
  const value = Number(rawValue);
  return Number.isInteger(value) && Role[value] !== undefined ? value as Role : null;
}

function equalArrayValues(previous: readonly string[], current: readonly string[]): boolean {
  return previous.length === current.length
    && previous.every(value => current.some(currentValue => value === currentValue));
}
