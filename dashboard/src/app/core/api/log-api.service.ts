import { Injectable, inject } from '@angular/core';
import {CallOptions, Client, createClient} from '@connectrpc/connect';
import {createGrpcWebTransport} from '@connectrpc/connect-web';
import {EMPTY, Observable} from 'rxjs';
import {catchError, map} from 'rxjs/operators';

import {CrawlLogListRequest, Log, PageLogListRequest} from '../../../api/log/v1/log_pb';
import {AuthService} from '../auth';
import {ErrorService} from '../error.service';
import {AppConfig} from '../../app.config';
import {CrawlLog, PageLog} from '../../shared/models';
import {fromServerStream} from './connect-observable';

@Injectable({providedIn: 'root'})
export class LogApiService {
  private authService = inject(AuthService);
  private appConfig = inject(AppConfig);
  private errorService = inject(ErrorService);

  private client?: Client<typeof Log>;

  private getClient(): Client<typeof Log> {
    if (!this.client) {
      if (!this.appConfig.grpcWebUrl) {
        throw new Error('grpcWebUrl is not configured yet');
      }
      this.client = createClient(Log, createGrpcWebTransport({
        baseUrl: this.appConfig.grpcWebUrl,
      }));
    }
    return this.client;
  }

  private get callOptions(): CallOptions {
    return {headers: this.authService.metadata};
  }

  listPageLogs(listRequest: PageLogListRequest): Observable<PageLog> {
    return fromServerStream(signal => this.getClient().listPageLogs(listRequest, {
      ...this.callOptions,
      signal,
    })).pipe(
      map(PageLog.fromProto),
      catchError(error => {
        this.errorService.dispatch(error);
        return EMPTY;
      }),
    );
  }

  listCrawlLogs(listRequest: CrawlLogListRequest): Observable<CrawlLog> {
    return fromServerStream(signal => this.getClient().listCrawlLogs(listRequest, {
      ...this.callOptions,
      signal,
    })).pipe(
      map(CrawlLog.fromProto),
      catchError(error => {
        this.errorService.dispatch(error);
        return EMPTY;
      }),
    );
  }
}
