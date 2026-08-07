import { CdkDrag,CdkDropList } from '@angular/cdk/drag-drop';
import { AsyncPipe } from '@angular/common';
import { ChangeDetectionStrategy,Component,Input,OnInit } from '@angular/core';
import { NG_VALUE_ACCESSOR,ReactiveFormsModule } from '@angular/forms';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIcon } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatTooltipModule } from '@angular/material/tooltip';
import { Label } from '../../../../shared/models';
import { LabelDisplayComponent } from '../../../../shared/components';
import { LabelComponent } from '../label/label.component';


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
    MatAutocompleteModule,
    MatButtonModule,
    MatCardModule,
    MatChipsModule,
    MatFormFieldModule,
    MatIcon,
    MatInputModule,
    MatTooltipModule,
    LabelDisplayComponent,
    ReactiveFormsModule
  ],
  standalone: true
})
export class SelectorComponent extends LabelComponent implements OnInit {


  @Input()
  override placeholderText = 'New selector...';
  override labelText = 'Script selector';
  protected override emojiPickerEnabled = false;

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
