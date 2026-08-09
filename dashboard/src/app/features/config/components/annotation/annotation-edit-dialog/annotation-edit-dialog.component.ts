import {AfterViewInit, ChangeDetectionStrategy, Component, ElementRef, ViewChild, inject} from '@angular/core';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {MatButtonModule} from '@angular/material/button';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatInputModule} from '@angular/material/input';

import {NO_COLON} from '../../../../../shared/validation/patterns';

export interface AnnotationEditDialogData {
  key: string;
  value: string;
  type?: 'annotation' | 'label';
}

export type AnnotationEditDialogResult = AnnotationEditDialogData;

@Component({
  selector: 'app-annotation-edit-dialog',
  templateUrl: './annotation-edit-dialog.component.html',
  styleUrls: ['./annotation-edit-dialog.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    ReactiveFormsModule,
  ],
  standalone: true,
})
export class AnnotationEditDialogComponent implements AfterViewInit {
  private readonly fb = inject(FormBuilder);
  private readonly dialogRef = inject<MatDialogRef<AnnotationEditDialogComponent, AnnotationEditDialogResult>>(
    MatDialogRef
  );
  readonly data = inject<AnnotationEditDialogData>(MAT_DIALOG_DATA);

  readonly form = this.fb.nonNullable.group({
    key: [this.data.key, [Validators.required, Validators.pattern(NO_COLON), Validators.pattern(/\S/)]],
    value: [this.data.value, [Validators.required, Validators.pattern(/\S/)]],
  });

  @ViewChild('valueInput') private valueInput: ElementRef<HTMLInputElement>;

  get key() {
    return this.form.controls.key;
  }

  get value() {
    return this.form.controls.value;
  }

  get canApply(): boolean {
    const {key, value} = this.form.getRawValue();
    return this.form.valid
      && this.form.dirty
      && (key.trim() !== this.data.key.trim() || value.trim() !== this.data.value.trim());
  }

  ngAfterViewInit(): void {
    this.valueInput.nativeElement.focus();
    this.valueInput.nativeElement.select();
  }

  onApply(): void {
    if (!this.canApply) {
      return;
    }

    this.dialogRef.close({
      key: this.key.value.trim(),
      value: this.value.value.trim(),
    });
  }
}
