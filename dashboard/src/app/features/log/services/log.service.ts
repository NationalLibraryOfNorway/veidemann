import { Injectable, inject } from '@angular/core';
import {Observable} from 'rxjs';

import {ConfigApiService} from '../../../core/api/config-api.service';
import {LogLevels} from '../../../shared/models';

@Injectable({providedIn: 'root'})
export class LogService {
  private configApiService = inject(ConfigApiService);


  getLogConfig(): Observable<LogLevels> {
    return this.configApiService.getLogConfig();
  }

  saveLogConfig(logLevels: LogLevels): Observable<LogLevels> {
    return this.configApiService.saveLogConfig(logLevels);
  }
}
