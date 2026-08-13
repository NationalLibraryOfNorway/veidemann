import {inject, Injectable} from '@angular/core';
import {defaultIfEmpty, map, mergeMap, reduce, shareReplay, startWith, switchMap} from 'rxjs/operators';
import {from, merge, Observable, of} from 'rxjs';

import {ControllerApiService} from '../../../core';
import {
  CrawlExecutionState,
  CrawlExecutionStatus,
  JobExecutionState,
  JobExecutionStatus,
  ListDataSource,
  ListItem,
} from '../../../shared/models';

export type ExecutionQueueCounts = ReadonlyMap<string, number>;

const MAX_IDS_PER_REQUEST = 100;
const MAX_CONCURRENT_REQUESTS = 4;

@Injectable({providedIn: 'root'})
export class ExecutionQueueCountService {
  private readonly controllerApiService = inject(ControllerApiService);

  forJobExecutions<Q>(
    dataSource: ListDataSource<JobExecutionStatus, Q>,
  ): Observable<ExecutionQueueCounts> {
    return this.forExecutions(
      dataSource,
      item => item.state === JobExecutionState.CREATED || item.state === JobExecutionState.RUNNING,
      ids => this.controllerApiService.queueCountsForJobExecutions(ids),
    );
  }

  forCrawlExecutions<Q>(
    dataSource: ListDataSource<CrawlExecutionStatus, Q>,
  ): Observable<ExecutionQueueCounts> {
    return this.forExecutions(
      dataSource,
      item => item.state === CrawlExecutionState.CREATED
        || item.state === CrawlExecutionState.FETCHING
        || item.state === CrawlExecutionState.SLEEPING,
      ids => this.controllerApiService.queueCountsForCrawlExecutions(ids),
    );
  }

  private forExecutions<T extends ListItem, Q>(
    dataSource: ListDataSource<T, Q>,
    isActive: (item: T) => boolean,
    load: (ids: readonly string[]) => Observable<ExecutionQueueCounts>,
  ): Observable<ExecutionQueueCounts> {
    const completedRows$ = dataSource.completed$.pipe(map(completed => completed.rows));
    return merge(of(dataSource.snapshot), completedRows$).pipe(
      map(rows => [...new Set(rows.filter(isActive).map(row => row.id))]),
      switchMap(ids => from(this.chunks(ids)).pipe(
        mergeMap(
          chunk => load(chunk).pipe(defaultIfEmpty(new Map<string, number>())),
          MAX_CONCURRENT_REQUESTS,
        ),
        reduce((counts, chunkCounts) => {
          chunkCounts.forEach((count, id) => counts.set(id, count));
          return counts;
        }, new Map<string, number>()),
      )),
      startWith(new Map<string, number>()),
      shareReplay({bufferSize: 1, refCount: true}),
    );
  }

  private chunks(ids: readonly string[]): string[][] {
    const chunks: string[][] = [];
    for (let offset = 0; offset < ids.length; offset += MAX_IDS_PER_REQUEST) {
      chunks.push(ids.slice(offset, offset + MAX_IDS_PER_REQUEST));
    }
    return chunks;
  }
}
