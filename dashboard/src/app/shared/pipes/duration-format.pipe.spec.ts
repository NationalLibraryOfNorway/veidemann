import {DurationFormatPipe} from './duration-format.pipe';

describe('DurationFormatPipe', () => {
  const pipe = new DurationFormatPipe();

  it('formats completed durations using compact units', () => {
    expect(pipe.transform('2026-01-01T00:00:00Z', '2026-01-02T02:03:04Z'))
      .toBe('1 d 2 h 3 min 4 s');
  });

  it('uses now for running executions', () => {
    expect(pipe.transform('2026-01-01T00:00:00Z', '', new Date('2026-01-01T00:01:00Z')))
      .toBe('1 min');
  });

  it('returns an em dash for invalid or missing starts and reversed ranges', () => {
    expect(pipe.transform('')).toBe('—');
    expect(pipe.transform('invalid')).toBe('—');
    expect(pipe.transform('2026-01-02T00:00:00Z', '2026-01-01T00:00:00Z')).toBe('—');
  });
});
