import {ChangeDetectionStrategy, Component, EventEmitter, Input, OnChanges, Output} from '@angular/core';
import {MatChipsModule} from '@angular/material/chips';
import {MatIcon} from '@angular/material/icon';
import {MatTooltipModule} from '@angular/material/tooltip';

import {
  isEmojiLabel,
  LabelDisplayComponent,
} from '../../../../shared/components/label-display/label-display.component';
import {ConfigQuery} from '../../../../shared/func';
import {ConfigObject, Kind} from '../../../../shared/models';
import {parseConfigSearchTerm} from '../../func/query';
import type {ConfigLabelSelector} from '../../func/query';
import type {ConfigOptions} from '../../func/options';
import {configKindIcon} from '../../func/config-kind-icon';

export interface ActiveConfigFilterChip {
  key: 'entityId' | 'crawlJobIdList' | 'scriptIdList' | 'labelSelector';
  value: string;
  label: string;
  icon?: string;
  tooltip?: string;
  labelSelector?: ConfigLabelSelector;
}

@Component({
  selector: 'app-active-filter-chips',
  templateUrl: './active-filter-chips.component.html',
  styleUrls: ['./active-filter-chips.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [LabelDisplayComponent, MatChipsModule, MatIcon, MatTooltipModule],
  standalone: true,
})
export class ActiveFilterChipsComponent implements OnChanges {
  @Input({required: true}) query: ConfigQuery;
  @Input() options: ConfigOptions | null = null;
  @Input() entity: ConfigObject | null = null;
  @Input() disabled = false;
  @Output() readonly removeFilter = new EventEmitter<ActiveConfigFilterChip>();

  chips: ActiveConfigFilterChip[] = [];

  ngOnChanges(): void {
    const chips: ActiveConfigFilterChip[] = [];
    if (this.query && this.query.kind !== Kind.ROLEMAPPING && this.query.kind !== Kind.UNDEFINED) {
      const labelSelector = parseConfigSearchTerm(this.query.term ?? '').label;
      if (labelSelector) {
        chips.push({
          key: 'labelSelector',
          value: labelSelector.selector,
          label: labelSelector.selector,
          icon: isEmojiLabel(labelSelector.key, labelSelector.value) ? undefined : 'label',
          labelSelector,
        });
      }
    }
    for (const id of this.query?.crawlJobIdList ?? []) {
      const name = this.options?.crawlJobs?.find(config => config.id === id)?.meta?.name || id;
      chips.push({
        key: 'crawlJobIdList',
        value: id,
        label: name,
        icon: configKindIcon(Kind.CRAWLJOB),
      });
    }
    if (this.query?.entityId) {
      const entityName = this.entity?.meta?.name;
      chips.push({
        key: 'entityId',
        value: this.query.entityId,
        label: entityName || $localize`:@@activeConfigEntityFilterFallbackChip:Entity: ${this.query.entityId}:ENTITY_ID:`,
        icon: configKindIcon(Kind.CRAWLENTITY),
        tooltip: $localize`:@@activeConfigEntityFilterTooltip:Entity ID: ${this.query.entityId}:ENTITY_ID:`,
      });
    }
    for (const id of this.query?.scriptIdList ?? []) {
      const name = this.options?.browserScripts?.find(config => config.id === id)?.meta?.name || id;
      chips.push({
        key: 'scriptIdList',
        value: id,
        label: name,
        icon: configKindIcon(Kind.BROWSERSCRIPT),
      });
    }
    this.chips = chips;
  }

  removeFilterLabel(chip: ActiveConfigFilterChip): string {
    if (chip.key === 'labelSelector') {
      return $localize`:@@configQueryRemoveLabelFilterLabel:Remove ${chip.value} label filter`;
    }
    if (chip.key === 'crawlJobIdList') {
      return $localize`:@@configQueryRemoveCrawlJobFilterLabel:Remove ${chip.label} crawl job filter`;
    }
    if (chip.key === 'scriptIdList') {
      return $localize`:@@activeConfigRemoveBrowserScriptFilterLabel:Remove BrowserScript ${chip.label} filter`;
    }
    if (chip.key === 'entityId') {
      return $localize`:@@activeConfigRemoveEntityFilterLabel:Remove entity ${chip.value}:ENTITY_ID: filter`;
    }
    return $localize`:@@activeConfigRemoveFilterLabel:Remove ${chip.label} filter`;
  }

  onRemoveFilter(chip: ActiveConfigFilterChip): void {
    if (!this.disabled) {
      this.removeFilter.emit(chip);
    }
  }
}
