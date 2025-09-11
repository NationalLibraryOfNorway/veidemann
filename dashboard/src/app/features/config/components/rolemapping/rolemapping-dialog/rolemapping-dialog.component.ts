import {Component, Inject, OnInit} from '@angular/core';
import {RoleMappingDetailsComponent} from '..';
import {ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';
import {AuthService} from '../../../../../core/auth';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject} from '../../../../../shared/models/config';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatSelectModule} from '@angular/material/select';
import {FlexLayoutModule} from '@ngbracket/ngx-layout';
import {MatInput} from '@angular/material/input';
import {MatButtonModule} from '@angular/material/button';

@Component({
  selector: 'app-rolemapping-dialog',
  templateUrl: './rolemapping-dialog.component.html',
  styleUrls: ['./rolemapping-dialog.component.css'],
  imports: [
    FlexLayoutModule,
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInput,
    MatSelectModule,
    ReactiveFormsModule
  ],
  standalone: true
})
export class RoleMappingDialogComponent extends RoleMappingDetailsComponent implements OnInit {

  constructor(protected override fb: UntypedFormBuilder,
              protected authService: AuthService,
              @Inject(MAT_DIALOG_DATA) public data: ConfigDialogData,
              public dialogRef: MatDialogRef<RoleMappingDialogComponent>) {
    super(fb);
    this.createForm();
    this.configObject = this.data.configObject;
    this.roles = this.data.options.roles;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onDialogClose(): ConfigObject {
    return this.prepareSave();
  }
}
