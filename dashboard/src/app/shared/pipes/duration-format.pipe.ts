import {Pipe, PipeTransform} from '@angular/core';
@Pipe({
  name: 'durationFormat',
  standalone: true
})
export class DurationFormatPipe implements PipeTransform {
  transform(startTime: string | Date | null | undefined,
            endTime?: string | Date | null,
            now: Date = new Date()): string {
    const start = this.toDate(startTime);
    const end = endTime ? this.toDate(endTime) : now;
    if (!start || !end || end.getTime() < start.getTime()) {
      return '—';
    }

    let seconds = Math.floor((end.getTime() - start.getTime()) / 1000);
    const days = Math.floor(seconds / 86_400);
    seconds %= 86_400;
    const hours = Math.floor(seconds / 3_600);
    seconds %= 3_600;
    const minutes = Math.floor(seconds / 60);
    seconds %= 60;

    const parts: string[] = [];
    if (days) parts.push(`${days} d`);
    if (hours) parts.push(`${hours} h`);
    if (minutes) parts.push(`${minutes} min`);
    if (seconds || parts.length === 0) parts.push(`${seconds} s`);
    return parts.join(' ');
  }

  private toDate(value: string | Date | null | undefined): Date | null {
    if (!value) return null;
    const date = value instanceof Date ? value : new Date(value);
    return Number.isNaN(date.getTime()) ? null : date;
  }
}
