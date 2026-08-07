import {Directive} from '@angular/core';

import {QueryComponent} from '../../../shared/components';
import {isValidDate} from '../../../shared/func';

interface StartTimeDateRangeQuery {
  startTimeFrom: string;
  startTimeTo: string;
}

@Directive()
export abstract class StartTimeDateRangeQueryComponent<
  T extends StartTimeDateRangeQuery
> extends QueryComponent<T> {

  protected override updateForm(): void {
    this.form.patchValue({
      ...this.query,
      startTimeFrom: this.toFormDate(this.query.startTimeFrom),
      startTimeTo: this.toFormDate(this.query.startTimeTo, -1),
    }, {emitEvent: false});
    this.form.markAsPristine();
    this.form.markAsUntouched();
  }

  protected override transform(query: T): T {
    return {
      ...query,
      startTimeFrom: this.toBoundary(query.startTimeFrom),
      startTimeTo: this.toBoundary(query.startTimeTo, 1),
    };
  }

  override onQuery(query: T): void {
    if (this.form.valid) {
      super.onQuery(query);
    }
  }

  private toBoundary(value: string | Date | null | undefined, dayOffset = 0): string {
    if (!value) {
      return '';
    }
    const date = new Date(value);
    if (!isValidDate(date)) {
      return '';
    }
    date.setHours(0, 0, 0, 0);
    date.setDate(date.getDate() + dayOffset);
    return date.toISOString();
  }

  private toFormDate(value: string | undefined, dayOffset = 0): Date | null {
    if (!value) {
      return null;
    }
    const date = new Date(value);
    if (!isValidDate(date)) {
      return null;
    }
    date.setDate(date.getDate() + dayOffset);
    date.setHours(0, 0, 0, 0);
    return date;
  }
}
