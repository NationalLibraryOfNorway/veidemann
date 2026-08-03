import { Pipe, PipeTransform, inject } from '@angular/core';
import {Observable} from 'rxjs';
import {first, map} from 'rxjs/operators';
import {CrawlExecutionService} from '../services';

@Pipe({
    name: 'getSeedNamePipe',
    standalone: true
})
export class SeedNamePipe implements PipeTransform {
  private crawlExecutionService = inject(CrawlExecutionService);


  transform(id: string): Observable<string> {
    return this.crawlExecutionService.getSeed(id).pipe(
      first(),
      map(configObject => configObject ? configObject.meta.name : '')
    );
  }
}
