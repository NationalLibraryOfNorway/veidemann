import {TestBed} from '@angular/core/testing';

import {ConnectError} from '@connectrpc/connect';

import {ConfigObject, Meta} from '../shared/models';
import {ReferrerError} from '../shared/error';
import {ApplicationErrorHandler} from './error.handler';
import {SnackBarService} from './snack-bar/snack-bar.service';

describe('ApplicationErrorHandler', () => {
  let handler: ApplicationErrorHandler;
  let openError: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    openError = vi.fn();
    TestBed.configureTestingModule({
      providers: [
        ApplicationErrorHandler,
        {provide: SnackBarService, useValue: {openError}},
      ],
    });
    handler = TestBed.inject(ApplicationErrorHandler);
  });

  it('forwards every handled error to the snackbar service', () => {
    const error = new Error('Something failed');

    handler.handleError(error);

    expect(openError).toHaveBeenCalledOnce();
    expect(openError).toHaveBeenCalledWith(error);
  });

  it('turns recognized delete reference failures into a contextual error', () => {
    const configObject = new ConfigObject({meta: new Meta({name: 'Referenced config'})});

    handler.handleDeleteError(
      new ConnectError('Cannot delete because there are references'),
      configObject,
    );

    const contextualError = openError.mock.calls[0][0];
    expect(contextualError).toBeInstanceOf(ReferrerError);
    expect(contextualError.message).toContain('Referenced config');
  });

  it('forwards unrecognized delete failures instead of suppressing them', () => {
    const error = new ConnectError('Storage unavailable');

    handler.handleDeleteError(error, new ConfigObject());

    expect(openError).toHaveBeenCalledOnce();
    expect(openError).toHaveBeenCalledWith(error);
  });
});
