import {BrowserScriptType, ConfigObject, ConfigRef, Kind} from '../../../shared/models';

export type ConfigRelationRole =
  | 'entity'
  | 'crawl-job'
  | 'schedule'
  | 'crawl-config'
  | 'scope-script'
  | 'collection'
  | 'browser-config'
  | 'politeness-config'
  | 'browser-script';

export type ConfigRelationSource = 'direct' | 'selector';

export interface RelatedConfigDescriptor {
  ref: ConfigRef;
  role: ConfigRelationRole;
  source: ConfigRelationSource;
}

/** Returns the direct configuration references stored by a configuration object. */
export function directConfigRefs(configObject: ConfigObject): ConfigRef[] {
  return directConfigDescriptors(configObject).map(descriptor => descriptor.ref);
}

/** Returns direct references with the relationship role used by their owner. */
export function directConfigDescriptors(configObject: ConfigObject): RelatedConfigDescriptor[] {
  const descriptors: RelatedConfigDescriptor[] = [];

  switch (configObject?.kind) {
    case Kind.SEED:
      descriptors.push(descriptor(configObject.seed?.entityRef, 'entity'));
      descriptors.push(...(configObject.seed?.jobRefList ?? []).map(ref => descriptor(ref, 'crawl-job')));
      break;
    case Kind.CRAWLJOB:
      descriptors.push(
        descriptor(configObject.crawlJob?.scheduleRef, 'schedule'),
        descriptor(configObject.crawlJob?.crawlConfigRef, 'crawl-config'),
        descriptor(configObject.crawlJob?.scopeScriptRef, 'scope-script'),
      );
      break;
    case Kind.CRAWLCONFIG:
      descriptors.push(
        descriptor(configObject.crawlConfig?.collectionRef, 'collection'),
        descriptor(configObject.crawlConfig?.browserConfigRef, 'browser-config'),
        descriptor(configObject.crawlConfig?.politenessRef, 'politeness-config'),
      );
      break;
    case Kind.BROWSERCONFIG:
      descriptors.push(...(configObject.browserConfig?.scriptRefList ?? [])
        .map(ref => descriptor(ref, 'browser-script')));
      break;
  }

  return uniqueDescriptors(descriptors);
}

/** Returns direct references plus BrowserScripts selected by a BrowserConfig's label selectors. */
export function relatedConfigRefs(
  configObject: ConfigObject,
  browserScripts: ConfigObject[] = [],
): ConfigRef[] {
  return relatedConfigDescriptors(configObject, browserScripts).map(descriptor => descriptor.ref);
}

/** Returns direct and selector-derived references without losing their origin. */
export function relatedConfigDescriptors(
  configObject: ConfigObject,
  browserScripts: ConfigObject[] = [],
): RelatedConfigDescriptor[] {
  const descriptors = directConfigDescriptors(configObject);
  const selectors = configObject?.browserConfig?.scriptSelectorList ?? [];

  if (configObject?.kind === Kind.BROWSERCONFIG && selectors.length) {
    descriptors.push(...browserScripts
      .filter(script => script?.kind === Kind.BROWSERSCRIPT
        && script.browserScript?.browserScriptType !== BrowserScriptType.SCOPE_CHECK
        && matchesSelectors(script, selectors))
      .map(script => descriptor(ConfigObject.toConfigRef(script), 'browser-script', 'selector')));
  }

  return uniqueDescriptors(descriptors);
}

function descriptor(
  ref: ConfigRef,
  role: ConfigRelationRole,
  source: ConfigRelationSource = 'direct',
): RelatedConfigDescriptor {
  return {ref, role, source};
}

function matchesSelectors(configObject: ConfigObject, selectors: string[]): boolean {
  const labels = configObject.meta?.labelList ?? [];

  return selectors.every(selector => {
    const separatorIndex = selector.indexOf(':');
    const key = (separatorIndex === -1 ? '' : selector.substring(0, separatorIndex)).toLowerCase();
    const rawValue = (separatorIndex === -1 ? selector : selector.substring(separatorIndex + 1)).toLowerCase();
    const prefixMatch = rawValue.endsWith('*');
    const value = prefixMatch ? rawValue.substring(0, rawValue.length - 1) : rawValue;

    return labels.some(label => {
      const labelKey = label.key.toLowerCase();
      const labelValue = label.value.toLowerCase();
      const keyMatches = !key || labelKey === key;
      const valueMatches = !value || (prefixMatch ? labelValue.startsWith(value) : labelValue === value);
      return keyMatches && valueMatches;
    });
  });
}

function uniqueDescriptors(descriptors: RelatedConfigDescriptor[]): RelatedConfigDescriptor[] {
  const seen = new Set<string>();
  return descriptors.filter(({ref}) => {
    if (!ref?.id || ref.kind === Kind.UNDEFINED) {
      return false;
    }
    const key = `${ref.kind}:${ref.id}`;
    if (seen.has(key)) {
      return false;
    }
    seen.add(key);
    return true;
  });
}
