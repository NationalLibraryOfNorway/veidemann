import {Kind} from '../../../shared/models';

/** Returns the Material icon used to represent a configuration kind. */
export function configKindIcon(kind: Kind): string {
  switch (kind) {
    case Kind.CRAWLENTITY:
      return 'business';
    case Kind.SEED:
      return 'link';
    case Kind.CRAWLJOB:
      return 'work';
    case Kind.CRAWLSCHEDULECONFIG:
      return 'schedule';
    case Kind.CRAWLCONFIG:
      return 'settings_system_daydream';
    case Kind.COLLECTION:
      return 'collections_bookmark';
    case Kind.BROWSERCONFIG:
      return 'web';
    case Kind.BROWSERSCRIPT:
      return 'web_asset';
    case Kind.POLITENESSCONFIG:
      return 'sentiment_very_satisfied';
    case Kind.CRAWLHOSTGROUPCONFIG:
      return 'group_work';
    case Kind.ROLEMAPPING:
      return 'people';
    default:
      return 'settings';
  }
}
