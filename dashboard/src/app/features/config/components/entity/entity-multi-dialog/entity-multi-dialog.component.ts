import { ChangeDetectionStrategy,Component,inject,OnInit,ViewChild } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA,MatDialogModule,MatDialogRef } from '@angular/material/dialog';
import { EntityDetailsComponent } from '..';
import { ConfigObject,Kind,Label } from '../../../../../shared/models/config';
import { ConfigDialogData } from '../../../func';
import { LabelMultiComponent } from '../../label/label-multi/label-multi.component';

@Component({
  selector: 'app-entity-multi-dialog',
  templateUrl: './entity-multi-dialog.component.html',
  styleUrls: ['../../mass-update-dialog.scss'],
  imports: [
    LabelMultiComponent,
    MatButtonModule,
    MatDialogModule
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class EntityMultiDialogComponent extends EntityDetailsComponent implements OnInit {
  data = inject<ConfigDialogData>(MAT_DIALOG_DATA);
  dialogRef = inject<MatDialogRef<EntityMultiDialogComponent>>(MatDialogRef);


  allSelected = false;
  shouldAddLabel = undefined;

  @ViewChild(LabelMultiComponent) labelMulti: LabelMultiComponent;

  constructor() {

    super();

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

  protected prepareMultiSave(): {updateTemplate: ConfigObject; pathList: string[]} {
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
    return this.prepareMultiSave();
  }

}
