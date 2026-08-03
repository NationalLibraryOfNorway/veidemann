import {SortDirection} from '@angular/material/sort';

export type ListSortValue = string | number | Date | null | undefined;

/** Compare list values with case-insensitive strings and nullish/invalid values last. */
export function compareListValues(
  left: ListSortValue,
  right: ListSortValue,
  direction: SortDirection,
  type: 'string' | 'number' | 'date' = 'string'
): number {
  const leftValue = normalize(left, type);
  const rightValue = normalize(right, type);
  const leftNull = leftValue === null;
  const rightNull = rightValue === null;
  if (leftNull || rightNull) {
    return leftNull === rightNull ? 0 : leftNull ? 1 : -1;
  }

  const comparison = type === 'string'
    ? String(leftValue).localeCompare(String(rightValue), undefined, {sensitivity: 'base'})
    : Number(leftValue) - Number(rightValue);
  return direction === 'desc' ? -comparison : comparison;
}

function normalize(value: ListSortValue, type: 'string' | 'number' | 'date'): string | number | null {
  if (value === null || value === undefined || value === '') {
    return null;
  }
  if (type === 'date') {
    const time = value instanceof Date ? value.getTime() : new Date(value).getTime();
    return Number.isNaN(time) ? null : time;
  }
  if (type === 'number') {
    const number = Number(value);
    return Number.isNaN(number) ? null : number;
  }
  return String(value);
}
