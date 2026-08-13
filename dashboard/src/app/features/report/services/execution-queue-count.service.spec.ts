import {DestroyRef} from '@angular/core';
import {TestBed} from '@angular/core/testing';
import {EMPTY, firstValueFrom, of} from 'rxjs';
import {skip} from 'rxjs/operators';

import {ControllerApiService} from '../../../core';
import {
  CrawlExecutionState,
  CrawlExecutionStatus,
  JobExecutionState,
  JobExecutionStatus,
  ListDataSource,
} from '../../../shared/models';
import {ExecutionQueueCountService} from './execution-queue-count.service';

describe('ExecutionQueueCountService', () => {
  let queueCountsForJobExecutions: ReturnType<typeof vi.fn>;
  let queueCountsForCrawlExecutions: ReturnType<typeof vi.fn>;
  let service: ExecutionQueueCountService;
  const destroyRef = {onDestroy: vi.fn()} as unknown as DestroyRef;

  beforeEach(() => {
    queueCountsForJobExecutions = vi.fn((ids: readonly string[]) => of(new Map(
      ids.map(id => [id, Number(id.split('-').at(-1))]),
    )));
    queueCountsForCrawlExecutions = vi.fn(() => EMPTY);
    TestBed.configureTestingModule({
      providers: [{
        provide: ControllerApiService,
        useValue: {queueCountsForJobExecutions, queueCountsForCrawlExecutions},
      }],
    });
    service = TestBed.inject(ExecutionQueueCountService);
  });

  it('chunks loaded active jobs into bounded requests and merges the responses', async () => {
    const rows = Array.from({length: 201}, (_, index) => new JobExecutionStatus({
      id: `job-${index}`,
      state: JobExecutionState.RUNNING,
    }));
    const dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(...rows),
      destroyRef,
    });

    const counts = await firstValueFrom(service.forJobExecutions(dataSource).pipe(skip(1)));

    expect(queueCountsForJobExecutions.mock.calls.map(call => call[0].length)).toEqual([100, 100, 1]);
    expect(counts.size).toBe(201);
    expect(counts.get('job-200')).toBe(200);
  });

  it('does not request terminal executions and leaves failed active chunks unavailable', async () => {
    const dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(
        new CrawlExecutionStatus({id: 'finished', state: CrawlExecutionState.FINISHED}),
        new CrawlExecutionStatus({id: 'active', state: CrawlExecutionState.FETCHING}),
      ),
      destroyRef,
    });

    const counts = await firstValueFrom(service.forCrawlExecutions(dataSource).pipe(skip(1)));

    expect(queueCountsForCrawlExecutions).toHaveBeenCalledWith(['active']);
    expect(counts).toEqual(new Map());
  });
});
