import {ChangeDetectionStrategy, Component, Input} from '@angular/core';

import {CrawlLog} from '../../../../shared/models';
import {BASE_LIST_IMPORTS, BaseListComponent} from '../../../../shared/components';
import {FlexDirective, LayoutDirective} from '@ngbracket/ngx-layout';
import {MatTableModule} from '@angular/material/table';
import {MatSortModule} from '@angular/material/sort';
import {DatePipe, NgTemplateOutlet} from '@angular/common';
import {MatTooltip} from '@angular/material/tooltip';
import {MatIcon} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {MatMenuModule} from '@angular/material/menu';

@Component({
  selector: 'app-crawl-log-list',
  templateUrl: './crawl-log-list.component.html',
  styleUrls: ['./crawl-log-list.component.scss',
    '../../../../shared/components/base-list/base-list.scss',
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    DatePipe,
    FlexDirective,
    LayoutDirective,
    MatButtonModule,
    MatIcon,
    MatMenuModule,
    MatSortModule,
    MatTableModule,
    MatTooltip,
    NgTemplateOutlet,
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

}
