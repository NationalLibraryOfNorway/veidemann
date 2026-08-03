import { Pipe, PipeTransform, inject } from '@angular/core';
import {Observable} from 'rxjs';
import {first, map} from 'rxjs/operators';
import {JobExecutionService} from '../services';


@Pipe({
    name: 'getJobNamePipe',
    standalone: true
})
export class JobNamePipe implements PipeTransform {
  private jobExecutionService = inject(JobExecutionService);


  transform(id: string): Observable<string> {
    return this.jobExecutionService.getJob(id).pipe(
      first(),
      map(configObject => configObject ? configObject.meta.name : '')
    );
  }
}
