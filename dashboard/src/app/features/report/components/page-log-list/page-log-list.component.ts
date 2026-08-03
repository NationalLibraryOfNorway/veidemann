import {CommonModule} from '@angular/common';
import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {MatMenuModule} from '@angular/material/menu';
import {MatPaginatorModule} from '@angular/material/paginator';
import {BASE_LIST_IMPORTS, BaseListComponent} from '../../../../shared/components';
import {PageLog} from '../../../../shared/models';
import {MatIconModule} from '@angular/material/icon';
import {UrlFormatPipe} from '../../../../shared/pipes/url-format.pipe';
import {MatTableModule} from '@angular/material/table';
import {MatSortModule} from '@angular/material/sort';
import {MatButtonModule} from '@angular/material/button';
import {FlexDirective, LayoutDirective} from '@ngbracket/ngx-layout';

@Component({
  selector: 'app-pagelog-list',
  templateUrl: './page-log-list.component.html',
  styleUrls: [
    '../../../../shared/components/base-list/base-list.scss',
    '../../../../shared/components/base-list/base-list-odd.scss',
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    CommonModule,
    FlexDirective,
    LayoutDirective,
    MatButtonModule,
    MatMenuModule,
    MatIconModule,
    MatPaginatorModule,
    MatSortModule,
    MatTableModule,
    UrlFormatPipe,
    ...BASE_LIST_IMPORTS,
  ]
})
export class PageLogListComponent extends BaseListComponent<PageLog> {

  @Input()
  override multiSelect = false;

  override displayedColumns: string[] = ['uri', 'nrOfResources', 'nrOfOutlinks', 'extra', 'action'];

}
