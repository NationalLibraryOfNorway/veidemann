import {HttpErrorResponse} from '@angular/common/http';
import {provideZonelessChangeDetection} from '@angular/core';
import {TestBed} from '@angular/core/testing';
import {MatSnackBar} from '@angular/material/snack-bar';

import {Code, ConnectError} from '@connectrpc/connect';
import {Subject} from 'rxjs';

import {SnackBarService} from './snack-bar.service';

interface SnackBarRefStub {
  readonly dismissed: Subject<void>;
  readonly dismiss: ReturnType<typeof vi.fn>;
  readonly afterDismissed: () => Subject<void>;
}

describe('SnackBarService', () => {
  let service: SnackBarService;
  let open: ReturnType<typeof vi.fn>;
  let refs: SnackBarRefStub[];
  let consoleError: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(0);
    refs = [];
    open = vi.fn(() => {
      const dismissed = new Subject<void>();
      const ref: SnackBarRefStub = {
        dismissed,
        dismiss: vi.fn(),
        afterDismissed: () => dismissed,
      };
      refs.push(ref);
      return ref;
    });
    consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);

    TestBed.configureTestingModule({
      providers: [
        provideZonelessChangeDetection(),
        SnackBarService,
        {provide: MatSnackBar, useValue: {open}},
      ],
    });
    service = TestBed.inject(SnackBarService);
  });

  afterEach(() => {
    consoleError.mockRestore();
    vi.useRealTimers();
  });

  it('logs and opens an isolated error as a polite error snackbar', () => {
    const error = new Error('Something failed');

    service.openError(error);

    expect(consoleError).toHaveBeenCalledOnce();
    expect(consoleError).toHaveBeenCalledWith(error);
    expect(open).toHaveBeenCalledWith('Error: Something failed', undefined, {
      duration: 0,
      panelClass: ['error-snackbar'],
      politeness: 'polite',
    });
  });

  it('normalizes ConnectRPC and HTTP response messages', () => {
    service.openError(new ConnectError('Missing crawl', Code.NotFound));
    refs[0].dismissed.next();
    service.openError(new HttpErrorResponse({
      status: 400,
      error: {message: '  Invalid\nconfiguration  '},
    }));

    expect(open.mock.calls[0][0]).toBe('Error: [not_found] Missing crawl');
    expect(open.mock.calls[1][0]).toBe('Error: Invalid configuration');
  });

  it('uses a generic message for malformed or empty errors', () => {
    service.openError({message: 'Do not expose arbitrary objects'});
    refs[0].dismissed.next();
    service.openError(new Error('  '));

    expect(open.mock.calls[0][0]).toBe('Error: Something went wrong. Try again.');
    expect(open.mock.calls[1][0]).toBe('Error: Something went wrong. Try again.');
  });

  it('bounds displayed error text to 240 characters without truncating console details', () => {
    const error = new Error('x'.repeat(300));

    service.openError(error);

    const displayed = open.mock.calls[0][0] as string;
    expect(displayed).toHaveLength(240);
    expect(displayed.endsWith('…')).toBe(true);
    expect(consoleError).toHaveBeenCalledWith(error);
  });

  it('does not collapse distinct errors whose displayed prefixes truncate identically', () => {
    service.openError(new Error(`${'x'.repeat(300)} first`));
    service.openError(new Error(`${'x'.repeat(300)} second`));

    refs[0].dismissed.next();

    expect(open).toHaveBeenCalledTimes(2);
  });

  it('shows an isolated error for ten seconds', () => {
    service.openError(new Error('First'));

    vi.advanceTimersByTime(9_999);
    expect(refs[0].dismiss).not.toHaveBeenCalled();

    vi.advanceTimersByTime(1);
    expect(refs[0].dismiss).toHaveBeenCalledOnce();
  });

  it('shortens an active error and gives queued burst errors four seconds each', () => {
    service.openError(new Error('First'));
    vi.advanceTimersByTime(2_000);

    service.openError(new Error('Second'));

    expect(open).toHaveBeenCalledOnce();
    vi.advanceTimersByTime(1_999);
    expect(refs[0].dismiss).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(refs[0].dismiss).toHaveBeenCalledOnce();

    refs[0].dismissed.next();
    expect(open.mock.calls[1][0]).toBe('Error: Second');
    vi.advanceTimersByTime(3_999);
    expect(refs[1].dismiss).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(refs[1].dismiss).toHaveBeenCalledOnce();
  });

  it('advances immediately when a late burst follows an error shown for four seconds', () => {
    service.openError(new Error('First'));
    vi.advanceTimersByTime(6_000);

    service.openError(new Error('Second'));

    expect(refs[0].dismiss).toHaveBeenCalledOnce();
  });

  it('collapses adjacent duplicate errors without changing the isolated timer', () => {
    service.openError(new Error('Repeated'));
    vi.advanceTimersByTime(5_000);

    service.openError(new Error('Repeated'));

    expect(open).toHaveBeenCalledOnce();
    vi.advanceTimersByTime(4_999);
    expect(refs[0].dismiss).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(refs[0].dismiss).toHaveBeenCalledOnce();
  });

  it('returns to a ten-second duration after a burst drains', () => {
    service.openError(new Error('First'));
    service.openError(new Error('Second'));
    refs[0].dismissed.next();
    refs[1].dismissed.next();

    service.openError(new Error('Third'));

    vi.advanceTimersByTime(9_999);
    expect(refs[2].dismiss).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(refs[2].dismiss).toHaveBeenCalledOnce();
  });

  it('preserves the existing eight-second success duration', () => {
    service.openSnackBar('Saved');

    vi.advanceTimersByTime(7_999);
    expect(refs[0].dismiss).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(refs[0].dismiss).toHaveBeenCalledOnce();
  });
});
