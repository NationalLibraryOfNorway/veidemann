import { ChangeDetectionStrategy,Component,inject,OnInit,ViewChild } from '@angular/core';
import { AbstractControl,ReactiveFormsModule } from '@angular/forms';
import { MAT_DIALOG_DATA,MatDialogModule,MatDialogRef } from '@angular/material/dialog';
import {MatButtonModule} from '@angular/material/button';
import { BrowserScriptDetailsComponent } from '..';
import { ConfigObject,Kind,Label } from '../../../../../shared/models';
import { ConfigDialogData } from '../../../func';
import { LabelMultiComponent } from '../../label/label-multi/label-multi.component';

@Component({
  selector: 'app-browserscript-multi-dialog',
  templateUrl: './browserscript-multi-dialog.component.html',
  styleUrls: ['./browserscript-multi-dialog.component.css', '../../mass-update-dialog.scss'],
  imports: [
    ReactiveFormsModule,
    MatDialogModule,
    MatButtonModule,
    LabelMultiComponent
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class BrowserScriptMultiDialogComponent extends BrowserScriptDetailsComponent implements OnInit {
  data = inject<ConfigDialogData>(MAT_DIALOG_DATA);
  dialogRef = inject<MatDialogRef<BrowserScriptMultiDialogComponent>>(MatDialogRef);


  shouldAddLabel = undefined;
  allSelected = false;

  @ViewChild(LabelMultiComponent) labelMulti: LabelMultiComponent;

  constructor() {

    super();

    this.configObject = this.data.configObject;
    this.allSelected = this.data.allSelected;
  }

  get labelList(): AbstractControl {
    return this.form.get('labelList');
  }

  override get canUpdate(): boolean {
    return this.form.valid && (this.form.dirty || (this.shouldAddLabel !== undefined && this.labelList.value.length));
  }

  get canRevert(): boolean {
    return this.form.dirty || this.shouldAddLabel !== undefined;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onUpdateLabels({add, labels}: { add: boolean, labels: Label[] }) {
    this.form.patchValue({
      labelList: labels
    });
    this.shouldAddLabel = add;
  }

  override onRevert() {
    this.shouldAddLabel = undefined;
    this.labelMulti.onRevert();
    super.onRevert();
  }

  protected override createForm() {
    this.form = this.fb.group({
      labelList: {value: [], disabled: false}
    });
  }

  protected override updateForm(): void {
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
    const formModel = this.form.value;
    const pathList: string[] = [];
    const updateTemplate = new ConfigObject({
      kind: Kind.BROWSERSCRIPT,
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

  onDialogClose(): { updateTemplate: ConfigObject, pathList: string[] } {
    return this.prepareMultiSave();
  }
}
