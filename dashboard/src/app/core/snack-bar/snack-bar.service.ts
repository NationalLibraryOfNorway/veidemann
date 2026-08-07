import {HttpErrorResponse} from '@angular/common/http';
import {Injectable, inject} from '@angular/core';
import {MatSnackBar} from '@angular/material/snack-bar';

import {ConnectError} from '@connectrpc/connect';

const SUCCESS_DURATION = 8_000;
const ISOLATED_ERROR_DURATION = 10_000;
const BURST_ERROR_DURATION = 4_000;
const MAX_ERROR_MESSAGE_LENGTH = 240;

type NotificationKind = 'message' | 'error';

interface QueuedNotification {
  readonly kind: NotificationKind;
  readonly message: string;
  readonly dedupeKey?: string;
  readonly action?: string;
  duration: number;
}

interface NormalizedError {
  readonly message: string;
  readonly dedupeKey: string;
}


@Injectable({providedIn: 'root'})
export class SnackBarService {
  private snackBar = inject(MatSnackBar);

  private readonly queue: QueuedNotification[] = [];
  private activeNotification: QueuedNotification | null = null;
  private activeRef: ReturnType<MatSnackBar['open']> | null = null;
  private activeOpenedAt = 0;
  private activeTimer: ReturnType<typeof setTimeout> | null = null;

  openSnackBar(message: string, action?: string, duration?: number): void {
    this.enqueue({
      kind: 'message',
      message,
      action,
      duration: duration ?? (action ? 0 : SUCCESS_DURATION),
    });
  }

  openError(error: unknown): void {
    console.error(error);
    const normalized = this.normalizeError(error);
    this.enqueue({
      kind: 'error',
      ...normalized,
      duration: ISOLATED_ERROR_DURATION,
    });
  }

  private enqueue(notification: QueuedNotification): void {
    if (this.isAdjacentDuplicateError(notification)) {
      return;
    }

    if (notification.kind === 'error' && this.hasPendingError()) {
      notification.duration = BURST_ERROR_DURATION;
      for (const pending of this.queue) {
        if (pending.kind === 'error') {
          pending.duration = BURST_ERROR_DURATION;
        }
      }
      this.shortenActiveError();
    }

    this.queue.push(notification);
    this.showNext();
  }

  private hasPendingError(): boolean {
    return this.activeNotification?.kind === 'error'
      || this.queue.some(notification => notification.kind === 'error');
  }

  private isAdjacentDuplicateError(notification: QueuedNotification): boolean {
    if (notification.kind !== 'error') {
      return false;
    }

    const tail = this.queue.at(-1);
    if (tail) {
      return tail.kind === 'error' && tail.dedupeKey === notification.dedupeKey;
    }
    return this.activeNotification?.kind === 'error'
      && this.activeNotification.dedupeKey === notification.dedupeKey;
  }

  private shortenActiveError(): void {
    if (this.activeNotification?.kind !== 'error' || !this.activeRef) {
      return;
    }

    this.activeNotification.duration = BURST_ERROR_DURATION;
    const elapsed = Date.now() - this.activeOpenedAt;
    const remaining = Math.max(0, BURST_ERROR_DURATION - elapsed);
    this.scheduleDismissal(this.activeRef, remaining);
  }

  private showNext(): void {
    if (this.activeRef || this.queue.length === 0) {
      return;
    }

    const notification = this.queue.shift()!;
    const ref = this.snackBar.open(notification.message, notification.action, {
      duration: 0,
      panelClass: notification.kind === 'error' ? ['error-snackbar'] : undefined,
      politeness: 'polite',
    });

    this.activeNotification = notification;
    this.activeRef = ref;
    this.activeOpenedAt = Date.now();
    this.scheduleDismissal(ref, notification.duration);

    ref.afterDismissed().subscribe(() => {
      if (this.activeRef !== ref) {
        return;
      }
      this.clearActiveTimer();
      this.activeNotification = null;
      this.activeRef = null;
      this.activeOpenedAt = 0;
      this.showNext();
    });
  }

  private scheduleDismissal(ref: ReturnType<MatSnackBar['open']>, delay: number): void {
    this.clearActiveTimer();
    if (delay <= 0) {
      ref.dismiss();
      return;
    }
    this.activeTimer = setTimeout(() => ref.dismiss(), delay);
  }

  private clearActiveTimer(): void {
    if (this.activeTimer !== null) {
      clearTimeout(this.activeTimer);
      this.activeTimer = null;
    }
  }

  private normalizeError(error: unknown): NormalizedError {
    let detail = '';

    if (error instanceof ConnectError) {
      detail = error.message;
    } else if (error instanceof HttpErrorResponse) {
      if (typeof error.error === 'string') {
        detail = error.error;
      } else if (this.hasStringMessage(error.error)) {
        detail = error.error.message;
      } else {
        detail = error.message;
      }
    } else if (error instanceof Error) {
      detail = error.message;
    }

    detail = detail.replace(/\s+/g, ' ').trim()
      || $localize`:@@snackBarMessage.genericError:Something went wrong. Try again.`;
    const dedupeKey = `${$localize`:@@snackBarMessage.errorPrefix:Error:`} ${detail}`;
    const message = dedupeKey.length <= MAX_ERROR_MESSAGE_LENGTH
      ? dedupeKey
      : `${dedupeKey.slice(0, MAX_ERROR_MESSAGE_LENGTH - 1).trimEnd()}…`;

    return {message, dedupeKey};
  }

  private hasStringMessage(value: unknown): value is {message: string} {
    return typeof value === 'object'
      && value !== null
      && 'message' in value
      && typeof value.message === 'string';
  }
}
