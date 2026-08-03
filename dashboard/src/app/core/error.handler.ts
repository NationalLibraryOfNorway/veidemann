import { ErrorHandler, Injectable, inject } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';

import {Code, ConnectError} from '@connectrpc/connect';

import {ErrorService} from './error.service';
import { ReferrerError } from '../shared/error';
import { ConfigObject } from '../shared/models';

@Injectable({
  providedIn: 'root'
})
export class ApplicationErrorHandler extends ErrorHandler {
  private errorService = inject(ErrorService);


  override handleError(error: unknown): void {
    console.warn('error handler', error);
    if (typeof error === 'object' && error !== null && 'code' in error) {
      this.handleGrpcError(error);
      return;
    }
    if (error instanceof HttpErrorResponse)
        this.errorService.dispatch(error);
    else if (error instanceof ReferrerError) {
        this.errorService.dispatch(error);
    } else {
        console.error(error);
    }
  }

  handleGrpcError(error: unknown) {
    const connectError = ConnectError.from(error);
    switch (connectError.code) {
      case Code.NotFound:
        console.error('NOT FOUND', connectError.rawMessage);
        break;
      case Code.Unauthenticated:
        console.error('UNAUTHENTICATED', connectError.rawMessage);
        break;
      default:
        console.error('gRPC code:', connectError.code, 'message:', connectError.rawMessage);
        break;
    }
  }

  handleDeleteError(error: unknown, configObject: ConfigObject): void {
    const connectError = ConnectError.from(error);
    if (connectError.rawMessage) {
      const errorString = connectError.rawMessage;
      const deleteError = /(?=.*delete)(?=.*there are)/gm;
      if (deleteError.test(errorString)) {
        this.errorService.dispatch(new ReferrerError({errorString, configObject}));
      }
    }
  }
}
