import {Injectable} from '@angular/core';
import {AuthService} from '../auth';
import {ErrorService} from '../error.service';
import {EMPTY, Observable, Observer} from 'rxjs';
import {catchError, map} from 'rxjs/operators';
import {CrawlLogListRequest, CrawlLogProto, LogPromiseClient, PageLogListRequest, PageLogProto} from '../../../api';
import {AppConfig} from '../../app.config';
import {CrawlLog, PageLog} from '../../shared/models';


@Injectable({
  providedIn: 'root'
})
export class LogApiService {

  private logClient: LogPromiseClient;

  constructor(private authService: AuthService,
              appConfigService: AppConfig,
              private errorService: ErrorService) {
    this.logClient = new LogPromiseClient(appConfigService.grpcWebUrl, null, null);
  }


  listPageLogs(listRequest: PageLogListRequest): Observable<PageLog> {
    const metadata = this.authService.metadata;

    return new Observable((observer: Observer<PageLogProto>) => {
      const stream = this.logClient.listPageLogs(listRequest, metadata)
        .on('data', data => observer.next(data))
        .on('error', error => observer.error(error))
        .on('end', () => observer.complete());
      return () => stream.cancel();
    }).pipe(
      map(PageLog.fromProto),
      catchError(error => {
        this.errorService.dispatch(error);
        return EMPTY;
      })
    );
  }


  listCrawlLogs(listRequest: CrawlLogListRequest): Observable<CrawlLog> {
    const metadata = this.authService.metadata;

    return new Observable((observer: Observer<CrawlLogProto>) => {
      const stream = this.logClient.listCrawlLogs(listRequest, metadata)
        .on('data', data => observer.next(data))
        .on('error', error => observer.error(error))
        .on('end', () => observer.complete());
      return () => stream.cancel();
    }).pipe(
      map(CrawlLog.fromProto),
      catchError(error => {
        this.errorService.dispatch(error);
        return EMPTY;
      })
    );
  }

}
