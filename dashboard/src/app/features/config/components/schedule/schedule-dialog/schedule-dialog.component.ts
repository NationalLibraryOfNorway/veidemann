import {Component, Inject, OnInit} from '@angular/core';
import {ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';
import {AuthService} from '../../../../../core/auth';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject} from '../../../../../shared/models/config';
import {ScheduleDetailsComponent} from '..';
import {MetaComponent} from '../../meta/meta.component';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatInput} from '@angular/material/input';
import {LayoutAlignDirective, LayoutDirective} from '@ngbracket/ngx-layout';
import {MatListSubheaderCssMatStyler} from '@angular/material/list';
import {MatDatepickerModule} from '@angular/material/datepicker';
import {MatButtonModule} from '@angular/material/button';
import {LayoutGapDirective} from '@ngbracket/ngx-layout/flex';

@Component({
  selector: 'app-schedule-dialog',
  templateUrl: './schedule-dialog.component.html',
  styleUrls: ['./schedule-dialog.component.css'],
  imports: [
    LayoutAlignDirective,
    LayoutDirective,
    LayoutGapDirective,
    MatButtonModule,
    MatDatepickerModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInput,
    MatListSubheaderCssMatStyler,
    MetaComponent,
    ReactiveFormsModule
  ],
  standalone: true
})
export class ScheduleDialogComponent extends ScheduleDetailsComponent implements OnInit {

  constructor(protected override fb: UntypedFormBuilder,
              protected override authService: AuthService,
              @Inject(MAT_DIALOG_DATA) public data: ConfigDialogData,
              public dialogRef: MatDialogRef<ScheduleDialogComponent>) {
    super(fb, authService);
    this.createForm();
    this.configObject = this.data.configObject;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onDialogClose(): ConfigObject {
    return this.prepareSave();
  }
}
