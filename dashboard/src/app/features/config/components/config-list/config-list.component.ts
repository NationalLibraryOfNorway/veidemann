import {ChangeDetectionStrategy, Component} from '@angular/core';
import {ConfigObject} from '../../../../shared/models/config';
import {animate, state, style, transition, trigger} from '@angular/animations';
import {BASE_LIST_IMPORTS, BaseListComponent} from '../../../../shared/components';

@Component({
    selector: 'app-config-list',
    templateUrl: '../../../../shared/components/base-list/base-list.html',
    styleUrls: [
        '../../../../shared/components/base-list/base-list.scss',
        '../../../../shared/components/base-list/base-list-odd-preview.scss',
    ],
    changeDetection: ChangeDetectionStrategy.OnPush,
    animations: [
        trigger('detailExpand', [
            state('collapsed', style({ height: '0px', minHeight: '0', opacity: 0 })),
            state('expanded', style({ height: '*', opacity: 1 })),
            transition('expanded <=> collapsed', animate('225ms cubic-bezier(0.4, 0.0, 0.2, 1)')),
        ]),
    ],
    imports: [...BASE_LIST_IMPORTS],
    standalone: true,
})

export class ConfigListComponent extends BaseListComponent<ConfigObject> {
  override isDisabled(config: ConfigObject): boolean {
    return config?.crawlJob?.disabled || config?.seed?.disabled;
  }
}
