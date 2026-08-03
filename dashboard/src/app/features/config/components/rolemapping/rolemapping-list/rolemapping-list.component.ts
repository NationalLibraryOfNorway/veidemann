import {ChangeDetectionStrategy, Component} from '@angular/core';
import {ConfigObject, Role} from '../../../../../shared/models';
import {BASE_LIST_IMPORTS, BaseListComponent} from '../../../../../shared/components';
import {MatTableModule} from '@angular/material/table';
import {MatCheckbox} from '@angular/material/checkbox';
import {NgClass} from '@angular/common';
import {MatPaginator} from '@angular/material/paginator';
import {MatButtonModule} from '@angular/material/button';


@Component({
  selector: 'app-rolemapping-list',
  templateUrl: './rolemapping-list.component.html',
  styleUrls: ['../../../../../shared/components/base-list/base-list.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatButtonModule,
    MatCheckbox,
    MatPaginator,
    MatTableModule,
    NgClass,
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
