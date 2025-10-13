import {Component, Inject, OnInit, ViewChild} from '@angular/core';
import {UntypedFormBuilder} from '@angular/forms';
import {AuthService} from '../../../../../core/auth';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {EntityDetailsComponent} from '..';
import {ConfigObject, Kind, Label} from '../../../../../shared/models/config';
import {LabelMultiComponent} from '../../label/label-multi/label-multi.component';
import {MatButtonModule} from '@angular/material/button';

@Component({
  selector: 'app-entity-multi-dialog',
  templateUrl: './entity-multi-dialog.component.html',
  imports: [
    LabelMultiComponent,
    MatButtonModule,
    MatDialogModule
  ],
  standalone: true
})
export class EntityMultiDialogComponent extends EntityDetailsComponent implements OnInit {

  allSelected = false;
  shouldAddLabel = undefined;

  @ViewChild(LabelMultiComponent) labelMulti: LabelMultiComponent;

  constructor(protected override fb: UntypedFormBuilder,
              protected override authService: AuthService,
              @Inject(MAT_DIALOG_DATA) public data: ConfigDialogData,
              public dialogRef: MatDialogRef<EntityMultiDialogComponent>) {
    super(fb, authService);
    this.configObject = this.data.configObject;
    this.allSelected = this.data.allSelected;
  }

  get labelList() {
    return this.form.get('labelList');
  }

  override get canUpdate(): boolean {
    return this.form.valid && (
      this.form.dirty || (this.shouldAddLabel !== undefined && this.labelList.value.length)
    );
  }

  override get canRevert(): boolean {
    return this.form.dirty || this.shouldAddLabel !== undefined;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  protected override createForm() {
    this.form = this.fb.group({
      labelList: {value: [], disabled: false}
    });
  }

  protected override updateForm() {
    this.form.setValue({
      labelList: this.configObject.meta.labelList
    });
    this.form.markAsPristine();
    this.form.markAsUntouched();
    if (!this.canEdit) {
      this.form.disable();
    }
  }

  protected override prepareSave(): any {
    const pathList: string[] = [];

    const formModel = this.form.value;

    const updateTemplate = new ConfigObject({
      kind: Kind.CRAWLENTITY,
    });

    if (this.labelList.value.length && this.shouldAddLabel !== undefined) {
      updateTemplate.meta.labelList = formModel.labelList;
      if (this.shouldAddLabel) {
        pathList.push('meta.label+');
      } else {
        pathList.push('meta.label-');
      }
    }

    return {updateTemplate, pathList};
  }

  override onRevert() {
    this.shouldAddLabel = undefined;
    this.labelMulti.onRevert();
    super.onRevert();
  }

  onUpdateLabels({add, labels}: { add: boolean, labels: Label[] }) {
    this.form.patchValue({
      labelList: labels
    });
    this.shouldAddLabel = add;
  }

  onDialogClose(): { updateTemplate: ConfigObject, pathList: string[] } {
    return this.prepareSave();
  }

}
