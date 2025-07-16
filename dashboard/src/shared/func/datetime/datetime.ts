import * as timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb.js';
import {
  addMilliseconds,
  endOfDay,
  formatDuration as formatFnsDuration,
  isValid,
  parseISO,
  set,
  startOfDay,
  differenceInMilliseconds,
} from 'date-fns';
import { enUS } from 'date-fns/locale';

export class DateTime {
  static dateToUtc(dateString: string, startOfDayFlag: boolean): string | null {
    const date = parseISO(dateString);
    if (isValid(date)) {
      const baseDate = new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate()));
      return (startOfDayFlag ? startOfDay(baseDate) : endOfDay(baseDate)).toISOString();
    } else {
      return null;
    }
  }

  static adjustTime(timestamp: string): Date {
    const date = parseISO(timestamp);
    if (!isValid(date)) return null;
    return set(new Date(date.toISOString()), { hours: 12, minutes: 0, seconds: 0 });
  }
}

export function isValidDate(d: Date): boolean {
  return isValid(d);
}

export function fromTimestampProto(proto: any): string {
  if (proto) {
    const ms = new Date(proto.getSeconds() * 1e3 + proto.getNanos() / 1e6);
    return ms.toISOString();
  } else {
    return '';
  }
}

export function toTimestampProto(timestamp: string): any {
  if (timestamp) {
    const date = new Date(timestamp);
    const timestampProto = new timestamp_pb.Timestamp();
    const seconds = date.getTime() / 1000;
    timestampProto.setSeconds(~(~seconds));
    return timestampProto;
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

const timeUnitMap: { [key: string]: number } = {
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

export function guessTimeZone(): string {
  return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
}
