import { timestampDate, timestampFromDate, type Timestamp } from '@bufbuild/protobuf/wkt';
import {differenceInMilliseconds, isValid, parseISO, set} from 'date-fns';

export class DateTime {

  static dateToUtc(value: string | Date, startOfDay: boolean): string | null {
    if (!value) return null;

    let d: Date;
    if (value instanceof Date) {
      d = new Date(Date.UTC(value.getFullYear(), value.getMonth(), value.getDate()));
    } else if (/^\d{1,2}\.\d{1,2}\.\d{4}$/.test(value.trim())) {
      const [day, month, year] = value.split('.').map(n => +n);
      d = new Date(Date.UTC(year, month - 1, day));
      if (d.getUTCFullYear() !== year || d.getUTCMonth() !== month - 1 || d.getUTCDate() !== day) return null;
    } else {
      d = parseISO(value);
      if (isNaN(d.getTime())) return null;
      if (/^\d{4}-\d{2}-\d{2}$/.test(value)) {
        d = new Date(Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate()));
      }
    }

    d.setUTCHours(startOfDay ? 0 : 23, startOfDay ? 0 : 59, startOfDay ? 0 : 59, startOfDay ? 0 : 999);
    return d.toISOString();
  }


  static adjustTime(timestamp: string): Date | null {
    // TODO debug
    console.log('adjustTime', timestamp);
    if (!timestamp) return null;
    const date = parseISO(timestamp);
    if (!isValid(date)) return null;
    return set(new Date(date.toISOString()), { hours: 12, minutes: 0, seconds: 0 });
  }
}

export function isValidDate(d: Date): boolean {
  return isValid(d);
}

export function fromTimestampProto(proto?: Timestamp): string {
  if (proto) {
    return timestampDate(proto).toISOString();
  } else {
    return '';
  }
}

export function toTimestampProto(timestamp: string): Timestamp | undefined {
  if (timestamp) {
    return timestampFromDate(new Date(timestamp));
  } else {
    return undefined;
  }
}

export function durationBetweenDates(startTime: string, endTime: string): string {
  const start = parseISO(startTime);
  const end = endTime === '' ? new Date() : parseISO(endTime);
  const diffMs = differenceInMilliseconds(end, start);
  return formatDuration(diffMs);
}

export function formatDuration(ms: number): string {
  const seconds = Math.floor(ms / 1000);
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;

  const parts: string[] = [];
  if (days > 0) parts.push(`${days}days`);
  if (hours > 0) parts.push(`${hours}hours`);
  if (minutes > 0) parts.push(`${minutes}min`);
  if (secs > 0 || parts.length === 0) parts.push(`${secs}s`);

  return parts.join(':');
}

const timeUnitMap: Record<string, number> = {
  ms: 1,
  s: 1000,
  m: 60000,
  h: 3600000,
  d: 86400000,
  w: 604800000,
  M: 2629800000, // average month
  y: 31557600000, // average year
};

export function timeToDuration(time: number, unit: string): string {
  const ms = time * (timeUnitMap[unit] || 1);
  return formatDuration(ms);
}
