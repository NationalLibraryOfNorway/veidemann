import { ChangeDetectionStrategy,Component,inject,OnInit } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA,MatDialogModule,MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { PolitenessConfigDetailsComponent } from '..';
import { ConfigObject } from '../../../../../shared/models/config';
import { ConfigDialogData } from '../../../func';
import { DurationPickerComponent } from '../../durationpicker/duration-picker';
import { MetaComponent } from '../../meta/meta.component';
import {BooleanStateChipComponent} from '../../../../../shared/components';

@Component({
  selector: 'app-politenessconfig-dialog',
  templateUrl: './politenessconfig-dialog.component.html',
  styleUrls: ['./politenessconfig-dialog.component.css'],
  imports: [
    DurationPickerComponent,
    BooleanStateChipComponent,
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MetaComponent,
    ReactiveFormsModule
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class PolitenessConfigDialogComponent extends PolitenessConfigDetailsComponent implements OnInit {
  data = inject<ConfigDialogData>(MAT_DIALOG_DATA);
  dialogRef = inject<MatDialogRef<PolitenessConfigDialogComponent>>(MatDialogRef);


  constructor() {

    super();

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
