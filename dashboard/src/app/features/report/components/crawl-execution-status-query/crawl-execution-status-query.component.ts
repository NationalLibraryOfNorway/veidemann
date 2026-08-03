import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy,Component,Input } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatTimepickerModule } from '@angular/material/timepicker';
import { FlexLayoutModule } from '@ngbracket/ngx-layout';
import { QueryComponent } from '../../../../shared/components';
import { ConfigObject,CrawlExecutionState,crawlExecutionStates } from '../../../../shared/models';
import { CrawlExecutionStatusQuery } from '../../services';

@Component({
    selector: 'app-crawl-execution-status-query',
    templateUrl: './crawl-execution-status-query.component.html',
    styleUrls: ['./crawl-execution-status-query.component.css'],
    changeDetection: ChangeDetectionStrategy.OnPush,
    standalone: true,
  imports: [
    CommonModule,
    FlexLayoutModule,
    MatCheckboxModule,
    MatDatepickerModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatTimepickerModule,
    ReactiveFormsModule,

  ]
})
export class CrawlExecutionStatusQueryComponent extends QueryComponent<CrawlExecutionStatusQuery> {

  readonly crawlExecutionStates = crawlExecutionStates;
  readonly CrawlExecutionState = CrawlExecutionState;

  @Input()
  crawlJobOptions: ConfigObject[];

  protected override createForm(): void {
    this.form = this.fb.group({
      stateList: null,
      seedId: '',
      jobId: '',
      jobExecutionId: '',
      startTimeFrom: '',
      startTimeTo: '',
      hasError: null,
      watch: null,
    });
  }
}
