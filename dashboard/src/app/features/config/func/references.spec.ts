import {
  BrowserConfig,
  BrowserScript,
  BrowserScriptType,
  ConfigObject,
  ConfigRef,
  CrawlConfig,
  CrawlJob,
  Kind,
  Label,
  Meta,
  Seed,
} from '../../../shared/models';
import {directConfigRefs, relatedConfigRefs} from './references';

describe('directConfigRefs', () => {
  it('returns crawljob references in display order and ignores empty references', () => {
    const config = new ConfigObject({
      kind: Kind.CRAWLJOB,
      crawlJob: new CrawlJob({
        scheduleRef: new ConfigRef({kind: Kind.CRAWLSCHEDULECONFIG}),
        crawlConfigRef: new ConfigRef({kind: Kind.CRAWLCONFIG, id: 'crawl-config'}),
        scopeScriptRef: new ConfigRef({kind: Kind.BROWSERSCRIPT, id: 'scope-script'}),
      }),
    });

    expect(directConfigRefs(config)).toEqual([
      expect.objectContaining({kind: Kind.CRAWLCONFIG, id: 'crawl-config'}),
      expect.objectContaining({kind: Kind.BROWSERSCRIPT, id: 'scope-script'}),
    ]);
  });

  it('returns seed and crawlconfig references without recursively expanding them', () => {
    const seed = new ConfigObject({
      kind: Kind.SEED,
      seed: new Seed({
        entityRef: new ConfigRef({kind: Kind.CRAWLENTITY, id: 'entity'}),
        jobRefList: [new ConfigRef({kind: Kind.CRAWLJOB, id: 'job'})],
      }),
    });
    const crawlConfig = new ConfigObject({
      kind: Kind.CRAWLCONFIG,
      crawlConfig: new CrawlConfig({
        collectionRef: new ConfigRef({kind: Kind.COLLECTION, id: 'collection'}),
        browserConfigRef: new ConfigRef({kind: Kind.BROWSERCONFIG, id: 'browser'}),
        politenessRef: new ConfigRef({kind: Kind.POLITENESSCONFIG, id: 'politeness'}),
      }),
    });

    expect(directConfigRefs(seed).map(ref => ref.id)).toEqual(['entity', 'job']);
    expect(directConfigRefs(crawlConfig).map(ref => ref.id)).toEqual(['collection', 'browser', 'politeness']);
  });

  it('deduplicates browser script references by kind and id', () => {
    const script = new ConfigRef({kind: Kind.BROWSERSCRIPT, id: 'script'});
    const config = new ConfigObject({
      kind: Kind.BROWSERCONFIG,
      browserConfig: new BrowserConfig({scriptRefList: [script, script]}),
    });

    expect(directConfigRefs(config)).toEqual([expect.objectContaining({id: 'script'})]);
  });

  it('appends BrowserScripts matching every selector in option order', () => {
    const config = new ConfigObject({
      kind: Kind.BROWSERCONFIG,
      browserConfig: new BrowserConfig({
        scriptRefList: [new ConfigRef({kind: Kind.BROWSERSCRIPT, id: 'explicit'})],
        scriptSelectorList: ['CATEGORY:News', 'environment:pro*'],
      }),
    });
    const browserScripts = [
      browserScript('implicit', [new Label({key: 'category', value: 'NEWS'}), new Label({key: 'environment', value: 'production'})]),
      browserScript('missing-selector', [new Label({key: 'category', value: 'news'})]),
      browserScript('wrong-prefix', [new Label({key: 'category', value: 'news'}), new Label({key: 'environment', value: 'staging'})]),
    ];

    expect(relatedConfigRefs(config, browserScripts).map(ref => ref.id)).toEqual(['explicit', 'implicit']);
  });

  it('excludes scope scripts from BrowserConfig selector results', () => {
    const config = new ConfigObject({
      kind: Kind.BROWSERCONFIG,
      browserConfig: new BrowserConfig({scriptSelectorList: ['profile:default']}),
    });
    const labels = [new Label({key: 'profile', value: 'default'})];
    const browserScripts = [
      browserScript('on-load', labels, BrowserScriptType.ON_LOAD),
      browserScript('scope', labels, BrowserScriptType.SCOPE_CHECK),
    ];

    expect(relatedConfigRefs(config, browserScripts).map(ref => ref.id)).toEqual(['on-load']);
  });

  it('supports key-only and value-only selectors', () => {
    const config = new ConfigObject({
      kind: Kind.BROWSERCONFIG,
      browserConfig: new BrowserConfig({scriptSelectorList: ['purpose:', ':DEFAULT']}),
    });
    const browserScripts = [
      browserScript('matching', [new Label({key: 'purpose', value: 'extract'}), new Label({key: 'profile', value: 'default'})]),
      browserScript('wrong-value', [new Label({key: 'purpose', value: 'extract'})]),
      browserScript('unlabelled'),
    ];

    expect(relatedConfigRefs(config, browserScripts).map(ref => ref.id)).toEqual(['matching']);
  });

  it('deduplicates scripts selected explicitly and implicitly', () => {
    const explicit = new ConfigRef({kind: Kind.BROWSERSCRIPT, id: 'both'});
    const config = new ConfigObject({
      kind: Kind.BROWSERCONFIG,
      browserConfig: new BrowserConfig({scriptRefList: [explicit, explicit], scriptSelectorList: ['active:true']}),
    });
    const browserScripts = [
      browserScript('both', [new Label({key: 'active', value: 'true'})]),
      browserScript('implicit', [new Label({key: 'active', value: 'true'})]),
    ];

    expect(relatedConfigRefs(config, browserScripts).map(ref => ref.id)).toEqual(['both', 'implicit']);
  });
});

function browserScript(
  id: string,
  labelList: Label[] = [],
  browserScriptType = BrowserScriptType.UNDEFINED,
): ConfigObject {
  return new ConfigObject({
    id,
    kind: Kind.BROWSERSCRIPT,
    meta: new Meta({labelList}),
    browserScript: new BrowserScript({browserScriptType}),
  });
}
