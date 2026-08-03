import {Kind} from '../../../shared/models';
import {configKindFromPath} from './kind';

describe('configKindFromPath', () => {
  it('maps configuration paths to kinds', () => {
    expect(configKindFromPath('seed')).toBe(Kind.SEED);
    expect(configKindFromPath('crawljobs')).toBe(Kind.CRAWLJOB);
  });

  it('maps missing and unknown paths to undefined', () => {
    expect(configKindFromPath(null)).toBe(Kind.UNDEFINED);
    expect(configKindFromPath('unknown')).toBe(Kind.UNDEFINED);
  });
});
