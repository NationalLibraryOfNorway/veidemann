import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import {MatButtonModule} from '@angular/material/button';

@Component({
  selector: 'app-error-dialog',
  template: `<h1 mat-dialog-title>{{ title }}</h1>
  <div mat-dialog-content>{{ content }}</div>
  <div mat-dialog-actions>
    <button mat-raised-button color="warn" matDialogClose>Ok</button>
  </div>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatButtonModule,
    MatDialogModule
  ],
  standalone: true
})
export class ErrorDialogComponent {
  dialogRef = inject<MatDialogRef<ErrorDialogComponent>>(MatDialogRef);
  data = inject(MAT_DIALOG_DATA);

  title: string;
  content: string;

  constructor() {
    const data = this.data;

    this.title = data.error.name || 'Error';
    this.content = data.error.message || data.error.code || data.error.toString();
  }
}
