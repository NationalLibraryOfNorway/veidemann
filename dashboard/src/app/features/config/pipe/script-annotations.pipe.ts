import {Pipe, PipeTransform} from '@angular/core';
import {ConfigService} from '../../commons/services';
import {Annotation} from '../../shared/models/config';
import {Observable, of} from 'rxjs';


@Pipe({
    name: 'getScriptAnnotations',
    standalone: true
})
export class ScriptAnnotationsPipe implements PipeTransform {
  constructor(private configService: ConfigService) {
  }

  transform(jobId: string, seedId?: string): Observable<Annotation[]> {
    if (!jobId) {
      return of([]);
    }
    return this.configService.getScriptAnnotations(jobId, seedId);
  }
}
