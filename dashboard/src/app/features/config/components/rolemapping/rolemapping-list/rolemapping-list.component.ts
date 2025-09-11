import {ChangeDetectionStrategy, Component, forwardRef} from '@angular/core';
import {ConfigObject, Role} from '../../../../../shared/models';
import {BASE_LIST_IMPORTS, BaseListComponent} from '../../../../../shared/components';
import {BASE_LIST} from '../../../../../shared/directives';
import {MatTableModule} from '@angular/material/table';
import {MatCheckbox} from '@angular/material/checkbox';
import {AsyncPipe, NgClass} from '@angular/common';
import {MatPaginator} from '@angular/material/paginator';
import {MatButtonModule} from '@angular/material/button';
import {FlexLayoutModule} from '@ngbracket/ngx-layout';


@Component({
  selector: 'app-rolemapping-list',
  templateUrl: './rolemapping-list.component.html',
  styleUrls: ['../../../../../shared/components/base-list/base-list.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    {
      provide: BASE_LIST,
      useExisting: forwardRef(() => RoleMappingListComponent)
    }
  ],
  imports: [
    AsyncPipe,
    FlexLayoutModule,
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

  constructor() {
    super();
  }

  getRoles(roles: Role[]): string[] {
    return roles.map(role => Role[role]);
  }
}
