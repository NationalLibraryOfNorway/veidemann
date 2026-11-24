import { ChangeDetectionStrategy,Component } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { QueryComponent } from '../../../../shared/components';
import { CrawlLogQuery } from '../../services';

@Component({
  selector: 'app-crawl-log-query',
  templateUrl: './crawl-log-query.component.html',
  styleUrls: ['./crawl-log-query.component.css'],
  imports: [
    ReactiveFormsModule
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class CrawlLogQueryComponent extends QueryComponent<CrawlLogQuery> {

  protected override createForm(): void {
    this.form = this.fb.group({
      executionId: '',
    });
  }
}
