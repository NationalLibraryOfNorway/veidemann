import {ChangeDetectorRef, Component, Inject, OnInit} from '@angular/core';
import {BrowserScriptDetailsComponent} from '..';
import {ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';

import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {AuthService} from '../../../../../core';
import {ConfigObject} from '../../../../../shared/models';
import {MetaComponent} from '../../meta/meta.component';
import {MatSelectModule} from '@angular/material/select';
import {MatChipsModule} from '@angular/material/chips';
import {MatIcon} from '@angular/material/icon';
import {MatCardModule} from '@angular/material/card';
import {FlexLayoutModule} from '@angular/flex-layout';
import {MatInput} from '@angular/material/input';
import {MatButtonModule} from '@angular/material/button';
import {EditorComponent} from 'ngx-monaco-editor-v2';


@Component({
  selector: 'app-browserscript-dialog',
  templateUrl: './browserscript-dialog.component.html',
  styleUrls: ['./browserscript-dialog.component.css'],
  imports: [
    EditorComponent,
    FlexLayoutModule,
    MatButtonModule,
    MatCardModule,
    MatChipsModule,
    MatDialogModule,
    MatInput,
    MatIcon,
    MatSelectModule,
    MetaComponent,
    ReactiveFormsModule
  ],
  standalone: true
})
export class BrowserScriptDialogComponent extends BrowserScriptDetailsComponent implements OnInit {

  constructor(protected override fb: UntypedFormBuilder,
              protected override authService: AuthService,
              @Inject(MAT_DIALOG_DATA) public data: ConfigDialogData,
              public dialogRef: MatDialogRef<BrowserScriptDialogComponent>,
              protected override cdr: ChangeDetectorRef) {
    super(fb, authService, cdr);
    this.createForm();
    this.configObject = this.data.configObject;
    this.browserScriptTypes = this.data.options.browserScriptTypes;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onDialogClose(): ConfigObject {
    return this.prepareSave();
  }

}
