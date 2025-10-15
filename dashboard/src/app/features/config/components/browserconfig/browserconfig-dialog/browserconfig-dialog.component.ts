import {Component, Inject, OnInit} from '@angular/core';
import {BrowserConfigDetailsComponent} from '..';
import {ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';
import {AuthService} from '../../../../../core';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject} from '../../../../../shared/models';
import {MetaComponent} from '../../meta/meta.component';
import {MatFormFieldModule} from '@angular/material/form-field';
import {DurationPickerComponent} from '../../durationpicker/duration-picker';
import {MatSelectModule} from '@angular/material/select';
import {SelectorComponent} from '../../selector/selector.component';
import {MatInputModule} from '@angular/material/input';
import {LayoutGapDirective} from '@ngbracket/ngx-layout/flex';


@Component({
  selector: 'app-browserconfig-dialog',
  templateUrl: './browserconfig-dialog.component.html',
  styleUrls: ['./browserconfig-dialog.component.css'],
  imports: [
    MatDialogModule,
    MetaComponent,
    MatInputModule,
    MatFormFieldModule,
    DurationPickerComponent,
    ReactiveFormsModule,
    MatSelectModule,
    SelectorComponent,
    LayoutGapDirective
  ],
  standalone: true
})
export class BrowserConfigDialogComponent extends BrowserConfigDetailsComponent implements OnInit {

  constructor(protected override fb: UntypedFormBuilder,
              protected override authService: AuthService,
              @Inject(MAT_DIALOG_DATA) public data: ConfigDialogData,
              public dialogRef: MatDialogRef<BrowserConfigDialogComponent>) {
    super(fb, authService);
    this.createForm();
    this.configObject = this.data.configObject;
    this.browserScripts = this.data.options.browserScripts;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onDialogClose(): ConfigObject {
    return this.prepareSave();
  }

}
