import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ChangeDetectionStrategy, Component, Inject} from '@angular/core';
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
  title: string;
  content: string;

  constructor(public dialogRef: MatDialogRef<ErrorDialogComponent>,
              @Inject(MAT_DIALOG_DATA) public data: any) {
    this.title = data.error.name || 'Error';
    this.content = data.error.message || data.error.code || data.error.toString();
  }
}
