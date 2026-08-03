import { ChangeDetectionStrategy,Component,inject,OnInit } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { BrowserScriptDetailsComponent } from '..';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MAT_DIALOG_DATA,MatDialogModule,MatDialogRef } from '@angular/material/dialog';
import { MatIcon } from '@angular/material/icon';
import { MatInput } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { FlexDirective,LayoutDirective } from '@ngbracket/ngx-layout';
import { LayoutGapDirective } from '@ngbracket/ngx-layout/flex';
import { EditorComponent } from 'ngx-monaco-editor-v2';
import { ConfigObject } from '../../../../../shared/models';
import { ConfigDialogData } from '../../../func';
import { MetaComponent } from '../../meta/meta.component';


@Component({
  selector: 'app-browserscript-dialog',
  templateUrl: './browserscript-dialog.component.html',
  styleUrls: ['./browserscript-dialog.component.css'],
  imports: [
    EditorComponent,
    FlexDirective,
    LayoutDirective,
    LayoutGapDirective,
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
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class BrowserScriptDialogComponent extends BrowserScriptDetailsComponent implements OnInit {
  data = inject<ConfigDialogData>(MAT_DIALOG_DATA);
  dialogRef = inject<MatDialogRef<BrowserScriptDialogComponent>>(MatDialogRef);


  constructor() {

    super();

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
