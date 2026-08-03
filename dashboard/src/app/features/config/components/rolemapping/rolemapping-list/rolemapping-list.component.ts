import {ChangeDetectionStrategy, Component} from '@angular/core';
import {ConfigObject, Role} from '../../../../../shared/models';
import {BASE_LIST_IMPORTS, BaseListComponent} from '../../../../../shared/components';
import {AsyncPipe} from '@angular/common';
import {MatButtonModule} from '@angular/material/button';


@Component({
  selector: 'app-rolemapping-list',
  templateUrl: './rolemapping-list.component.html',
  styleUrls: ['../../../../../shared/components/base-list/base-list.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatButtonModule,
    AsyncPipe,
    ...BASE_LIST_IMPORTS
  ],
  standalone: true
})

export class RoleMappingListComponent extends BaseListComponent<ConfigObject> {
  override displayedColumns = ['select', 'email', 'group', 'role'];

  getRoles(roles: Role[]): string[] {
    return roles.map(role => Role[role]);
  }
}
