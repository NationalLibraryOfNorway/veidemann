import {ErrorHandler, Injectable, inject} from '@angular/core';

import {ConnectError} from '@connectrpc/connect';

import {SnackBarService} from './snack-bar/snack-bar.service';
import {ReferrerError} from '../shared/error';
import {ConfigObject} from '../shared/models';

@Injectable({
  providedIn: 'root'
})
export class ApplicationErrorHandler extends ErrorHandler {
  private snackBarService = inject(SnackBarService);

  override handleError(error: unknown): void {
    this.snackBarService.openError(error);
  }

  handleDeleteError(error: unknown, configObject: ConfigObject): void {
    const connectError = ConnectError.from(error);
    if (connectError.rawMessage) {
      const errorString = connectError.rawMessage;
      const deleteError = /(?=.*delete)(?=.*there are)/gm;
      if (deleteError.test(errorString)) {
        this.handleError(new ReferrerError({errorString, configObject}));
        return;
      }
    }
    this.handleError(error);
  }
}
