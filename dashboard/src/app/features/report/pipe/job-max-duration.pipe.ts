import {Pipe, PipeTransform, inject} from '@angular/core';
import {Observable} from 'rxjs';
import {defaultIfEmpty, map, take} from 'rxjs/operators';
import {JobExecutionService} from '../services';

/**
 * Resolves the configured maximum duration (seconds) of a crawl job. The job is
 * served from the cache the job name lookup already fills, so this adds no requests.
 * Emits 0 when the job has no limit or cannot be loaded.
 */
@Pipe({
  name: 'getJobMaxDurationPipe',
  standalone: true
})
export class JobMaxDurationPipe implements PipeTransform {
  private jobExecutionService = inject(JobExecutionService);

  transform(id: string): Observable<number> {
    return this.jobExecutionService.getJob(id).pipe(
      take(1),
      map(configObject => configObject?.crawlJob?.limits?.maxDurationS ?? 0),
      defaultIfEmpty(0),
    );
  }
}
