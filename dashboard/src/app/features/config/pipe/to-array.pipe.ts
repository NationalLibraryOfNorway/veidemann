import {Pipe, PipeTransform} from '@angular/core';
import {Observable} from 'rxjs';
import {toArray} from 'rxjs/operators';

@Pipe({
    name: 'toArray',
    standalone: true
})
export class ToArrayPipe implements PipeTransform {

  transform<T>(configObject: Observable<T>): Observable<T[]> {
    return configObject.pipe(toArray());
  }
}
