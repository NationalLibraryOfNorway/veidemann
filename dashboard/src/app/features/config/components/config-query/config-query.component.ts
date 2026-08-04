import { AfterViewInit,ChangeDetectionStrategy,Component,ElementRef,Input,OnChanges,ViewChild } from '@angular/core';
import { FormsModule,ReactiveFormsModule } from '@angular/forms';

import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIcon } from '@angular/material/icon';
import { MatInput } from '@angular/material/input';
import { MatRadioModule } from '@angular/material/radio';
import { MatSelectModule } from '@angular/material/select';
import { QueryComponent } from '../../../../shared/components';
import { ConfigQuery } from '../../../../shared/func';
import { Kind } from '../../../../shared/models';
import { ConfigOptions } from '../../func';


@Component({
  selector: 'app-config-query',
  styleUrls: ['config-query.component.scss'],
  templateUrl: './config-query.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    FormsModule,
    MatButtonModule,
    MatFormFieldModule,
    MatIcon,
    MatInput,
    MatRadioModule,
    MatSelectModule,
    ReactiveFormsModule
  ],
  standalone: true
})
export class ConfigQueryComponent extends QueryComponent<ConfigQuery> implements OnChanges, AfterViewInit {

  readonly Kind = Kind;

  term: string;

  @Input()
  options: ConfigOptions;

  @ViewChild('search') searchElement: ElementRef;

  override onQuery(query: ConfigQuery) {
    super.onQuery({term: this.term, ...query});
  }

  onSearch(term: string) {
    this.onQuery({...this.form.value, term});
  }

  protected override createForm(): void {
    this.form = this.fb.group({
      entityId: '',
      scheduleId: '',
      crawlConfigId: '',
      collectionId: '',
      browserConfigId: '',
      politenessId: '',
      crawlJobIdList: {value: [], disabled: false},
      scriptIdList: {value: [], disabled: false},
      disabled: {value: null, disabled: false},
    });
  }

  protected override updateForm(): void {
    this.term = this.query.term;
    super.updateForm();
  }
}
