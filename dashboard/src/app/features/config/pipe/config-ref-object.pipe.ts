import {inject, Pipe, PipeTransform} from '@angular/core';
import {Observable, of} from 'rxjs';
import {catchError, take} from 'rxjs/operators';

import {ConfigObject, ConfigRef} from '../../../shared/models';
import {ConfigService} from '../../../shared/services';

@Pipe({
  name: 'getConfigRefObject',
  standalone: true,
})
export class ConfigRefObjectPipe implements PipeTransform {
  private configService = inject(ConfigService);

  transform(ref: ConfigRef | null | undefined): Observable<ConfigObject | null> {
    if (!ref?.id) {
      return of(null);
    }
    return this.configService.get(ref).pipe(
      take(1),
      catchError(() => of(null)),
    );
  }
}
