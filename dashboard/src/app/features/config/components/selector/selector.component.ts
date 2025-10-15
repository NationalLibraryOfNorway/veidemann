import {ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit} from '@angular/core';
import {NG_VALUE_ACCESSOR, ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';
import {Label} from '../../../../shared/models';
import {LabelService} from '../../services/label.service';
import {LabelComponent} from '../label/label.component';
import {AsyncPipe} from '@angular/common';
import {CdkDrag, CdkDropList} from '@angular/cdk/drag-drop';
import {FlexLayoutModule} from '@ngbracket/ngx-layout';
import {MatAutocompleteModule} from '@angular/material/autocomplete';
import {MatButtonModule} from '@angular/material/button';
import {MatCardModule} from '@angular/material/card';
import {MatChipsModule} from '@angular/material/chips';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatIcon} from '@angular/material/icon';
import {MatInputModule} from '@angular/material/input';


@Component({
  selector: 'app-selector',
  templateUrl: '../label/label.component.html',
  styleUrls: ['../label/label.component.scss'],
  providers: [{provide: NG_VALUE_ACCESSOR, useExisting: SelectorComponent, multi: true}],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    AsyncPipe,
    CdkDrag,
    CdkDropList,
    FlexLayoutModule,
    MatAutocompleteModule,
    MatButtonModule,
    MatCardModule,
    MatChipsModule,
    MatFormFieldModule,
    MatIcon,
    MatInputModule,
    ReactiveFormsModule
  ],
  standalone: true
})
export class SelectorComponent extends LabelComponent implements OnInit {

  @Input()
  override placeholderText = 'New selector...';
  override labelText = 'Script selector';

  constructor(protected override fb: UntypedFormBuilder,
              protected override cdr: ChangeDetectorRef,
              protected override labelService: LabelService) {
    super(fb, cdr, labelService);
  }

  // eslint-disable-next-line @angular-eslint/no-empty-lifecycle-method
  override ngOnInit(): void {
    // prevent fetching label by not calling super();
  }

  protected override save(value: string): void {
    let key = '';
    value = value.trim();

    if (value === '') {
      return;
    }

    const parts = value.split(':');
    if (parts.length > 1) {
      key = parts.shift();
      value = parts.join(':');
    } else {
      key = parts[0].trim();
      value = '';
    }

    if (this.findLabelIndex(key, value) > -1) {
      return;
    }

    this.labels.push(new Label({key, value}));
  }

  protected override createForm(): void {
    this.labelForm = this.fb.group({
      key: '',
      value: ''
    });
    this.labelForm.disable();
  }
}
