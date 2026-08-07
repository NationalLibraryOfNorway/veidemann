import { Pipe, PipeTransform, inject } from '@angular/core';
import {ActivatedRoute} from '@angular/router';
import {Observable} from 'rxjs';
import {first, map} from 'rxjs/operators';
import {ConfigObject} from '../../../shared/models/config';
import {OptionsService} from '../services/options.service';


@Pipe({
    name: 'getBrowserConfigName',
    standalone: true
})
export class BrowserConfigNamePipe implements PipeTransform {
  private route = inject(ActivatedRoute);
  private optionsService = inject(OptionsService);


  transform(configObject: ConfigObject): Observable<string> {
    return this.optionsService.options$.pipe(
      first(),
      map(options => {
        const found = options?.browserConfigs?.find(
          browserConfig => browserConfig.id === configObject.crawlConfig.browserConfigRef.id);
        return found?.meta.name || configObject.crawlConfig.browserConfigRef.id;
      }));
  }
}
