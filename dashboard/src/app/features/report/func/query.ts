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

interface RouteListQuery {
  active: string;
  direction: SortDirection;
  pageIndex: number;
  pageSize: number;
  watch: boolean;
}

export function pageLogQueryFromParamMap(params: ParamMap): PageLogQuery {
  return {
    ...listQueryFromParamMap(params),
    uri: params.get('uri'),
    executionId: params.get('execution_id'),
    jobExecutionId: params.get('job_execution_id'),
  };
}

export function crawlLogQueryFromParamMap(params: ParamMap): CrawlLogQuery {
  return {
    ...listQueryFromParamMap(params),
    executionId: params.get('execution_id'),
    jobExecutionId: params.get('job_execution_id'),
  };
}

export function jobExecutionQueryFromParamMap(params: ParamMap): JobExecutionStatusQuery {
  return {
    ...listQueryFromParamMap(params),
    jobId: params.get('job_id'),
    stateList: numericValues<JobExecutionState>(params.getAll('state')),
    startTimeFrom: validDate(params.get('start_time_from')),
    startTimeTo: validDate(params.get('start_time_to')),
  };
}

export function crawlExecutionQueryFromParamMap(params: ParamMap): CrawlExecutionStatusQuery {
  return {
    ...listQueryFromParamMap(params),
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
  return equalListQuery(previous, current)
    && previous.uri === current.uri
    && previous.executionId === current.executionId
    && previous.jobExecutionId === current.jobExecutionId;
}

export function equalCrawlLogQuery(previous: CrawlLogQuery, current: CrawlLogQuery): boolean {
  return equalListQuery(previous, current)
    && previous.executionId === current.executionId
    && previous.jobExecutionId === current.jobExecutionId;
}

export function equalJobExecutionQuery(
  previous: JobExecutionStatusQuery,
  current: JobExecutionStatusQuery
): boolean {
  return equalListQuery(previous, current)
    && previous.jobId === current.jobId
    && equalArrayValues(previous.stateList, current.stateList)
    && previous.startTimeFrom === current.startTimeFrom
    && previous.startTimeTo === current.startTimeTo;
}

export function equalCrawlExecutionQuery(
  previous: CrawlExecutionStatusQuery,
  current: CrawlExecutionStatusQuery
): boolean {
  return equalListQuery(previous, current)
    && previous.jobId === current.jobId
    && previous.jobExecutionId === current.jobExecutionId
    && previous.seedId === current.seedId
    && equalArrayValues(previous.stateList, current.stateList)
    && previous.hasError === current.hasError
    && previous.startTimeFrom === current.startTimeFrom
    && previous.startTimeTo === current.startTimeTo;
}

export function unknownPageLength(query: RouteListQuery, rows: readonly unknown[]): number {
  const loaded = query.pageIndex * query.pageSize + rows.length;
  return loaded + (rows.length === query.pageSize ? 1 : 0);
}

function listQueryFromParamMap(params: ParamMap): RouteListQuery {
  const [active = '', parsedDirection = ''] = params.get('sort')?.split(':') ?? [];
  const direction = parsedDirection ? parsedDirection as SortDirection : '';

  return {
    active: direction ? active : '',
    direction,
    pageIndex: Number.parseInt(params.get('p'), 10) || 0,
    pageSize: Number.parseInt(params.get('s'), 10) || 25,
    watch: params.get('watch') === 'true',
  };
}

function equalListQuery(previous: RouteListQuery, current: RouteListQuery): boolean {
  return previous.active === current.active
    && previous.direction === current.direction
    && previous.pageIndex === current.pageIndex
    && previous.pageSize === current.pageSize
    && previous.watch === current.watch;
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
