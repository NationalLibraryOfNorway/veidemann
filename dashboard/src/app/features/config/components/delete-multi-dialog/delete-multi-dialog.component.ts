import {Component, Inject} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule} from '@angular/material/dialog';
import {MatFormFieldModule} from '@angular/material/form-field';
import {FormsModule} from '@angular/forms';
import {MatInputModule} from '@angular/material/input';
import {MatButtonModule} from '@angular/material/button';
import {FlexDirective} from '@ngbracket/ngx-layout';

export interface DeleteDialogData {
  numberOfConfigs: number;
}

@Component({
  selector: 'app-delete-multi-dialog',
  styleUrls: ['delete-multi-dialog.component.scss'],
  templateUrl: 'delete-multi-dialog.component.html',
  imports: [
    FlexDirective,
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    FormsModule
  ],
  standalone: true
})
export class DeleteMultiDialogComponent {

  numberOfConfigs: number;

  constructor(@Inject(MAT_DIALOG_DATA) public data: DeleteDialogData) {
  }
}
