import { ErrorHandler, Injectable, inject } from '@angular/core';
import {CallOptions, Client, createClient} from '@connectrpc/connect';
import {createGrpcWebTransport} from '@connectrpc/connect-web';
import {from, MonoTypeOperatorFunction, Observable, of} from 'rxjs';
import {catchError, map} from 'rxjs/operators';

import {
  Config,
  GetLabelKeysRequest,
  GetScriptAnnotationsRequest,
  ListRequest,
  UpdateRequest,
} from '../../../api/config/v1/config_pb';
import {AuthService} from '../auth';
import {
  Annotation,
  ConfigObject,
  ConfigRef,
} from '../../shared/models/config';
import {ApplicationErrorHandler} from '../error.handler';
import {AppConfig} from '../../app.config';
import {fromServerStream} from './connect-observable';

const catchConfigError = <T>(errorService: ErrorHandler, returnValue: T): MonoTypeOperatorFunction<T> =>
  catchError((error: unknown) => {
    errorService.handleError(error);
    return of(returnValue);
  });

@Injectable({providedIn: 'root'})
export class ConfigApiService {
  protected authService = inject(AuthService);
  private appConfig = inject(AppConfig);
  private errorHandler = inject(ApplicationErrorHandler);

  private client?: Client<typeof Config>;

  private getClient(): Client<typeof Config> {
    if (!this.client) {
      if (!this.appConfig.grpcWebUrl) {
        throw new Error('grpcWebUrl is not configured yet');
      }
      this.client = createClient(Config, createGrpcWebTransport({
        baseUrl: this.appConfig.grpcWebUrl,
      }));
    }
    return this.client;
  }

  private get callOptions(): CallOptions {
    return {headers: this.authService.metadata};
  }

  list(listRequest: ListRequest): Observable<ConfigObject> {
    return fromServerStream(signal => this.getClient().listConfigObjects(listRequest, {
      ...this.callOptions,
      signal,
    })).pipe(
      map(ConfigObject.fromProto),
      catchConfigError<ConfigObject>(this.errorHandler, null),
    );
  }

  count(request: ListRequest): Observable<number> {
    return from(this.getClient().countConfigObjects(request, this.callOptions)).pipe(
      map(response => Number(response.count)),
      catchConfigError<number>(this.errorHandler, 0),
    );
  }

  get(configRef: ConfigRef): Observable<ConfigObject> {
    return from(this.getClient().getConfigObject(ConfigRef.toProto(configRef), this.callOptions)).pipe(
      map(ConfigObject.fromProto),
      catchConfigError<ConfigObject>(this.errorHandler, null),
    );
  }

  save(config: ConfigObject): Observable<ConfigObject> {
    return from(this.getClient().saveConfigObject(ConfigObject.toProto(config), this.callOptions)).pipe(
      map(ConfigObject.fromProto),
      catchConfigError<ConfigObject>(this.errorHandler, null),
    );
  }

  update(updateRequest: UpdateRequest): Observable<number> {
    return from(this.getClient().updateConfigObjects(updateRequest, this.callOptions)).pipe(
      map(response => Number(response.updated)),
      catchConfigError<number>(this.errorHandler, 0),
    );
  }

  delete(configObject: ConfigObject): Observable<boolean> {
    return from(this.getClient().deleteConfigObject(ConfigObject.toProto(configObject), this.callOptions)).pipe(
      map(response => response.deleted),
      catchError(error => {
        this.errorHandler.handleDeleteError(error, configObject);
        return of(false);
      }),
    );
  }

  getLabelKeys(request: GetLabelKeysRequest): Observable<string[]> {
    return from(this.getClient().getLabelKeys(request, this.callOptions)).pipe(
      map(response => response.key),
      catchConfigError<string[]>(this.errorHandler, []),
    );
  }

  getScriptAnnotations(request: GetScriptAnnotationsRequest): Observable<Annotation[]> {
    return from(this.getClient().getScriptAnnotations(request, this.callOptions)).pipe(
      map(response => response.annotation.map(Annotation.fromProto)),
      catchConfigError<Annotation[]>(this.errorHandler, []),
    );
  }

}
