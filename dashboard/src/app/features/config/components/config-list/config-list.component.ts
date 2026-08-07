import {ChangeDetectionStrategy, Component, EventEmitter, inject, Output} from '@angular/core';
import {ConfigObject, Label} from '../../../../shared/models/config';
import {CONFIG_LIST_IMPORTS, ConfigListBaseComponent} from './config-list-base';
import {isEmojiLabel, LabelDisplayComponent} from '../../../../shared/components';
import {AppConfig, LabelLinkConfig} from '../../../../app.config';

interface ResolvedLabelLink {
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

@Component({
    selector: 'app-config-list',
    templateUrl: './config-list.component.html',
    styleUrls: [
        './config-list-base.scss',
        './config-list.component.scss',
    ],
    changeDetection: ChangeDetectionStrategy.OnPush,
    imports: [...CONFIG_LIST_IMPORTS, LabelDisplayComponent],
    standalone: true,
})

export class ConfigListComponent extends ConfigListBaseComponent<ConfigObject> {
  private appConfig = inject(AppConfig);
  @Output() readonly labelClick = new EventEmitter<Label>();
  protected readonly isEmojiLabel = isEmojiLabel;
  protected override readonly autoSelectAppendedRows = true;

  override isDisabled(config: ConfigObject): boolean {
    return config?.crawlJob?.disabled || config?.seed?.disabled;
  }

  labelSearchAriaLabel(label: Label): string {
    const labelText = `${label.key}:${label.value}`;
    return $localize`:@@configurationListLabelSearchAriaLabel:Search for exact label ${labelText}:LABEL:`;
  }

  onLabelClick(label: Label, event: Event): void {
    event.preventDefault();
    event.stopPropagation();
    this.labelClick.emit(label);
  }

  externalLabelLink(label: Label): ResolvedLabelLink | null {
    return resolveLabelLink(this.appConfig.labelLinks, label);
  }

  configTypePluralLabel(): string {
    switch (this.selectedRows()[0]?.kind) {
      case this.Kind.CRAWLENTITY:
        return $localize`:@@configurationListTypeEntities:entities`;
      case this.Kind.SEED:
        return $localize`:@@configurationListTypeSeeds:seeds`;
      case this.Kind.CRAWLJOB:
        return $localize`:@@configurationListTypeCrawlJobs:crawl jobs`;
      case this.Kind.CRAWLSCHEDULECONFIG:
        return $localize`:@@configurationListTypeSchedules:schedules`;
      case this.Kind.CRAWLCONFIG:
        return $localize`:@@configurationListTypeCrawlConfigurations:crawl configurations`;
      case this.Kind.COLLECTION:
        return $localize`:@@configurationListTypeCollections:collections`;
      case this.Kind.BROWSERCONFIG:
        return $localize`:@@configurationListTypeBrowserConfigurations:browser configurations`;
      case this.Kind.BROWSERSCRIPT:
        return $localize`:@@configurationListTypeBrowserScripts:browser scripts`;
      case this.Kind.POLITENESSCONFIG:
        return $localize`:@@configurationListTypePolitenessConfigurations:politeness configurations`;
      case this.Kind.CRAWLHOSTGROUPCONFIG:
        return $localize`:@@configurationListTypeCrawlHostGroups:crawl host groups`;
      default:
        return $localize`:@@configurationListTypeConfigurations:configurations`;
    }
  }
}
