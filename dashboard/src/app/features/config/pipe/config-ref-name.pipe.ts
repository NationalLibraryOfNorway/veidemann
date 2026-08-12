import {Pipe, PipeTransform, inject} from '@angular/core';
import {Observable, of} from 'rxjs';
import {catchError, map, take} from 'rxjs/operators';

import {ConfigRef} from '../../../shared/models';
import {ConfigService} from '../../../shared/services';

@Pipe({
  name: 'getConfigRefName',
  standalone: true,
})
export class ConfigRefNamePipe implements PipeTransform {
  private configService = inject(ConfigService);

  transform(ref: ConfigRef | null | undefined): Observable<string> {
    if (!ref?.id) {
      return of('');
    }
    return this.configService.get(ref).pipe(
      take(1),
      map(configObject => configObject?.meta?.name?.trim() || ref.id),
      catchError(() => of(ref.id)),
    );
  }
}
