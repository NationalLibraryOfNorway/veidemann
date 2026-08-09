import {LabelLinkConfig} from '../../../app.config';
import {Label} from '../../../shared/models';

export interface ResolvedLabelLink {
  href: string;
  text: string;
}

export function resolveLabelLink(
  links: Record<string, LabelLinkConfig>,
  label: Label,
): ResolvedLabelLink | null {
  const link = links?.[label.key];
  if (!link?.text?.trim() || !link.urlTemplate?.includes('{value}')) {
    return null;
  }

  const href = link.urlTemplate.replaceAll('{value}', encodeURIComponent(label.value));
  try {
    const url = new URL(href);
    if (url.protocol !== 'http:' && url.protocol !== 'https:') {
      return null;
    }
  } catch {
    return null;
  }

  return {href, text: link.text};
}
