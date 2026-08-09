import { Injectable, inject } from '@angular/core';
import {Observable, of} from 'rxjs';
import {create} from '@bufbuild/protobuf';
import {GetLabelKeysRequestSchema} from '../../../../api/config/v1/config_pb';
import {ConfigApiService} from '../../../core';
import {Kind} from '../../../shared/models';

@Injectable({
  providedIn: 'root'
})
export class LabelService {
  private configService = inject(ConfigApiService);

  getLabelKeys(kind: Kind): Observable<string[]> {
    if (kind === Kind.UNDEFINED) {
      return of([]);
    }
    const request = create(GetLabelKeysRequestSchema, {kind: kind.valueOf()});
    return this.configService.getLabelKeys(request);
  }
}
