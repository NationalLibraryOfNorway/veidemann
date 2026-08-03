import {CustomError} from './custom-error';
import {ConfigObject} from '../models';

interface ReferrerErrorOptions {
  configObject?: ConfigObject;
  numConfigs?: number;
  numDeleted?: number;
  errorString?: string;
}

export class ReferrerError extends CustomError {
  constructor(options: ReferrerErrorOptions) {
    super(ReferrerError.createMessage(options));
  }

  private static createMessage({configObject, numConfigs, numDeleted, errorString}: ReferrerErrorOptions): string {
    if (numConfigs !== undefined && numDeleted !== undefined) {
      const notDeletedMsg = numConfigs - numDeleted + ' ble ikke slettet siden de brukes i andre konfigurasjoner ';
      const deletedMsg = numDeleted + '/' + numConfigs + ' konfigurasjoner  ble  slettet. ';
      return deletedMsg + notDeletedMsg;
    }
    return errorString + ': Error deleting config ' + configObject?.meta.name;
  }
}
