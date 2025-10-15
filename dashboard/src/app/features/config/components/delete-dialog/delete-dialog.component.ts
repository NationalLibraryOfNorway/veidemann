import {Component, EventEmitter, Inject, Output} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../shared/models/config';
import {MatButtonModule} from '@angular/material/button';
import {FlexDirective} from '@ngbracket/ngx-layout';

@Component({
  selector: 'app-delete-dialog',
  templateUrl: 'delete-dialog.component.html',
  imports: [
    FlexDirective,
    MatButtonModule,
    MatDialogModule
  ],
  standalone: true
})
export class DeleteDialogComponent {
  readonly Kind = Kind;
  @Output()
  delete = new EventEmitter();

  configObject: ConfigObject;


  constructor(@Inject(MAT_DIALOG_DATA) public data: any) {
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
