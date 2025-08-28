import {ChangeDetectionStrategy, ChangeDetectorRef, Component, forwardRef, Input} from '@angular/core';
import {BaseListComponent} from '../../../../shared/components';
import {CrawlExecutionState, crawlExecutionStates, CrawlExecutionStatus, ListDataSource} from '../../../../shared/models';
import {BASE_LIST} from '../../../../shared/directives';
import {animate, state, style, transition, trigger} from '@angular/animations';
import { MatPaginatorModule } from '@angular/material/paginator';
import { MatTableDataSource, MatTableModule } from '@angular/material/table';

@Component({
    selector: 'app-crawl-execution-status-list',
    templateUrl: './crawl-execution-status-list.component.html',
    styleUrls: [
        '../../../commons/components/base-list/base-list.scss',
        '../../../commons/components/base-list/base-list-odd-preview.scss',
    ],
    changeDetection: ChangeDetectionStrategy.OnPush,
    providers: [
        ListDataSource,
        {
            provide: BASE_LIST,
            useExisting: forwardRef(() => CrawlExecutionStatusListComponent)
        }
    ],
    animations: [
        trigger('detailExpand', [
            state('collapsed', style({ height: '0px', minHeight: '0' })),
            state('expanded', style({ height: '*' })),
            transition('expanded <=> collapsed', animate('225ms cubic-bezier(0.4, 0.0, 0.2, 1)')),
        ]),
    ],
    standalone: true,
    imports: [
        MatPaginatorModule,
        MatTableModule,
        MatTableDataSource,
    ]
})
export class CrawlExecutionStatusListComponent extends BaseListComponent<CrawlExecutionStatus> {
  readonly CrawlExecutionState = CrawlExecutionState;
  readonly crawlExecutionStates = crawlExecutionStates;

  override multiSelect = false;

  @Input()
  override sortActive = 'startTime';

  override displayedColumns: string[] = ['seedId', 'jobId', 'state', 'desiredState', 'errorCode', 'documentsCrawled', 'queueCount', 'startTime', 'endTime', 'extra', 'action'];

  constructor(protected override cdr: ChangeDetectorRef) {
    super(cdr);
  }
}
