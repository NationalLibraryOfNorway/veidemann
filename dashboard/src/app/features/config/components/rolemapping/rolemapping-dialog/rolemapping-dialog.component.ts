import { ChangeDetectionStrategy,Component,inject,OnInit } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA,MatDialogModule,MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInput } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { RoleMappingDetailsComponent } from '..';
import { AuthService } from '../../../../../core/auth';
import { ConfigObject } from '../../../../../shared/models/config';
import { ConfigDialogData } from '../../../func';

@Component({
  selector: 'app-rolemapping-dialog',
  templateUrl: './rolemapping-dialog.component.html',
  styleUrls: ['./rolemapping-dialog.component.css'],
  imports: [
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInput,
    MatSelectModule,
    ReactiveFormsModule
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class RoleMappingDialogComponent extends RoleMappingDetailsComponent implements OnInit {
  protected authService = inject(AuthService);
  data = inject<ConfigDialogData>(MAT_DIALOG_DATA);
  dialogRef = inject<MatDialogRef<RoleMappingDialogComponent>>(MatDialogRef);


  constructor() {

    super();

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
