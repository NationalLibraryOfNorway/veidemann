import {Component, Inject, OnInit} from '@angular/core';
import {PolitenessConfigDetailsComponent} from '..';
import {ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';
import {AuthService} from '../../../../../core/auth';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject} from '../../../../../shared/models/config';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatSelectModule} from '@angular/material/select';
import {DurationPickerComponent} from '../../durationpicker/duration-picker';
import {MatCheckbox} from '@angular/material/checkbox';
import {MetaComponent} from '../../meta/meta.component';
import {MatInputModule} from '@angular/material/input';
import {LayoutGapDirective} from '@ngbracket/ngx-layout/flex';
import {LayoutDirective} from '@ngbracket/ngx-layout';
import {MatButtonModule} from '@angular/material/button';

@Component({
  selector: 'app-politenessconfig-dialog',
  templateUrl: './politenessconfig-dialog.component.html',
  styleUrls: ['./politenessconfig-dialog.component.css'],
  imports: [
    DurationPickerComponent,
    LayoutDirective,
    LayoutGapDirective,
    MatButtonModule,
    MatCheckbox,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MetaComponent,
    ReactiveFormsModule
  ],
  standalone: true
})
export class PolitenessConfigDialogComponent extends PolitenessConfigDetailsComponent implements OnInit {

  constructor(protected override fb: UntypedFormBuilder,
              protected override authService: AuthService,
              @Inject(MAT_DIALOG_DATA) public data: ConfigDialogData,
              public dialogRef: MatDialogRef<PolitenessConfigDialogComponent>) {
    super(fb, authService);
    this.createForm();
    this.configObject = this.data.configObject;
    this.robotsPolicies = this.data.options.robotsPolicies;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onDialogClose(): ConfigObject {
    return this.prepareSave();
  }
}
