import {Pipe, PipeTransform} from '@angular/core';
import {filesize, FilesizeOptions} from 'filesize';

@Pipe({
  name: 'filesize',
  standalone: true
})
export class FileSizePipe implements PipeTransform {
  private static transformOne(value: number, options?: FilesizeOptions): string {
    return filesize(value, options);
  }

  transform(value: number, options?: FilesizeOptions): string;
  transform(value: number[], options?: FilesizeOptions): string[];
  transform(value: number | number[], options?: FilesizeOptions): string | string[] {
    if (Array.isArray(value)) {
      return value.map(val => FileSizePipe.transformOne(val, options));
    }

    return FileSizePipe.transformOne(value, options);
  }
}
