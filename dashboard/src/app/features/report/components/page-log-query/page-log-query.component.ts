import {Component} from '@angular/core';
import {ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';
import {MatCheckboxModule} from '@angular/material/checkbox';
import {MatFormFieldModule} from '@angular/material/form-field';
import {QueryComponent} from '../../../../shared/components';
import {PageLogQuery} from '../../services';
import {LayoutDirective} from '@ngbracket/ngx-layout';
import {MatInputModule} from '@angular/material/input';

@Component({
  selector: 'app-page-log-query',
  templateUrl: './page-log-query.component.html',
  styleUrls: ['./page-log-query.component.css'],
  standalone: true,
  imports: [
    LayoutDirective,
    MatFormFieldModule,
    MatInputModule,
    MatCheckboxModule,
    ReactiveFormsModule,
  ]
})
export class PageLogQueryComponent extends QueryComponent<PageLogQuery> {

  constructor(protected override fb: UntypedFormBuilder) {
    super(fb);
  }

  protected override createForm(): void {
    this.form = this.fb.group({
      jobExecutionId: '',
      executionId: '',
      uri: '',
      watch: null,
    });
  }
}
