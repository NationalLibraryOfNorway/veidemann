import {Kind} from '../../../shared/models';
import {configKindFromPath} from './kind';
import {configKindIcon} from './config-kind-icon';

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

describe('configKindIcon', () => {
  it.each([
    [Kind.CRAWLENTITY, 'business'],
    [Kind.SEED, 'link'],
    [Kind.CRAWLJOB, 'work'],
    [Kind.CRAWLSCHEDULECONFIG, 'schedule'],
    [Kind.CRAWLCONFIG, 'settings_system_daydream'],
    [Kind.COLLECTION, 'collections_bookmark'],
    [Kind.BROWSERCONFIG, 'web'],
    [Kind.BROWSERSCRIPT, 'web_asset'],
    [Kind.POLITENESSCONFIG, 'sentiment_very_satisfied'],
    [Kind.CRAWLHOSTGROUPCONFIG, 'group_work'],
    [Kind.ROLEMAPPING, 'people'],
  ])('maps kind %s to %s', (kind, icon) => {
    expect(configKindIcon(kind as Kind)).toBe(icon);
  });

  it('uses settings for an unknown kind', () => {
    expect(configKindIcon(Kind.UNDEFINED)).toBe('settings');
  });
});
