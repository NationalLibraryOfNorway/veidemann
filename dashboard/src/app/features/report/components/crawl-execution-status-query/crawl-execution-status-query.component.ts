import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';
import {QueryComponent} from '../../../../shared/components';
import {ConfigObject, CrawlExecutionState, crawlExecutionStates} from '../../../../shared/models';
import {CrawlExecutionStatusQuery} from '../../services';
import {MatFormFieldModule} from '@angular/material/form-field';
import {CommonModule} from '@angular/common';
import {MatSelectModule} from '@angular/material/select';
import {MatDatepickerModule} from '@angular/material/datepicker';
import {MatTimepickerModule} from '@angular/material/timepicker';
import {MatInputModule} from '@angular/material/input';
import {MatCheckboxModule} from '@angular/material/checkbox';
import {FlexLayoutModule} from '@ngbracket/ngx-layout';

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

  constructor(protected override fb: UntypedFormBuilder) {
    super(fb);
  }

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
