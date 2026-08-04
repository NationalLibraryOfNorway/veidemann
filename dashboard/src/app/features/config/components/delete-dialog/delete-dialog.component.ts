import { Component, EventEmitter, Output, ChangeDetectionStrategy, inject } from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../shared/models/config';
import {MatButtonModule} from '@angular/material/button';

@Component({
  selector: 'app-delete-dialog',
  templateUrl: 'delete-dialog.component.html',
  imports: [
    MatButtonModule,
    MatDialogModule
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class DeleteDialogComponent {
  data = inject(MAT_DIALOG_DATA);

  readonly Kind = Kind;
  @Output()
  delete = new EventEmitter();

  configObject: ConfigObject;


  constructor() {
    const data = this.data;

    this.configObject = data.configObject;
  }

  get roleMapping(): string {
    if (this.configObject.roleMapping.group) {
      return this.configObject.roleMapping.group;
    } else {
      return this.configObject.roleMapping.email;
    }
  }
}
