import {Pipe, PipeTransform} from '@angular/core';
import {timeToDuration} from '../../shared/func';

@Pipe({
    name: 'durationFormat',
    standalone: true
})
export class DurationFormatPipe implements PipeTransform {

  transform(time: number, unit: string): string {
    return timeToDuration(time, unit);
  }
}
