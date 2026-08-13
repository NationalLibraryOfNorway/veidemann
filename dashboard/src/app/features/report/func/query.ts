import {SortDirection} from '@angular/material/sort';
import {ParamMap} from '@angular/router';

import {isValidDate} from '../../../shared/func';
import {CrawlExecutionState, JobExecutionState} from '../../../shared/models';
import {
  CrawlExecutionStatusQuery,
  CrawlLogQuery,
  JobExecutionStatusQuery,
  PageLogQuery
} from '../services';

interface RouteSortQuery {
  active: string;
  direction: SortDirection;
}

export function pageLogQueryFromParamMap(params: ParamMap): PageLogQuery {
  return {
    ...sortQueryFromParamMap(params),
    uri: params.get('uri'),
    executionId: params.get('execution_id'),
    jobExecutionId: params.get('job_execution_id'),
  };
}

export function crawlLogQueryFromParamMap(params: ParamMap): CrawlLogQuery {
  return {
    ...sortQueryFromParamMap(params),
    executionId: params.get('execution_id'),
    jobExecutionId: params.get('job_execution_id'),
  };
}

export function jobExecutionQueryFromParamMap(params: ParamMap): JobExecutionStatusQuery {
  return {
    ...sortQueryFromParamMap(params),
    jobId: params.get('job_id'),
    stateList: numericValues<JobExecutionState>(params.getAll('state')),
    startTimeFrom: validDate(params.get('start_time_from')),
    startTimeTo: validDate(params.get('start_time_to')),
  };
}

export function crawlExecutionQueryFromParamMap(params: ParamMap): CrawlExecutionStatusQuery {
  return {
    ...sortQueryFromParamMap(params),
    jobId: params.get('job_id'),
    jobExecutionId: params.get('job_execution_id'),
    seedId: params.get('seed_id'),
    stateList: numericValues<CrawlExecutionState>(params.getAll('state')),
    hasError: params.get('has_error') === 'true',
    startTimeFrom: validDate(params.get('start_time_from')),
    startTimeTo: validDate(params.get('start_time_to')),
  };
}

export function equalPageLogQuery(previous: PageLogQuery, current: PageLogQuery): boolean {
  return equalSortQuery(previous, current)
    && previous.uri === current.uri
    && previous.executionId === current.executionId
    && previous.jobExecutionId === current.jobExecutionId;
}

export function equalCrawlLogQuery(previous: CrawlLogQuery, current: CrawlLogQuery): boolean {
  return equalSortQuery(previous, current)
    && previous.executionId === current.executionId
    && previous.jobExecutionId === current.jobExecutionId;
}

export function equalJobExecutionQuery(
  previous: JobExecutionStatusQuery,
  current: JobExecutionStatusQuery
): boolean {
  return equalSortQuery(previous, current)
    && previous.jobId === current.jobId
    && equalArrayValues(previous.stateList, current.stateList)
    && previous.startTimeFrom === current.startTimeFrom
    && previous.startTimeTo === current.startTimeTo;
}

export function equalCrawlExecutionQuery(
  previous: CrawlExecutionStatusQuery,
  current: CrawlExecutionStatusQuery
): boolean {
  return equalSortQuery(previous, current)
    && previous.jobId === current.jobId
    && previous.jobExecutionId === current.jobExecutionId
    && previous.seedId === current.seedId
    && equalArrayValues(previous.stateList, current.stateList)
    && previous.hasError === current.hasError
    && previous.startTimeFrom === current.startTimeFrom
    && previous.startTimeTo === current.startTimeTo;
}

function sortQueryFromParamMap(params: ParamMap): RouteSortQuery {
  const [active = '', parsedDirection = ''] = params.get('sort')?.split(':') ?? [];
  const direction = parsedDirection ? parsedDirection as SortDirection : '';

  return {
    active: direction ? active : '',
    direction,
  };
}

function equalSortQuery(previous: RouteSortQuery, current: RouteSortQuery): boolean {
  return previous.active === current.active
    && previous.direction === current.direction;
}

function equalArrayValues<T>(previous: readonly T[], current: readonly T[]): boolean {
  return previous.length === current.length
    && previous.every(value => current.some(currentValue => value === currentValue));
}

function numericValues<T extends number>(values: string[]): T[] {
  return values
    .map(value => Number.parseInt(value, 10))
    .filter(value => !Number.isNaN(value)) as T[];
}

function validDate(value: string): string {
  return value && isValidDate(new Date(value)) ? value : null;
}
