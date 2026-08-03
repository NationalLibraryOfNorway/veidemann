import { ChangeDetectionStrategy,Component,inject,OnInit } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA,MatDialogModule,MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { LayoutGapDirective } from '@ngbracket/ngx-layout/flex';
import { BrowserConfigDetailsComponent } from '..';
import { ConfigObject } from '../../../../../shared/models';
import { ConfigDialogData } from '../../../func';
import { DurationPickerComponent } from '../../durationpicker/duration-picker';
import { MetaComponent } from '../../meta/meta.component';
import { SelectorComponent } from '../../selector/selector.component';


@Component({
  selector: 'app-browserconfig-dialog',
  templateUrl: './browserconfig-dialog.component.html',
  styleUrls: ['./browserconfig-dialog.component.css'],
  imports: [
    MatButtonModule,
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
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class BrowserConfigDialogComponent extends BrowserConfigDetailsComponent implements OnInit {
  data = inject<ConfigDialogData>(MAT_DIALOG_DATA);
  dialogRef = inject<MatDialogRef<BrowserConfigDialogComponent>>(MatDialogRef);


  constructor() {

    super();

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
