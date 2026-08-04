import { ChangeDetectionStrategy,Component } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { QueryComponent } from '../../../../shared/components';
import { PageLogQuery } from '../../services';

@Component({
  selector: 'app-page-log-query',
  templateUrl: './page-log-query.component.html',
  styleUrls: ['./page-log-query.component.css'],
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatFormFieldModule,
    MatInputModule,
    MatCheckboxModule,
    ReactiveFormsModule,
  ]
})
export class PageLogQueryComponent extends QueryComponent<PageLogQuery> {

  protected override createForm(): void {
    this.form = this.fb.group({
      jobExecutionId: '',
      executionId: '',
      uri: '',
      watch: null,
    });
  }
}
