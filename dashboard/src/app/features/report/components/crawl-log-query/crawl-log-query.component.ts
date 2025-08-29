import {Component} from '@angular/core';
import {ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';
import {CrawlLogQuery} from '../../services';
import {QueryComponent} from '../../../../shared/components';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatCheckboxModule} from '@angular/material/checkbox';
import {MatInputModule} from '@angular/material/input';
import {FlexLayoutModule} from '@angular/flex-layout';

@Component({
  selector: 'app-crawl-log-query',
  templateUrl: './crawl-log-query.component.html',
  styleUrls: ['./crawl-log-query.component.css'],
  imports: [
    FlexLayoutModule,
    MatCheckboxModule,
    MatFormFieldModule,
    MatInputModule,
    ReactiveFormsModule
  ],
  standalone: true
})
export class CrawlLogQueryComponent extends QueryComponent<CrawlLogQuery> {

  constructor(protected override fb: UntypedFormBuilder) {
    super(fb);
  }

  protected override createForm(): void {
    this.form = this.fb.group({
      jobExecutionId: '',
      executionId: '',
      watch: null,
    });
  }
}
