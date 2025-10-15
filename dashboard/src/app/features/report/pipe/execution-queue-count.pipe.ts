import {Pipe, PipeTransform} from '@angular/core';
import {ControllerApiService} from '../../../core';
import {Observable, of} from 'rxjs';
import {ExecutionId} from '../../../shared/models';
import {CrawlExecutionState, CrawlExecutionStatus} from '../../../shared/models';
import {first, map} from 'rxjs/operators';

@Pipe({
    name: 'getExecutionQueueCountPipe',
    standalone: true
})
export class ExecutionQueueCountPipe implements PipeTransform {
  constructor(private controllerApiService: ControllerApiService) {
  }

  transform(crawlExecutionStatus: CrawlExecutionStatus): Observable<number> {
    const activeStates = [CrawlExecutionState.CREATED, CrawlExecutionState.FETCHING, CrawlExecutionState.SLEEPING];
    if (activeStates.includes(crawlExecutionStatus.state)) {
      const exectionId = new ExecutionId({id: crawlExecutionStatus.id});
      return this.controllerApiService.queueCountForCrawlExecution(exectionId).pipe(
        first(),
        map(countResponse => countResponse.count));
    }
    return of(0);
  }
}
