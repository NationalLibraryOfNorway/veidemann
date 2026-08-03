import {ChangeDetectionStrategy, Component} from '@angular/core';
import {ConfigObject} from '../../../../shared/models/config';
import {BASE_LIST_IMPORTS, BaseListComponent} from '../../../../shared/components';

@Component({
    selector: 'app-config-list',
    templateUrl: '../../../../shared/components/base-list/base-list.html',
    styleUrls: [
        '../../../../shared/components/base-list/base-list.scss',
    ],
    changeDetection: ChangeDetectionStrategy.OnPush,
    imports: [...BASE_LIST_IMPORTS],
    standalone: true,
})

export class ConfigListComponent extends BaseListComponent<ConfigObject> {
  override isDisabled(config: ConfigObject): boolean {
    return config?.crawlJob?.disabled || config?.seed?.disabled;
  }
}
