import { Injectable, inject } from '@angular/core';
import {Resolve} from '@angular/router';
import {Observable} from 'rxjs';
import {map, toArray} from 'rxjs/operators';
import { ConfigApiService } from '../../../core';
import { create } from '@bufbuild/protobuf';
import { ListRequestSchema } from '../../../../api/config/v1/config_pb';
import { ConfigObject, Kind } from '../../../shared/models';


export interface ConfigOptions {
  crawlJobs?: ConfigObject[];
}

@Injectable({
  providedIn: 'root'
})
export class OptionsResolver implements Resolve<ConfigOptions> {
  private backendService = inject(ConfigApiService);


  resolve(): Observable<ConfigOptions> | Promise<ConfigOptions> | ConfigOptions {
    const listRequest = create(ListRequestSchema, {kind: Kind.CRAWLJOB.valueOf()});
    return this.backendService.list(listRequest).pipe(
      toArray(),
      map(crawlJobs => crawlJobs.sort((a, b) => a.meta.name.localeCompare(b.meta.name))),
      map(crawlJobs => ({crawlJobs}))
    );
  }

}
