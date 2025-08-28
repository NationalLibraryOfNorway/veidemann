import {Component} from '@angular/core';
import {UntypedFormBuilder} from '@angular/forms';
import {CrawlLogQuery} from '../../services';
import {QueryComponent} from '../../../../shared/components';

@Component({
    selector: 'app-crawl-log-query',
    templateUrl: './crawl-log-query.component.html',
    styleUrls: ['./crawl-log-query.component.css'],
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
