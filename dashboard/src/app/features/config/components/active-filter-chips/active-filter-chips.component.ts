import {ChangeDetectionStrategy, Component, EventEmitter, Input, OnChanges, Output} from '@angular/core';
import {MatChipsModule} from '@angular/material/chips';
import {MatIcon} from '@angular/material/icon';

import {LabelDisplayComponent} from '../../../../shared/components/label-display/label-display.component';
import {ConfigQuery} from '../../../../shared/func';
import {ConfigObject, Kind} from '../../../../shared/models';
import {parseConfigSearchTerm} from '../../func/query';
import type {ConfigLabelSelector} from '../../func/query';
import type {ConfigOptions} from '../../func/options';

export interface ActiveConfigFilterChip {
  key: 'entityId' | 'scriptIdList' | 'labelSelector';
  value: string;
  label: string;
  labelSelector?: ConfigLabelSelector;
}

@Component({
  selector: 'app-active-filter-chips',
  templateUrl: './active-filter-chips.component.html',
  styleUrls: ['./active-filter-chips.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [LabelDisplayComponent, MatChipsModule, MatIcon],
  standalone: true,
})
export class ActiveFilterChipsComponent implements OnChanges {
  @Input({required: true}) query: ConfigQuery;
  @Input() options: ConfigOptions | null = null;
  @Input() entity: ConfigObject | null = null;
  @Output() readonly removeFilter = new EventEmitter<ActiveConfigFilterChip>();

  chips: ActiveConfigFilterChip[] = [];

  ngOnChanges(): void {
    const chips: ActiveConfigFilterChip[] = [];
    if (this.query?.entityId) {
      chips.push({
        key: 'entityId',
        value: this.query.entityId,
        label: $localize`Entity: ${this.entity?.meta?.name || this.query.entityId}`,
      });
    }
    for (const id of this.query?.scriptIdList ?? []) {
      const name = this.options?.browserScripts?.find(config => config.id === id)?.meta?.name || id;
      chips.push({key: 'scriptIdList', value: id, label: $localize`BrowserScript: ${name}`});
    }
    if (this.query && this.query.kind !== Kind.ROLEMAPPING && this.query.kind !== Kind.UNDEFINED) {
      const labelSelector = parseConfigSearchTerm(this.query.term ?? '').label;
      if (labelSelector) {
        chips.push({
          key: 'labelSelector',
          value: labelSelector.selector,
          label: labelSelector.selector,
          labelSelector,
        });
      }
    }
    this.chips = chips;
  }

  removeFilterLabel(chip: ActiveConfigFilterChip): string {
    if (chip.key === 'labelSelector') {
      return $localize`:@@configQueryRemoveLabelFilterLabel:Remove ${chip.value} label filter`;
    }
    return $localize`:@@activeConfigRemoveFilterLabel:Remove ${chip.label} filter`;
  }
}
