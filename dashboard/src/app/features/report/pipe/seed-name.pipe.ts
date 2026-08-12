import { Pipe, PipeTransform, inject } from '@angular/core';
import {Observable} from 'rxjs';
import {map} from 'rxjs/operators';
import {CrawlExecutionService} from '../services';

export interface SeedNameResolution {
  deleted: boolean;
  label: string;
}

@Pipe({
    name: 'getSeedNamePipe',
    standalone: true
})
export class SeedNamePipe implements PipeTransform {
  private crawlExecutionService = inject(CrawlExecutionService);


  transform(id: string): Observable<SeedNameResolution> {
    return this.crawlExecutionService.getSeed(id).pipe(
      map(configObject => configObject
        ? {deleted: false, label: configObject.meta.name || id}
        : {deleted: true, label: $localize`:@@crawlExecutionDeletedSeedName:Deleted seed`})
    );
  }
}
