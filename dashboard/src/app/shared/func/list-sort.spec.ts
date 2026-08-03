import {compareListValues} from './list-sort';

describe('compareListValues', () => {
  it('sorts strings case-insensitively and keeps nulls last in both directions', () => {
    const values = ['beta', null, 'Alpha'];
    expect([...values].sort((a, b) => compareListValues(a, b, 'asc'))).toEqual(['Alpha', 'beta', null]);
    expect([...values].sort((a, b) => compareListValues(a, b, 'desc'))).toEqual(['beta', 'Alpha', null]);
  });

  it('compares numbers and dates by value', () => {
    expect(compareListValues(2, 10, 'asc', 'number')).toBeLessThan(0);
    expect(compareListValues('2025-01-01', '2024-01-01', 'asc', 'date')).toBeGreaterThan(0);
    expect(compareListValues('invalid', '2024-01-01', 'desc', 'date')).toBeGreaterThan(0);
  });
});
