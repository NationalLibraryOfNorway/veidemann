import {FileSizePipe} from './filesize.pipe';

describe('FileSizePipe', () => {
  const pipe = new FileSizePipe();

  it('formats a file size', () => {
    expect(pipe.transform(1024)).toBe('1.02 kB');
  });

  it('formats an array of file sizes', () => {
    expect(pipe.transform([1000, 2000])).toEqual(['1 kB', '2 kB']);
  });

  it('passes options to filesize', () => {
    expect(pipe.transform(1024, {standard: 'iec'})).toBe('1 KiB');
  });
});
