import {ChangeDetectionStrategy, ChangeDetectorRef, Component, forwardRef, Input} from '@angular/core';

import {CrawlLog, ListDataSource} from '../../../../shared/models';
import {animate, state, style, transition, trigger} from '@angular/animations';
import {KeyboardShortcutsModule} from 'ng-keyboard-shortcuts';
import {BASE_LIST_IMPORTS, BaseListComponent} from '../../../../shared/components';
import {BASE_LIST} from '../../../../shared/directives';
import {FlexDirective, LayoutDirective} from '@ngbracket/ngx-layout';
import {MatTableModule} from '@angular/material/table';
import {MatSortModule} from '@angular/material/sort';
import {UrlFormatPipe} from '../../../../shared/pipes/url-format.pipe';
import {AsyncPipe, DatePipe, NgTemplateOutlet} from '@angular/common';
import {MatTooltip} from '@angular/material/tooltip';
import {MatIcon} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {MatMenuModule} from '@angular/material/menu';
import {CrawlLogPreviewComponent} from '../crawl-log-preview/crawl-log-preview.component';
import {MatPaginatorModule} from '@angular/material/paginator';

@Component({
  selector: 'app-crawl-log-list',
  templateUrl: './crawl-log-list.component.html',
  styleUrls: ['./crawl-log-list.component.scss',
    '../../../../shared/components/base-list/base-list.scss',
    '../../../../shared/components/base-list/base-list-odd-preview.scss',
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    ListDataSource,
    {
      provide: BASE_LIST,
      useExisting: forwardRef(() => CrawlLogListComponent)
    }
  ],
  animations: [
    trigger('detailExpand', [
      state('collapsed', style({height: '0px', minHeight: '0', opacity: 0})),
      state('expanded', style({height: '*', opacity: 1})),
      transition('expanded <=> collapsed', animate('225ms cubic-bezier(0.4, 0.0, 0.2, 1)')),
    ]),
  ],
  imports: [
    AsyncPipe,
    CrawlLogPreviewComponent,
    DatePipe,
    FlexDirective,
    KeyboardShortcutsModule,
    LayoutDirective,
    MatButtonModule,
    MatIcon,
    MatMenuModule,
    MatPaginatorModule,
    MatSortModule,
    MatTableModule,
    MatTooltip,
    NgTemplateOutlet,
    UrlFormatPipe,
    ...BASE_LIST_IMPORTS,
  ],
  standalone: true
})
export class CrawlLogListComponent extends BaseListComponent<CrawlLog> {

  @Input()
  override multiSelect = false;

  @Input()
  override sortActive = 'timestamp';

  override displayedColumns: string[] =
    ['requestedUri', 'timestamp', 'statusCode', 'discoveryPath', 'contentType', 'extra', 'action'];

  constructor(protected override cdr: ChangeDetectorRef) {
    super(cdr);
  }
}
