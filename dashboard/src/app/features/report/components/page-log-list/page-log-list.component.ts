import {CommonModule} from '@angular/common';
import {ChangeDetectionStrategy, ChangeDetectorRef, Component, forwardRef, Input} from '@angular/core';
import {MatMenuModule} from '@angular/material/menu';
import {MatPaginatorModule} from '@angular/material/paginator';
import {BaseListComponent} from '../../../../shared/components';
import {BASE_LIST} from '../../../../shared/directives';
import {ListDataSource, PageLog} from '../../../../shared/models';
import {MatIconModule} from '@angular/material/icon';
import {UrlFormatPipe} from '../../../../shared/pipes/url-format.pipe';
import {MatTableModule} from '@angular/material/table';
import {MatSortModule} from '@angular/material/sort';
import {KeyboardShortcutsModule} from 'ng-keyboard-shortcuts';
import {FlexLayoutModule} from '@ngbracket/ngx-layout';
import {MatButtonModule} from '@angular/material/button';

@Component({
  selector: 'app-pagelog-list',
  templateUrl: './page-log-list.component.html',
  styleUrls: [
    '../../../../shared/components/base-list/base-list.scss',
    '../../../../shared/components/base-list/base-list-odd.scss',
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    ListDataSource,
    {
      provide: BASE_LIST,
      useExisting: forwardRef(() => PageLogListComponent)
    }
  ],
  standalone: true,
  imports: [
    CommonModule,
    FlexLayoutModule,
    KeyboardShortcutsModule,
    MatButtonModule,
    MatMenuModule,
    MatIconModule,
    MatPaginatorModule,
    MatSortModule,
    MatTableModule,
    UrlFormatPipe,
  ]
})
export class PageLogListComponent extends BaseListComponent<PageLog> {

  @Input()
  override multiSelect = false;

  override displayedColumns: string[] = ['uri', 'nrOfResources', 'nrOfOutlinks', 'extra', 'action'];

  constructor(protected override cdr: ChangeDetectorRef) {
    super(cdr);
  }
}
