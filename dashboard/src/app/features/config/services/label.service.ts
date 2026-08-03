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


  private _kind: Kind;

  get kind(): Kind {
    return this._kind;
  }

  set kind(kind: Kind) {
    this._kind = kind;
  }

  getLabelKeys(): Observable<string[]> {
    if (!this.kind) {
      return of([]);
    }
    const request = create(GetLabelKeysRequestSchema, {kind: this._kind.valueOf()});
    return this.configService.getLabelKeys(request);
  }
}
