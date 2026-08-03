import { ChangeDetectionStrategy,Component,inject,OnInit } from '@angular/core';
import { ReactiveFormsModule,Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA,MatDialogModule,MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectModule } from '@angular/material/select';
import { RoleMappingDetailsComponent } from '..';
import { ConfigObject,Kind } from '../../../../../shared/models/config';
import { CustomValidators } from '../../../../../shared/validation';
import { ConfigDialogData } from '../../../func';

@Component({
  selector: 'app-rolemapping-multi-dialog',
  templateUrl: './rolemapping-multi-dialog.component.html',
  styleUrls: ['./rolemapping-multi-dialog.component.css'],
  imports: [
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatSelectModule,
    ReactiveFormsModule,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class RoleMappingMultiDialogComponent extends RoleMappingDetailsComponent implements OnInit {
  data = inject<ConfigDialogData>(MAT_DIALOG_DATA);
  dialogRef = inject<MatDialogRef<RoleMappingMultiDialogComponent>>(MatDialogRef);


  allSelected = false;

  constructor() {

    super();

    this.configObject = this.data.configObject;
    this.roles = this.data.options.roles;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  protected override createForm() {
    this.form = this.fb.group({
      roleList: [[], [Validators.required, CustomValidators.nonEmpty]]
    });
  }

  protected override updateForm() {
    this.form.patchValue({
      roleList: this.configObject.roleMapping.roleList,
    });
    this.form.markAsPristine();
    this.form.markAsUntouched();
  }

  protected prepareMultiSave(): {updateTemplate: ConfigObject; pathList: string[]} {
    const formModel = this.form.value;
    const pathList: string[] = [];

    const updateTemplate = new ConfigObject({kind: Kind.ROLEMAPPING});
    const roleMapping = updateTemplate.roleMapping;

    roleMapping.roleList = formModel.roleList;

    // TODO: compare array properly (not by reference)
    if (this.roleList.dirty && (this.allSelected || formModel.roleList !== this.configObject.roleMapping.roleList)) {
      roleMapping.roleList = formModel.roleList;
      pathList.push('roleMapping.role');
    }

    return {updateTemplate, pathList};
  }

  onDialogClose(): { updateTemplate: ConfigObject, pathList: string[] } {
    return this.prepareMultiSave();
  }
}
