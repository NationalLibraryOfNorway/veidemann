import {ChangeDetectionStrategy, Component} from '@angular/core';
import {ConfigObject, Role} from '../../../../../shared/models';
import {CONFIG_LIST_IMPORTS, ConfigListBaseComponent} from '../../config-list/config-list-base';


@Component({
  selector: 'app-rolemapping-list',
  templateUrl: './rolemapping-list.component.html',
  styleUrls: ['../../config-list/config-list-base.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [...CONFIG_LIST_IMPORTS],
  standalone: true
})

export class RoleMappingListComponent extends ConfigListBaseComponent<ConfigObject> {
  override displayedColumns = ['select', 'email', 'group', 'role'];

  getRoles(roles: Role[]): string[] {
    return roles.map(role => Role[role]);
  }
}
