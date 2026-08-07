import {ConfigObject, ConfigRef, Kind} from '../../../shared/models';

/** Returns the direct configuration references stored by a configuration object. */
export function directConfigRefs(configObject: ConfigObject): ConfigRef[] {
  const refs: ConfigRef[] = [];

  switch (configObject?.kind) {
    case Kind.SEED:
      refs.push(configObject.seed?.entityRef, ...(configObject.seed?.jobRefList ?? []));
      break;
    case Kind.CRAWLJOB:
      refs.push(
        configObject.crawlJob?.scheduleRef,
        configObject.crawlJob?.crawlConfigRef,
        configObject.crawlJob?.scopeScriptRef,
      );
      break;
    case Kind.CRAWLCONFIG:
      refs.push(
        configObject.crawlConfig?.collectionRef,
        configObject.crawlConfig?.browserConfigRef,
        configObject.crawlConfig?.politenessRef,
      );
      break;
    case Kind.BROWSERCONFIG:
      refs.push(...(configObject.browserConfig?.scriptRefList ?? []));
      break;
  }

  return uniqueConfigRefs(refs);
}

/** Returns direct references plus BrowserScripts selected by a BrowserConfig's label selectors. */
export function relatedConfigRefs(
  configObject: ConfigObject,
  browserScripts: ConfigObject[] = [],
): ConfigRef[] {
  const refs = directConfigRefs(configObject);
  const selectors = configObject?.browserConfig?.scriptSelectorList ?? [];

  if (configObject?.kind === Kind.BROWSERCONFIG && selectors.length) {
    refs.push(...browserScripts
      .filter(script => script?.kind === Kind.BROWSERSCRIPT && matchesSelectors(script, selectors))
      .map(ConfigObject.toConfigRef));
  }

  return uniqueConfigRefs(refs);
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

function uniqueConfigRefs(refs: ConfigRef[]): ConfigRef[] {
  const seen = new Set<string>();
  return refs.filter(ref => {
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
