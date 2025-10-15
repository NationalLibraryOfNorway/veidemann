import {ChangeDetectorRef, Component, Inject, OnInit, ViewChild} from '@angular/core';
import {BrowserScriptDetailsComponent} from '..';
import {AbstractControl, ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {LabelMultiComponent} from '../../label/label-multi/label-multi.component';
import { AuthService } from '../../../../../core/auth';
import {ConfigObject, Kind, Label} from '../../../../../shared/models';

@Component({
  selector: 'app-browserscript-multi-dialog',
  templateUrl: './browserscript-multi-dialog.component.html',
  styleUrls: ['./browserscript-multi-dialog.component.css'],
  imports: [
    ReactiveFormsModule,
    MatDialogModule,
    LabelMultiComponent
  ],
  standalone: true
})
export class BrowserScriptMultiDialogComponent extends BrowserScriptDetailsComponent implements OnInit {

  shouldAddLabel = undefined;
  allSelected = false;

  @ViewChild(LabelMultiComponent) labelMulti: LabelMultiComponent;

  constructor(protected override fb: UntypedFormBuilder,
              protected override authService: AuthService,
              @Inject(MAT_DIALOG_DATA) public data: ConfigDialogData,
              public dialogRef: MatDialogRef<BrowserScriptMultiDialogComponent>,
              protected override cdr: ChangeDetectorRef) {
    super(fb, authService, cdr);
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

  protected override prepareSave(): any {
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
    return this.prepareSave();
  }
}
