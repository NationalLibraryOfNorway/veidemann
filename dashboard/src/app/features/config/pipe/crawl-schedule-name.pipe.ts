import { Pipe, PipeTransform, inject } from '@angular/core';
import {ActivatedRoute} from '@angular/router';
import {Observable} from 'rxjs';
import {map} from 'rxjs/operators';
import {ConfigObject} from '../../../shared/models/config';
import {OptionsService} from '../services/options.service';


@Pipe({
    name: 'getCrawlScheduleName',
    standalone: true
})
export class CrawlScheduleNamePipe implements PipeTransform {
  private route = inject(ActivatedRoute);
  private optionsService = inject(OptionsService);


  transform(configObject: ConfigObject): Observable<string> {
    return this.optionsService.options$.pipe(
      map(options => {
        const found = options.crawlScheduleConfigs.find(
          crawlSchedule => crawlSchedule.id === configObject.crawlJob.scheduleRef.id);
        return found ? found.meta.name : 'crawlSchedule';
      }));
  }
}
