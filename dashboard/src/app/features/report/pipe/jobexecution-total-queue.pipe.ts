import {Pipe, PipeTransform} from '@angular/core';
import {CrawlExecutionState, CrawlExecutionStatus, JobExecutionState, JobExecutionStatus} from '../../../shared/models';
import {Observable, of} from 'rxjs';
import {map, mergeMap, reduce,} from 'rxjs/operators';
import {ExecutionId} from '../../../shared/models';
import {ControllerApiService, ReportApiService} from '../../../core';
import {create} from '@bufbuild/protobuf';
import {FieldMaskSchema} from '../../../../api/commons/v1/resources_pb';
import {CrawlExecutionsListRequestSchema} from '../../../../api/report/v1/report_pb';

@Pipe({
    name: 'getUrlQueueForJobExecution',
    standalone: true
})
export class JobexecutionTotalQueuePipe implements PipeTransform {
  constructor(private reportApiService: ReportApiService, private controllerApiService: ControllerApiService) {
  }

  transform(jobExectionStatus: JobExecutionStatus): Observable<number> {
    if (!jobExectionStatus) {
      return of(0);
    }

    const activeJobStates = [
      JobExecutionState.CREATED,
      JobExecutionState.RUNNING
    ];

    const activeExecutionStates = [
      CrawlExecutionState.CREATED.valueOf(),
      CrawlExecutionState.FETCHING.valueOf(),
      CrawlExecutionState.SLEEPING.valueOf()
    ];

    if (!activeJobStates.includes(jobExectionStatus.state)) {
      return of(0);
    }

    const queryTemplate = new CrawlExecutionStatus();
    queryTemplate.jobExecutionId = jobExectionStatus.id;
    queryTemplate.jobId = jobExectionStatus.jobId;

    const listRequest = create(CrawlExecutionsListRequestSchema, {
      queryTemplate: CrawlExecutionStatus.toProto(queryTemplate),
      queryMask: create(FieldMaskSchema, {paths: ['jobExecutionId', 'jobId']}),
      returnedFieldsMask: create(FieldMaskSchema, {paths: ['state', 'id', 'jobExecutionId']}),
      state: activeExecutionStates
    });

    return this.reportApiService.listCrawlExecutions(listRequest).pipe(
      mergeMap(crawlExecutionStatus => {
        if (!activeExecutionStates.includes(crawlExecutionStatus.state)) {
          return of(0);
        }
        const executionId = new ExecutionId({id: crawlExecutionStatus.id});
        return this.controllerApiService.queueCountForCrawlExecution(executionId).pipe(
          map(countResponse => countResponse.count));
      }),
      reduce((acc, curr) => acc + curr, 0),
    );
  }
}
